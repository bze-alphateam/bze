package interchaintest_test

import (
	"context"
	"strconv"
	"testing"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/interchaintest/v10"
	"github.com/cosmos/interchaintest/v10/chain/cosmos"
	"github.com/cosmos/interchaintest/v10/ibc"
	"github.com/cosmos/interchaintest/v10/testutil"
	"github.com/stretchr/testify/require"

	"github.com/bze-alphateam/bze/interchaintest/chain"
	"github.com/bze-alphateam/bze/interchaintest/helpers"
)

// tok charges a 1% transfer fee and a 50% gas surcharge, with a community
// pool; brn has the same rates and no community pool. Both are worth 1 ubze
// in their deep pools, so their gas price is 0.01 × 1.5 = 0.015.
const (
	xfAdminFunds = 3_000_000_000_000 // 3,000,000 BZE: denoms, two pools, a market
	xfUserFunds  = 50_000_000_000    // 50,000 BZE each: create-denom fees, buy orders
	xfMint       = 1_000_000_000_000
	xfUserTok    = 300_000_000_000 // tok and brn to u1: create-denom fees paid in the token
	xfSmallTok   = 30_000          // tok to u3: gas only, never a create-denom fee
	xfGrantTok   = 100_000
	xfPoolTok    = 200_000_000_000
	xfPoolBze    = 200_000_000_000
	xfPoolDest   = `{"treasury":"0","burner":"0","providers":"1"}`
	xfRate       = "0.01"
	xfSurcharge  = "0.5"
	xfMinNative  = "0.01" // the genesis ValidatorMinGasFee, in ubze

	// big is worth 100,000 ubze a unit (its pool holds xfBigPoolBze against
	// xfBigPoolBig), so a 200,000 gas send needs a fee of 1 big:
	// ceil(0.01 / 100,000 × 1.5 × 200,000) = ceil(0.03)
	xfBigPoolBig = 2_000_000
	xfBigPoolBze = 200_000_000_000

	xfSendGas = 200_000
	xfTxGas   = 500_000

	xfGasSurchargeT = "bze.txfeecollector.GasSurchargePaidEvent"
	xfCpDest        = "txfeecollector_cp"
	xfBurnerDest    = "txfeecollector_burner"
)

// TestTransferFees proves the factory-token transfer fee and gas surcharge
// (BZE-220) on a running chain. The transfer fee is charged only on protocol
// fees paid in the token (tradebin's fee payer): plain sends and module flows
// pay nothing. It goes to the token's community pool, or to the burner
// without one. The gas surcharge is split out of the tx fee, fee grants
// included. Rate changes apply at once. Subtests run in order and share
// state.
func TestTransferFees(t *testing.T) {
	if testing.Short() {
		t.Skip("e2e: skipped in short mode")
	}
	ctx := context.Background()
	net := chain.Start(t, ctx)
	s := setupTransferFees(t, ctx, net.Chain)

	t.Run("1 sends, swaps and orders pay no transfer fee", s.noTransferFee)
	t.Run("2 gas paid in the token pays the surcharge", s.gasSurcharge)
	t.Run("3 fee grant", s.feeGrant)
	t.Run("4 create-denom fee paid in the token", s.protocolFee)
	t.Run("5 token without a community pool is burned", s.noCommunityPool)
	t.Run("6 order fee paid in the token", s.orderFee)
	t.Run("7 too little token falls back to ubze", s.fallback)
	t.Run("8 rates change at once", s.rateChanges)
	t.Run("9 a tiny gas fee still splits one unit", s.tinyGasFee)
}

type transferFeeSuite struct {
	*ammSuite

	admin, u1, u2, u3, granter ibc.Wallet
	cp                         string // tok's community pool account
	tok, brn                   string
	tokPool, brnPool           string
	tokMarket                  string
}

func setupTransferFees(t *testing.T, ctx context.Context, bze *cosmos.CosmosChain) *transferFeeSuite {
	s := &transferFeeSuite{ammSuite: &ammSuite{ctx: ctx, bze: bze}}

	s.admin = interchaintest.GetAndFundTestUsers(t, ctx, "tf-admin", sdkmath.NewInt(xfAdminFunds), bze)[0]
	users := interchaintest.GetAndFundTestUsers(t, ctx, "tf", sdkmath.NewInt(xfUserFunds), bze, bze, bze, bze, bze)
	s.u1, s.u2, s.u3, s.granter = users[0], users[1], users[2], users[3]
	s.cp = users[4].FormattedAddress()
	require.NoError(t, testutil.WaitForBlocks(ctx, 2, bze))

	for name, addr := range map[string]*string{
		"tradebin":              &s.tradebin,
		"burner":                &s.burner,
		"txfeecollector":        &s.txFees,
		"txfeecollector_burner": &s.burnerFees,
		"txfeecollector_cp":     &s.cpFees,
	} {
		var err error
		*addr, err = helpers.ModuleAddress(ctx, bze, name)
		require.NoError(t, err)
	}
	s.params = s.queryParams(t)

	factory := "factory/" + s.admin.FormattedAddress() + "/"
	s.tok, s.brn = factory+"tok", factory+"brn"
	for _, d := range []string{s.tok, s.brn} {
		s.mustExecWith(t, s.admin, []helpers.TxOption{helpers.WithGas(xfTxGas)}, "tokenfactory", "create-denom", d[len(factory):])
		s.mustExec(t, s.admin, "tokenfactory", "mint", coinStr(xfMint, d))
	}

	for _, send := range []struct {
		to    ibc.Wallet
		coins sdk.Coins
	}{
		{s.u1, sdk.NewCoins(sdk.NewInt64Coin(s.tok, xfUserTok), sdk.NewInt64Coin(s.brn, xfUserTok))},
		{s.u2, coins(1_000_000, s.tok)},
		{s.u3, coins(xfSmallTok, s.tok)},
		{s.granter, coins(xfGrantTok, s.tok)},
	} {
		s.mustExec(t, s.admin, "bank", "send", s.admin.FormattedAddress(), send.to.FormattedAddress(), send.coins.String())
	}

	s.tokPool, s.brnPool = s.tok+"_"+chain.Denom, s.brn+"_"+chain.Denom
	for _, d := range []string{s.tok, s.brn} {
		s.mustExecWith(t, s.admin, []helpers.TxOption{helpers.WithGas(xfTxGas)},
			"tradebin", "create-liquidity-pool", d, chain.Denom, ammPoolFee, xfPoolDest, "false",
			strconv.Itoa(xfPoolTok), strconv.Itoa(xfPoolBze))
	}
	s.tokMarket = s.tok + "/" + chain.Denom
	s.mustExecWith(t, s.admin, []helpers.TxOption{helpers.WithGas(xfTxGas)}, "tradebin", "create-market", s.tok, chain.Denom)

	s.setConfig(t, s.tok, s.cp, xfRate, xfSurcharge)
	s.setConfig(t, s.brn, "", xfRate, xfSurcharge)
	return s
}

// 1. With ubze paying the tx fee, a bank send, a sell swap and a sell order
// in tok take exactly the amount: the transfer fee only applies to protocol
// fees paid in the token.
func (s *transferFeeSuite) noTransferFee(t *testing.T) {
	u1, u2 := s.u1.FormattedAddress(), s.u2.FormattedAddress()

	before, recv := s.balance(t, u1, s.tok), s.balance(t, u2, s.tok)
	res := s.mustExec(t, s.u1, "bank", "send", u1, u2, coinStr(1_050, s.tok))
	s.requireNoTransferFee(t, res, u1, s.tok)
	require.Equal(t, before.SubRaw(1_050), s.balance(t, u1, s.tok))
	require.Equal(t, recv.AddRaw(1_050), s.balance(t, u2, s.tok))

	before = s.balance(t, u1, s.tok)
	res = s.mustExec(t, s.u1, "tradebin", "multi-swap", s.tokPool, coinStr(1_000_000, s.tok), coinStr(1, chain.Denom))
	s.requireNoTransferFee(t, res, u1, s.tok)
	require.Equal(t, before.SubRaw(1_000_000), s.balance(t, u1, s.tok))

	before = s.balance(t, u1, s.tok)
	res = s.mustExec(t, s.u1, "tradebin", "create-order", "sell", "1000000", "2", s.tokMarket)
	s.requireNoTransferFee(t, res, u1, s.tok)
	require.Equal(t, before.SubRaw(1_000_000), s.balance(t, u1, s.tok), "only the order amount is locked")
}

// 2. tok pays gas at the converted price × 1.5 (0.01 ubze over the pool's
// spot price, about 0.015 tok; step 1's swap moved the price, so the minimum
// is read from the reserves): one unit less than the minimum is refused at
// 200,000 gas, the minimum is accepted. The whole fee goes to the
// txfeecollector, then floor(fee × 0.5 / 1.5) moves to txfeecollector_cp.
// Gas pays no transfer fee.
func (s *transferFeeSuite) gasSurcharge(t *testing.T) {
	u1 := s.u1.FormattedAddress()
	send := func(fee int64) (helpers.TxResult, error) {
		return helpers.ExecTxWith(s.ctx, s.bze, s.u1, []helpers.TxOption{helpers.WithFees(coinStr(fee, s.tok)), helpers.WithGas(xfSendGas)},
			"bank", "send", u1, s.u2.FormattedAddress(), coinStr(1, chain.Denom))
	}
	required := s.minGasFee(t, s.tokPool, xfSendGas).Int64()
	res, err := send(required - 1)
	reason, err := helpers.RejectionReason(res, err)
	require.NoError(t, err)
	require.Contains(t, reason, "insufficient fees")
	require.Contains(t, reason, coinStr(required, s.tok))

	before := s.balance(t, u1, s.tok)
	res, err = send(required)
	require.NoError(t, err)
	require.Zero(t, res.Code, res.RawLog)
	require.Equal(t, before.SubRaw(required), s.balance(t, u1, s.tok), "only the tx fee")
	s.requireNoTransferFee(t, res, u1, s.tok)
	require.Equal(t, sdkmath.NewInt(required), s.transferred(t, res.Events, u1, s.txFees, s.tok))
	s.requireSurcharge(t, res, u1, s.tok, required/3, xfCpDest)
}

// minGasFee is the minimum fee in the pool's non-native denom for gas:
// ceil(0.01 / spot × (1 + surcharge) × gas), with spot = ubze reserve / token
// reserve, as the ante computes it.
func (s *transferFeeSuite) minGasFee(t *testing.T, poolID string, gas int64) sdkmath.Int {
	t.Helper()
	pool := s.pool(t, poolID)
	native, other := pool.reserveQuote(), pool.reserveBase()
	if pool.Base == chain.Denom {
		native, other = other, native
	}
	spot := sdkmath.LegacyNewDecFromInt(native).Quo(sdkmath.LegacyNewDecFromInt(other))
	price := sdkmath.LegacyMustNewDecFromStr(xfMinNative).Quo(spot).Mul(sdkmath.LegacyOneDec().Add(sdkmath.LegacyMustNewDecFromStr(xfSurcharge)))
	return price.MulInt64(gas).Ceil().RoundInt()
}

// 3. A fee grant pays the full fee, surcharge included, from the granter.
func (s *transferFeeSuite) feeGrant(t *testing.T) {
	granter, grantee := s.granter.FormattedAddress(), s.u2.FormattedAddress()
	s.mustExec(t, s.granter, "feegrant", "grant", granter, grantee, "--spend-limit", coinStr(xfGrantTok/2, s.tok))

	gr, ge := s.balance(t, granter, s.tok), s.balance(t, grantee, s.tok)
	res, err := helpers.ExecTxWith(s.ctx, s.bze, s.u2, []helpers.TxOption{helpers.WithFees(coinStr(4_500, s.tok)), helpers.WithGas(xfSendGas)},
		"bank", "send", grantee, s.u1.FormattedAddress(), coinStr(1, chain.Denom), "--fee-granter", granter)
	require.NoError(t, err)
	require.Zero(t, res.Code, res.RawLog)
	require.Equal(t, gr.SubRaw(4_500), s.balance(t, granter, s.tok), "the granter pays")
	require.Equal(t, ge, s.balance(t, grantee, s.tok))
	s.requireSurcharge(t, res, granter, s.tok, 1_500, xfCpDest)
}

// 4. A create-denom fee paid with the tx fee in tok is captured in tok and
// pays ceil(captured × 1%) on top to txfeecollector_cp, forwarded to the
// token's community pool in the same block's EndBlock.
func (s *transferFeeSuite) protocolFee(t *testing.T) {
	u1 := s.u1.FormattedAddress()
	cp := s.balance(t, s.cp, s.tok)

	res := s.mustExecWith(t, s.u1, []helpers.TxOption{helpers.WithFees(coinStr(20_000, s.tok)), helpers.WithGas(xfTxGas)}, "tokenfactory", "create-denom", "paidtok")
	captured := s.transferred(t, res.Events, u1, s.tradebin, s.tok)
	require.True(t, captured.IsPositive(), "the fee is captured in tok")
	fee := expectedFee(captured, xfRate)
	s.requireTransferFee(t, res, u1, s.tok, fee, xfCpDest)
	require.Len(t, helpers.TxEvents(res, xfGasSurchargeT), 1, "only the tx fee pays the surcharge")

	surcharge := s.transferred(t, res.Events, s.txFees, s.cpFees, s.tok)
	require.Equal(t, cp.AddRaw(fee).Add(surcharge), s.balance(t, s.cp, s.tok), "forwarded to the token's community pool")
}

// 5. brn has rates but no community pool: the transfer fee on a protocol fee
// paid in brn, and the gas surcharge, go to txfeecollector_burner.
func (s *transferFeeSuite) noCommunityPool(t *testing.T) {
	u1 := s.u1.FormattedAddress()
	res := s.mustExecWith(t, s.u1, []helpers.TxOption{helpers.WithFees(coinStr(20_000, s.brn)), helpers.WithGas(xfTxGas)}, "tokenfactory", "create-denom", "paidbrn")
	captured := s.transferred(t, res.Events, u1, s.tradebin, s.brn)
	require.True(t, captured.IsPositive(), "the fee is captured in brn")
	s.requireTransferFee(t, res, u1, s.brn, expectedFee(captured, xfRate), xfBurnerDest)
	s.requireSurcharge(t, res, u1, s.brn, 20_000/3, xfBurnerDest) // floor(20000 × 0.5 / 1.5)
	require.True(t, s.balance(t, s.cpFees, s.brn).IsZero())
}

// 6. The order fee of a buy order paid with the tx fee in tok pays the
// transfer fee too. The order itself locks ubze, so all the tok captured is
// the order fee.
func (s *transferFeeSuite) orderFee(t *testing.T) {
	u1 := s.u1.FormattedAddress()
	res := s.mustExecWith(t, s.u1, []helpers.TxOption{helpers.WithFees(coinStr(20_000, s.tok)), helpers.WithGas(xfTxGas)},
		"tradebin", "create-order", "buy", "1000000", "1", s.tokMarket)
	captured := s.transferred(t, res.Events, u1, s.tradebin, s.tok)
	require.True(t, captured.IsPositive(), "the order fee is captured in tok")
	s.requireTransferFee(t, res, u1, s.tok, expectedFee(captured, xfRate), xfCpDest)
}

// 7. u3 holds enough tok for gas only: the create-denom fee is taken in ubze
// and no transfer fee is charged.
func (s *transferFeeSuite) fallback(t *testing.T) {
	u3 := s.u3.FormattedAddress()
	createFee := s.createDenomFee(t)

	ubze := s.balance(t, u3, chain.Denom)
	res := s.mustExecWith(t, s.u3, []helpers.TxOption{helpers.WithFees(coinStr(20_000, s.tok)), helpers.WithGas(xfTxGas)}, "tokenfactory", "create-denom", "paidbze")
	s.requireNoTransferFee(t, res, u3, s.tok)
	require.True(t, s.transferred(t, res.Events, u3, s.tradebin, s.tok).IsZero(), "nothing captured in tok")
	require.Equal(t, ubze.Sub(createFee), s.balance(t, u3, chain.Denom), "the fee is taken in ubze")
}

// 8. Rate changes apply from the next protocol fee.
func (s *transferFeeSuite) rateChanges(t *testing.T) {
	u1 := s.u1.FormattedAddress()
	for _, rate := range []string{"0.05", "0.001", xfRate} {
		s.setConfig(t, s.tok, s.cp, rate, xfSurcharge)
		res := s.mustExecWith(t, s.u1, []helpers.TxOption{helpers.WithFees(coinStr(20_000, s.tok)), helpers.WithGas(xfTxGas)},
			"tradebin", "create-order", "buy", "1000000", "1", s.tokMarket)
		captured := s.transferred(t, res.Events, u1, s.tradebin, s.tok)
		require.True(t, captured.IsPositive())
		s.requireTransferFee(t, res, u1, s.tok, expectedFee(captured, rate), xfCpDest)
	}
}

// setConfig sets a token's DenomConfig as its admin.
// 9. A gas fee whose split floors to 0 still sends one unit to the token's
// destination when it is above 1; a fee of 1 stays whole with the validators.
// big, worth 100,000 ubze a unit with a 50% surcharge, needs a fee of 1 for a
// send: a fee of 1 splits nothing, a fee of 2 (floor(2 × 0.5 / 1.5) = 0)
// splits 1.
func (s *transferFeeSuite) tinyGasFee(t *testing.T) {
	admin, u1 := s.admin.FormattedAddress(), s.u1.FormattedAddress()
	big := "factory/" + admin + "/big"
	s.mustExecWith(t, s.admin, []helpers.TxOption{helpers.WithGas(xfTxGas)}, "tokenfactory", "create-denom", "big")
	s.mustExec(t, s.admin, "tokenfactory", "mint", coinStr(2*xfBigPoolBig, big))
	s.mustExecWith(t, s.admin, []helpers.TxOption{helpers.WithGas(xfTxGas)},
		"tradebin", "create-liquidity-pool", big, chain.Denom, ammPoolFee, xfPoolDest, "false",
		strconv.Itoa(xfBigPoolBig), strconv.Itoa(xfBigPoolBze))
	s.mustExec(t, s.admin, "bank", "send", admin, u1, coinStr(100, big))
	s.setConfig(t, big, s.cp, xfRate, xfSurcharge)
	require.Equal(t, sdkmath.OneInt(), s.minGasFee(t, big+"_"+chain.Denom, xfSendGas), "the minimum fee of a send is 1 big")

	send := func(fee int64) helpers.TxResult {
		t.Helper()
		before := s.balance(t, u1, big)
		res, err := helpers.ExecTxWith(s.ctx, s.bze, s.u1, []helpers.TxOption{helpers.WithFees(coinStr(fee, big)), helpers.WithGas(xfSendGas)},
			"bank", "send", u1, s.u2.FormattedAddress(), coinStr(1, chain.Denom))
		require.NoError(t, err)
		require.Zero(t, res.Code, res.RawLog)
		require.Equal(t, before.SubRaw(fee), s.balance(t, u1, big), "only the tx fee")
		require.Equal(t, sdkmath.NewInt(fee), s.transferred(t, res.Events, u1, s.txFees, big), "the fee to the txfeecollector")
		return res
	}

	res := send(1)
	require.Empty(t, helpers.TxEvents(res, xfGasSurchargeT), "a fee of 1 is not split")
	require.True(t, s.transferred(t, res.Events, s.txFees, s.cpFees, big).IsZero(), "nothing to txfeecollector_cp")

	res = send(2)
	s.requireSurcharge(t, res, u1, big, 1, xfCpDest)
}

func (s *transferFeeSuite) setConfig(t *testing.T, denom, cp, transferFee, gasSurcharge string) {
	t.Helper()
	s.mustExec(t, s.admin, "tokenfactory", "set-denom-config", denom, cp, transferFee, gasSurcharge)
}

// requireTransferFee asserts the payer sent exactly amount of denom to the
// destination fee collector in the tx: the transfer fee on a protocol fee.
func (s *transferFeeSuite) requireTransferFee(t *testing.T, res helpers.TxResult, payer, denom string, amount int64, destination string) {
	t.Helper()
	to := s.cpFees
	if destination == xfBurnerDest {
		to = s.burnerFees
	}
	require.Equal(t, sdkmath.NewInt(amount), s.transferred(t, res.Events, payer, to, denom), "transfer fee in tx %s", res.TxHash)
}

// requireNoTransferFee asserts the payer sent nothing to either fee collector.
func (s *transferFeeSuite) requireNoTransferFee(t *testing.T, res helpers.TxResult, payer, denom string) {
	t.Helper()
	require.True(t, s.transferred(t, res.Events, payer, s.cpFees, denom).IsZero(), "no transfer fee to txfeecollector_cp")
	require.True(t, s.transferred(t, res.Events, payer, s.burnerFees, denom).IsZero(), "no transfer fee to txfeecollector_burner")
}

// expectedFee is ceil(captured × rate), the transfer fee on a protocol fee
// captured in the token.
func expectedFee(captured sdkmath.Int, rate string) int64 {
	return sdkmath.LegacyNewDecFromInt(captured).Mul(sdkmath.LegacyMustNewDecFromStr(rate)).Ceil().TruncateInt().Int64()
}

// requireSurcharge asserts the tx's GasSurchargePaidEvent and the module
// send of the surcharge to its destination.
func (s *transferFeeSuite) requireSurcharge(t *testing.T, res helpers.TxResult, payer, denom string, amount int64, destination string) {
	t.Helper()
	events := helpers.EventsWith(helpers.TxEvents(res, xfGasSurchargeT), map[string]string{
		"payer": payer, "denom": denom, "amount": strconv.FormatInt(amount, 10), "destination": destination,
	})
	require.Len(t, events, 1, "GasSurchargePaidEvent in tx %s: %+v", res.TxHash, helpers.TxEvents(res, xfGasSurchargeT))
	to := s.cpFees
	if destination == xfBurnerDest {
		to = s.burnerFees
	}
	require.Equal(t, sdkmath.NewInt(amount), s.transferred(t, res.Events, s.txFees, to, denom), "surcharge to %s", destination)
}

// createDenomFee is tokenfactory's create_denom_fee.
func (s *transferFeeSuite) createDenomFee(t *testing.T) sdkmath.Int {
	t.Helper()
	res, err := helpers.Query[struct {
		Params struct {
			CreateDenomFee struct {
				Denom  string `json:"denom"`
				Amount string `json:"amount"`
			} `json:"create_denom_fee"`
		} `json:"params"`
	}](s.ctx, s.bze, "tokenfactory", "params")
	require.NoError(t, err)
	require.Equal(t, chain.Denom, res.Params.CreateDenomFee.Denom)
	return parseInt(res.Params.CreateDenomFee.Amount)
}
