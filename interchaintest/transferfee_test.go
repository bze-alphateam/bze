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
// pool; brn has the same rates and no community pool. tok is worth 1 ubze in
// its deep tok/ubze pool, so its gas price is 0.01 × 1.5 = 0.015 tok.
const (
	xfAdminFunds = 3_000_000_000_000 // 3,000,000 BZE: denoms, a pool, a market, a reward, a DAO, proposals
	xfUserFunds  = 50_000_000_000    // 50,000 BZE each: create-denom fees
	xfMint       = 1_000_000_000_000
	xfUserTok    = 100_000_000_000 // tok to u1, enough to pay a create-denom fee in tok
	xfSmallTok   = 30_000          // tok to u3: gas only, never a create-denom fee
	xfGrantTok   = 100_000
	xfPoolTok    = 200_000_000_000
	xfPoolBze    = 200_000_000_000
	xfPoolDest   = `{"treasury":"0","burner":"0","providers":"1"}`
	xfRate       = "0.01"
	xfSurcharge  = "0.5"

	xfSendGas = 200_000
	xfTxGas   = 500_000

	xfTransferFeeT  = "bze.txfeecollector.TransferFeePaidEvent"
	xfGasSurchargeT = "bze.txfeecollector.GasSurchargePaidEvent"
	xfCpDest        = "txfeecollector_cp"
	xfBurnerDest    = "txfeecollector_burner"
)

// TestTransferFees proves the factory-token transfer fee and gas surcharge
// (BZE-220) on a running chain: who pays the transfer fee (user senders, on
// top) and who doesn't (modules), its routing to the token's community pool,
// or to the burner without one, the gas surcharge split, fee grants, the
// protocol fee fallback, live rate changes and the per-block routing cap.
// The IBC escrow case is step 13 of TestBlockedInboundIbc. Subtests run in
// order and share state.
func TestTransferFees(t *testing.T) {
	if testing.Short() {
		t.Skip("e2e: skipped in short mode")
	}
	ctx := context.Background()
	net := chain.Start(t, ctx)
	s := setupTransferFees(t, ctx, net.Chain)

	t.Run("1 bank send and multisend", s.bankSends)
	t.Run("2 token without a community pool is burned", s.noCommunityPool)
	t.Run("3 gas paid in the token pays the surcharge only", s.gasSurcharge)
	t.Run("4 fee grant", s.feeGrant)
	t.Run("5 authz exec", s.authzExec)
	t.Run("6 swaps", s.swaps)
	t.Run("7 order place, fill and cancel", s.orders)
	t.Run("8 add and remove liquidity", s.liquidity)
	t.Run("9 burn", s.burn)
	t.Run("10 rewards stake and exit", s.staking)
	t.Run("11 DAO treasury spend", s.daoSpend)
	t.Run("12 protocol fee in the token", s.protocolFee)
	t.Run("13 rates change at once", s.rateChanges)
	t.Run("14 routing cap", s.routingCap)
}

type transferFeeSuite struct {
	*ammSuite

	admin, u1, u2, u3, granter ibc.Wallet
	cp                         string // tok's community pool account
	tok, brn                   string
	capped                     []string // step 14: three tokens with a community pool
	tokPool, tokMarket         string
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
	s.capped = []string{factory + "caa", factory + "cab", factory + "cac"}
	for _, d := range append([]string{s.tok, s.brn}, s.capped...) {
		s.mustExecWith(t, s.admin, []helpers.TxOption{helpers.WithGas(xfTxGas)}, "tokenfactory", "create-denom", d[len(factory):])
		s.mustExec(t, s.admin, "tokenfactory", "mint", coinStr(xfMint, d))
	}

	// everything is handed out before any rate is set, so nobody pays here
	for _, send := range []struct {
		to    ibc.Wallet
		coins sdk.Coins
	}{
		{s.u1, sdk.NewCoins(sdk.NewInt64Coin(s.tok, xfUserTok), sdk.NewInt64Coin(s.brn, 1_000_000),
			sdk.NewInt64Coin(s.capped[0], 1_000_000), sdk.NewInt64Coin(s.capped[1], 1_000_000), sdk.NewInt64Coin(s.capped[2], 1_000_000))},
		{s.u2, coins(1_000_000, s.tok)},
		{s.u3, coins(xfSmallTok, s.tok)},
		{s.granter, coins(xfGrantTok, s.tok)},
	} {
		s.mustExec(t, s.admin, "bank", "send", s.admin.FormattedAddress(), send.to.FormattedAddress(), send.coins.String())
	}

	s.tokPool = s.tok + "_" + chain.Denom
	s.mustExecWith(t, s.admin, []helpers.TxOption{helpers.WithGas(xfTxGas)},
		"tradebin", "create-liquidity-pool", s.tok, chain.Denom, ammPoolFee, xfPoolDest, "false",
		strconv.Itoa(xfPoolTok), strconv.Itoa(xfPoolBze))
	s.tokMarket = s.tok + "/" + chain.Denom
	s.mustExecWith(t, s.admin, []helpers.TxOption{helpers.WithGas(xfTxGas)}, "tradebin", "create-market", s.tok, chain.Denom)

	s.setConfig(t, s.tok, s.cp, xfRate, xfSurcharge)
	s.setConfig(t, s.brn, "", xfRate, xfSurcharge)
	for _, d := range s.capped {
		s.setConfig(t, d, s.cp, xfRate, "0")
	}
	return s
}

// 1. A bank send pays ceil(amount × 1%) on top: the recipient gets the full
// amount, the fee reaches txfeecollector_cp in the tx and the token's
// community pool in the same block's EndBlock. A multisend pays per output.
// A send the balance can't cover with its fee is refused, naming the fee.
func (s *transferFeeSuite) bankSends(t *testing.T) {
	u1, u2, u3 := s.u1.FormattedAddress(), s.u2.FormattedAddress(), s.u3.FormattedAddress()

	before, recv, cp := s.balance(t, u1, s.tok), s.balance(t, u2, s.tok), s.balance(t, s.cp, s.tok)
	res := s.mustExec(t, s.u1, "bank", "send", u1, u2, coinStr(1_050, s.tok))
	s.requireTransferFee(t, res, u1, s.tok, 11, xfCpDest) // ceil(10.5)
	require.Equal(t, before.SubRaw(1_061), s.balance(t, u1, s.tok), "sender pays amount + fee")
	require.Equal(t, recv.AddRaw(1_050), s.balance(t, u2, s.tok), "recipient gets the full amount")
	require.Equal(t, sdkmath.NewInt(11), s.transferred(t, s.blockEvents(t, res.Height), s.cpFees, s.cp, s.tok), "forwarded to the token's community pool")
	require.Equal(t, cp.AddRaw(11), s.balance(t, s.cp, s.tok))

	before, recv = s.balance(t, u1, s.tok), s.balance(t, u3, s.tok)
	res = s.mustExec(t, s.u1, "bank", "multi-send", u1, u2, u3, coinStr(1_000, s.tok))
	fees := helpers.TxEvents(res, xfTransferFeeT)
	require.Len(t, fees, 2, "one fee per output")
	for _, e := range fees {
		require.Equal(t, "10", e.Attributes["amount"])
	}
	require.Equal(t, before.SubRaw(2_020), s.balance(t, u1, s.tok))
	require.Equal(t, recv.AddRaw(1_000), s.balance(t, u3, s.tok))

	balance := s.balance(t, u2, s.tok)
	res, err := helpers.ExecTx(s.ctx, s.bze, s.u2, "bank", "send", u2, u1, sdk.NewCoin(s.tok, balance).String())
	reason, err := helpers.RejectionReason(res, err)
	require.NoError(t, err)
	require.Contains(t, reason, "transfer fee")
	require.Equal(t, balance, s.balance(t, u2, s.tok), "a refused send charges nothing")
}

// 2. brn has rates but no community pool: its fee goes to
// txfeecollector_burner and, unswappable, on to the burner at EndBlock.
func (s *transferFeeSuite) noCommunityPool(t *testing.T) {
	u1 := s.u1.FormattedAddress()
	res := s.mustExec(t, s.u1, "bank", "send", u1, s.u2.FormattedAddress(), coinStr(1_000, s.brn))
	s.requireTransferFee(t, res, u1, s.brn, 10, xfBurnerDest)
	require.Equal(t, sdkmath.NewInt(10), s.transferred(t, res.Events, u1, s.burnerFees, s.brn))
	require.Equal(t, sdkmath.NewInt(10), s.transferred(t, s.blockEvents(t, res.Height), s.burnerFees, s.burner, s.brn), "burned")
	require.True(t, s.balance(t, s.cpFees, s.brn).IsZero())
}

// 3. tok pays gas at 0.015 tok (0.01 converted × 1.5): 2,999 tok is refused
// at 200,000 gas, 3,000 is accepted. The whole fee goes to the
// txfeecollector, then floor(3000 × 0.5 / 1.5) = 1000 moves to
// txfeecollector_cp. The fee pays no transfer fee.
func (s *transferFeeSuite) gasSurcharge(t *testing.T) {
	u1 := s.u1.FormattedAddress()
	send := func(fee int64) (helpers.TxResult, error) {
		return helpers.ExecTxWith(s.ctx, s.bze, s.u1, []helpers.TxOption{helpers.WithFees(coinStr(fee, s.tok)), helpers.WithGas(xfSendGas)},
			"bank", "send", u1, s.u2.FormattedAddress(), coinStr(1, chain.Denom))
	}
	res, err := send(2_999)
	reason, err := helpers.RejectionReason(res, err)
	require.NoError(t, err)
	require.Contains(t, reason, "insufficient fees")
	require.Contains(t, reason, coinStr(3_000, s.tok))

	before := s.balance(t, u1, s.tok)
	res, err = send(3_000)
	require.NoError(t, err)
	require.Zero(t, res.Code, res.RawLog)
	require.Equal(t, before.SubRaw(3_000), s.balance(t, u1, s.tok), "only the tx fee, no transfer fee")
	require.Empty(t, helpers.TxEvents(res, xfTransferFeeT), "gas never pays the transfer fee")
	require.Equal(t, sdkmath.NewInt(3_000), s.transferred(t, res.Events, u1, s.txFees, s.tok))
	s.requireSurcharge(t, res, u1, 1_000)
}

// 4. A fee grant pays the full fee, surcharge included, from the granter.
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
	s.requireSurcharge(t, res, granter, 1_500)
}

// 5. A send run through authz exec is the granter's send: the granter pays.
func (s *transferFeeSuite) authzExec(t *testing.T) {
	u1, u2, u3 := s.u1.FormattedAddress(), s.u2.FormattedAddress(), s.u3.FormattedAddress()
	s.mustExec(t, s.u1, "authz", "grant", u3, "send", "--spend-limit", coinStr(10_000, s.tok))

	before, recv := s.balance(t, u1, s.tok), s.balance(t, u2, s.tok)
	res := s.mustExecMsgs(t, s.u3, map[string]any{
		"@type":   "/cosmos.authz.v1beta1.MsgExec",
		"grantee": u3,
		"msgs": []any{map[string]any{
			"@type":        "/cosmos.bank.v1beta1.MsgSend",
			"from_address": u1,
			"to_address":   u2,
			"amount":       []map[string]string{{"denom": s.tok, "amount": "2000"}},
		}},
	})
	s.requireTransferFee(t, res, u1, s.tok, 20, xfCpDest)
	require.Equal(t, before.SubRaw(2_020), s.balance(t, u1, s.tok))
	require.Equal(t, recv.AddRaw(2_000), s.balance(t, u2, s.tok))
}

// 6. Selling tok pays the fee on top and the swap runs on the exact input.
// Buying tok pays nothing: the pool sends.
func (s *transferFeeSuite) swaps(t *testing.T) {
	u1 := s.u1.FormattedAddress()
	before := s.balance(t, u1, s.tok)
	res := s.mustExec(t, s.u1, "tradebin", "multi-swap", s.tokPool, coinStr(1_000_000, s.tok), coinStr(1, chain.Denom))
	s.requireTransferFee(t, res, u1, s.tok, 10_000, xfCpDest)
	require.Equal(t, sdkmath.NewInt(1_000_000), s.transferred(t, res.Events, u1, s.tradebin, s.tok), "the swap takes the exact input")
	require.Equal(t, before.SubRaw(1_010_000), s.balance(t, u1, s.tok))

	before = s.balance(t, u1, s.tok)
	res = s.mustExec(t, s.u1, "tradebin", "multi-swap", s.tokPool, coinStr(1_000_000, chain.Denom), coinStr(1, s.tok))
	require.Empty(t, helpers.TxEvents(res, xfTransferFeeT), "a buy pays no transfer fee")
	out := s.transferred(t, res.Events, s.tradebin, u1, s.tok)
	require.True(t, out.IsPositive())
	require.Equal(t, before.Add(out), s.balance(t, u1, s.tok), "the buyer gets the whole output")
}

// 7. Placing a sell order pays the fee; the fill (the module sends) and the
// cancel refund (the full order amount) pay nothing.
func (s *transferFeeSuite) orders(t *testing.T) {
	u1, u2 := s.u1.FormattedAddress(), s.u2.FormattedAddress()

	before := s.balance(t, u1, s.tok)
	res := s.mustExec(t, s.u1, "tradebin", "create-order", "sell", "1000000", "1", s.tokMarket)
	s.requireTransferFee(t, res, u1, s.tok, 10_000, xfCpDest)
	require.Equal(t, before.SubRaw(1_010_000), s.balance(t, u1, s.tok), "order amount locked + fee")

	recv := s.balance(t, u2, s.tok)
	res = s.mustExec(t, s.u2, "tradebin", "create-order", "buy", "1000000", "1", s.tokMarket)
	require.Equal(t, sdkmath.NewInt(1_000_000), s.transferred(t, s.blockEvents(t, res.Height), s.tradebin, u2, s.tok), "the fill sends the full amount")
	require.Equal(t, recv.AddRaw(1_000_000), s.balance(t, u2, s.tok))
	s.requireNoBlockTransferFee(t, res.Height)

	before = s.balance(t, u1, s.tok)
	res = s.mustExec(t, s.u1, "tradebin", "create-order", "sell", "500000", "2", s.tokMarket)
	saved := helpers.EventsWith(filterType(s.blockEvents(t, res.Height), tradebinEventPrefix+"OrderSavedEvent"), map[string]string{"owner": u1})
	require.Len(t, saved, 1)
	require.Equal(t, before.SubRaw(505_000), s.balance(t, u1, s.tok))

	res = s.mustExec(t, s.u1, "tradebin", "cancel-order", s.tokMarket, saved[0].Attributes["id"], "sell")
	require.Equal(t, sdkmath.NewInt(500_000), s.transferred(t, s.blockEvents(t, res.Height), s.tradebin, u1, s.tok), "the cancel refunds the full amount")
	require.Equal(t, before.SubRaw(5_000), s.balance(t, u1, s.tok), "only the placement fee is gone")
	s.requireNoBlockTransferFee(t, res.Height)
}

// 8. Adding liquidity pays the fee on the tok deposited; removing pays none.
func (s *transferFeeSuite) liquidity(t *testing.T) {
	u1 := s.u1.FormattedAddress()
	pool := s.pool(t, s.tokPool)
	lpBefore := s.balance(t, u1, pool.LpDenom)
	before := s.balance(t, u1, s.tok)

	res := s.mustExec(t, s.u1, "tradebin", "add-liquidity", s.tokPool, "1000000", "1000000", "1")
	deposited := s.transferred(t, res.Events, u1, s.tradebin, s.tok)
	fee := sdkmath.LegacyNewDecFromInt(deposited).Mul(sdkmath.LegacyMustNewDecFromStr(xfRate)).Ceil().TruncateInt()
	s.requireTransferFee(t, res, u1, s.tok, fee.Int64(), xfCpDest)
	require.Equal(t, before.Sub(deposited).Sub(fee), s.balance(t, u1, s.tok))

	minted := s.balance(t, u1, pool.LpDenom).Sub(lpBefore)
	before = s.balance(t, u1, s.tok)
	res = s.mustExec(t, s.u1, "tradebin", "remove-liquidity", s.tokPool, minted.String(), "1", "1")
	require.Empty(t, helpers.TxEvents(res, xfTransferFeeT), "removing liquidity pays no fee")
	require.Equal(t, before.Add(s.transferred(t, res.Events, s.tradebin, u1, s.tok)), s.balance(t, u1, s.tok))
}

// 9. Burning pays the fee: 1,000 burned, 10 to the community pool.
func (s *transferFeeSuite) burn(t *testing.T) {
	admin := s.admin.FormattedAddress()
	before, supply := s.balance(t, admin, s.tok), s.supply(t, s.tok)
	res := s.mustExec(t, s.admin, "tokenfactory", "burn", coinStr(1_000, s.tok))
	s.requireTransferFee(t, res, admin, s.tok, 10, xfCpDest)
	require.Equal(t, before.SubRaw(1_010), s.balance(t, admin, s.tok))
	require.Equal(t, supply.SubRaw(1_000), s.supply(t, s.tok), "exactly the amount is burned")
}

// 10. Joining a staking reward on tok pays the fee; exiting it (lock 0, the
// stake returns in the exit tx) pays none.
func (s *transferFeeSuite) staking(t *testing.T) {
	u1 := s.u1.FormattedAddress()
	res := s.mustExec(t, s.admin, "rewards", "create-staking-reward", "1000", chain.Denom, s.tok, "100", "1", "0")
	created := helpers.TxEvents(res, "bze.rewards.StakingRewardCreateEvent")
	require.Len(t, created, 1)
	id := created[0].Attributes["reward_id"]

	before := s.balance(t, u1, s.tok)
	res = s.mustExec(t, s.u1, "rewards", "join-staking", id, "1000000")
	s.requireTransferFee(t, res, u1, s.tok, 10_000, xfCpDest)
	require.Equal(t, before.SubRaw(1_010_000), s.balance(t, u1, s.tok))

	res = s.mustExec(t, s.u1, "rewards", "exit-staking", id)
	require.Empty(t, helpers.TxEvents(res, xfTransferFeeT), "unstaking pays no fee")
	require.Equal(t, before.SubRaw(10_000), s.balance(t, u1, s.tok), "the whole stake is back")
}

// 11. A DAO treasury is a plain account: its admin-executed spend pays
// amount + fee from the treasury.
func (s *transferFeeSuite) daoSpend(t *testing.T) {
	u1, u3 := s.u1.FormattedAddress(), s.u3.FormattedAddress()
	res := s.mustExecWith(t, s.admin, []helpers.TxOption{helpers.WithGas(xfTxGas)},
		"dao", "create-dao", "static-list", "Fee DAO", defaultCfg().cli(), "--members", s.u2.FormattedAddress(), "--admin", u1)
	created := daoEvents(res, "EventDaoCreated")
	require.Len(t, created, 1)
	treasury := created[0].Attributes["address"]
	s.mustExec(t, s.u1, "bank", "send", u1, treasury, coinStr(1_000_000, s.tok))

	before, recv := s.balance(t, treasury, s.tok), s.balance(t, u3, s.tok)
	res = s.mustExecWith(t, s.u1, []helpers.TxOption{helpers.WithGas(xfTxGas)}, "dao", "admin-execute", idOf(created[0]),
		mustJSON(map[string]any{
			"@type":        "/cosmos.bank.v1beta1.MsgSend",
			"from_address": treasury,
			"to_address":   u3,
			"amount":       []map[string]string{{"denom": s.tok, "amount": "100000"}},
		}))
	s.requireTransferFee(t, res, treasury, s.tok, 1_000, xfCpDest)
	require.Equal(t, before.SubRaw(101_000), s.balance(t, treasury, s.tok))
	require.Equal(t, recv.AddRaw(100_000), s.balance(t, u3, s.tok))
}

// 12. A create-denom fee paid with the tx fee in tok is captured in tok and
// pays the transfer fee on that capture (not the surcharge: only the tx fee
// pays that). With too little tok it is taken in ubze instead.
func (s *transferFeeSuite) protocolFee(t *testing.T) {
	u1, u3 := s.u1.FormattedAddress(), s.u3.FormattedAddress()
	createFee := s.createDenomFee(t)

	res := s.mustExecWith(t, s.u1, []helpers.TxOption{helpers.WithFees(coinStr(20_000, s.tok)), helpers.WithGas(xfTxGas)}, "tokenfactory", "create-denom", "paidtok")
	captured := s.transferred(t, res.Events, u1, s.tradebin, s.tok)
	require.True(t, captured.IsPositive(), "the fee is captured in tok")
	fee := sdkmath.LegacyNewDecFromInt(captured).Mul(sdkmath.LegacyMustNewDecFromStr(xfRate)).Ceil().TruncateInt()
	s.requireTransferFee(t, res, u1, s.tok, fee.Int64(), xfCpDest)
	require.Len(t, helpers.TxEvents(res, xfGasSurchargeT), 1, "only the tx fee pays the surcharge")

	ubze := s.balance(t, u3, chain.Denom)
	res = s.mustExecWith(t, s.u3, []helpers.TxOption{helpers.WithFees(coinStr(20_000, s.tok)), helpers.WithGas(xfTxGas)}, "tokenfactory", "create-denom", "paidbze")
	require.Empty(t, helpers.TxEvents(res, xfTransferFeeT), "nothing captured in tok")
	require.Equal(t, ubze.Sub(createFee), s.balance(t, u3, chain.Denom), "the fee is taken in ubze")
}

// 13. Rate changes apply from the next send.
func (s *transferFeeSuite) rateChanges(t *testing.T) {
	u1, u2 := s.u1.FormattedAddress(), s.u2.FormattedAddress()
	for _, c := range []struct {
		rate string
		fee  int64
	}{{"0.05", 50}, {"0.001", 1}, {xfRate, 10}} {
		s.setConfig(t, s.tok, s.cp, c.rate, xfSurcharge)
		res := s.mustExec(t, s.u1, "bank", "send", u1, u2, coinStr(1_000, s.tok))
		s.requireTransferFee(t, res, u1, s.tok, c.fee, xfCpDest)
	}
}

// 14. With max_balance_iterations at 2, a send of three fee-bearing tokens
// leaves three coins in txfeecollector_cp: two are forwarded in that block's
// EndBlock, the third in the next one.
func (s *transferFeeSuite) routingCap(t *testing.T) {
	gov, err := helpers.GovAuthority(s.ctx, s.bze)
	require.NoError(t, err)
	setIterations := func(n int) {
		_, err := helpers.SubmitAndPassProposalWith(s.ctx, s.bze, s.admin, []helpers.TxOption{helpers.WithGas(xfTxGas)}, helpers.Proposal{
			Title: "txfeecollector iterations", Summary: "e2e: routing cap",
			Messages: []any{map[string]any{
				"@type":     "/bze.txfeecollector.MsgUpdateParams",
				"authority": gov,
				"params": map[string]any{
					"ValidatorMinGasFee":   map[string]string{"denom": chain.Denom, "amount": "0.01"},
					"MaxBalanceIterations": strconv.Itoa(n),
					"BlockedIbcInbound":    []any{},
				},
			}},
		})
		require.NoError(t, err)
	}
	setIterations(2)
	defer setIterations(100)

	u1 := s.u1.FormattedAddress()
	amt := sdk.NewCoins(sdk.NewInt64Coin(s.capped[0], 1_000), sdk.NewInt64Coin(s.capped[1], 1_000), sdk.NewInt64Coin(s.capped[2], 1_000))
	res := s.mustExec(t, s.u1, "bank", "send", u1, s.u2.FormattedAddress(), amt.String())
	require.Len(t, helpers.TxEvents(res, xfTransferFeeT), 3)

	first, err := helpers.BlockEvents(s.ctx, s.bze, res.Height, communityPoolPaidT)
	require.NoError(t, err)
	require.Len(t, first, 2, "two coins routed in the send's block")
	require.NoError(t, testutil.WaitForBlocks(s.ctx, 1, s.bze))
	second, err := helpers.BlockEvents(s.ctx, s.bze, res.Height+1, communityPoolPaidT)
	require.NoError(t, err)
	require.Len(t, second, 1, "the third in the next block")
	require.Equal(t, s.capped[2], second[0].Attributes["denom"])
	for _, d := range s.capped {
		require.True(t, s.balance(t, s.cpFees, d).IsZero(), "%s fully routed", d)
	}
}

// setConfig sets a token's DenomConfig as its admin.
func (s *transferFeeSuite) setConfig(t *testing.T, denom, cp, transferFee, gasSurcharge string) {
	t.Helper()
	s.mustExec(t, s.admin, "tokenfactory", "set-denom-config", denom, cp, transferFee, gasSurcharge)
}

// requireTransferFee asserts exactly one TransferFeePaidEvent in the tx and
// the matching bank transfer from the payer to the destination.
func (s *transferFeeSuite) requireTransferFee(t *testing.T, res helpers.TxResult, payer, denom string, amount int64, destination string) {
	t.Helper()
	events := helpers.TxEvents(res, xfTransferFeeT)
	require.Len(t, events, 1, "want one TransferFeePaidEvent in tx %s, got %+v", res.TxHash, events)
	require.Equal(t, map[string]string{"payer": payer, "denom": denom, "amount": strconv.FormatInt(amount, 10), "destination": destination},
		map[string]string{"payer": events[0].Attributes["payer"], "denom": events[0].Attributes["denom"], "amount": events[0].Attributes["amount"], "destination": events[0].Attributes["destination"]})
	to := s.cpFees
	if destination == xfBurnerDest {
		to = s.burnerFees
	}
	require.Equal(t, sdkmath.NewInt(amount), s.transferred(t, res.Events, payer, to, denom), "fee transfer")
}

// requireSurcharge asserts the tx's GasSurchargePaidEvent and the module
// send of the surcharge to txfeecollector_cp.
func (s *transferFeeSuite) requireSurcharge(t *testing.T, res helpers.TxResult, payer string, amount int64) {
	t.Helper()
	events := helpers.EventsWith(helpers.TxEvents(res, xfGasSurchargeT), map[string]string{
		"payer": payer, "denom": s.tok, "amount": strconv.FormatInt(amount, 10), "destination": xfCpDest,
	})
	require.Len(t, events, 1, "GasSurchargePaidEvent in tx %s: %+v", res.TxHash, helpers.TxEvents(res, xfGasSurchargeT))
	require.Equal(t, sdkmath.NewInt(amount), s.transferred(t, res.Events, s.txFees, s.cpFees, s.tok), "surcharge to txfeecollector_cp")
}

// requireNoBlockTransferFee asserts the EndBlock at height charged nobody.
func (s *transferFeeSuite) requireNoBlockTransferFee(t *testing.T, height int64) {
	t.Helper()
	events, err := helpers.BlockEvents(s.ctx, s.bze, height, xfTransferFeeT)
	require.NoError(t, err)
	require.Empty(t, events, "the EndBlock at %d charged a transfer fee", height)
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
