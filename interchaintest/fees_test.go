package interchaintest_test

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
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

// The network runs its nodes with minimum-gas-prices 0.001ubze, a tenth of
// the txfeecollector ValidatorMinGasFee param (0.01ubze), so every minimum
// the suite asserts is the param floor, not the node's own setting.
//
// Pool prices: gld and pul are worth 1000 ubze each, so a fee of a few gld
// pays for a tx but is too small to swap (the 0.2% pool fee of anything under
// 500 units rounds to zero); tre is worth 1 ubze. nsw has no ubze pool at all.
// The gld, pul and tre pools are deep (ubze side above the tradebin
// MinNativeLiquidityForModuleSwap, 1e11) until step 6 pulls the pul liquidity.
const (
	feAdminFunds      = 2_000_000_000_000 // 2,000,000 BZE: five denoms, four pools, a market, liquidity, proposals
	feUserFunds       = 10_000_000_000    // 10,000 BZE each
	fePoorFunds       = 1_000             // less than any fee the suite pays
	feNodeMinGasPrice = "0.001"
	feNodeMinGas      = feNodeMinGasPrice + chain.Denom

	feMint = 1_000_000_000_000 // of each factory denom, to the admin

	feProvidersDest = `{"treasury":"0","burner":"0","providers":"1"}`
	feTreasuryDest  = `{"treasury":"1","burner":"0","providers":"0"}`

	feGldPoolGld = 200_000_000
	feGldPoolBze = 200_000_000_000
	feTrePoolTre = 200_000_000_000
	feTrePoolBze = 200_000_000_000
	// pul starts shallow and step setup deepens it; step 6 removes the deposit
	fePulPoolPul    = 50_000_000
	fePulPoolBze    = 50_000_000_000
	fePulDepositPul = 75_000_000
	fePulDepositBze = 75_000_000_000
	// nsw/gld: no ubze side, swap fees all to the treasury (txfeecollector_cp)
	feNswPoolGld = 10_000_000
	feNswPoolNsw = 10_000_000_000

	feUserTre = 100_000_000
	feUserNsw = 100_000_000
	feUserGld = 10_000_000
	feUserPul = 1_000_000
	// step 11: thr is worth 1000 ubze; its pool's ubze side is exactly
	// MinNativeLiquidityForModuleSwap
	feUserThr    = 1_000_000
	feThrPoolThr = 100_000_000
	feThrFee     = 10

	feSendGas   = 200_000
	feTxGas     = 500_000
	feMinGasGas = 2_000_000 // step 8: a larger limit so the gld minimum is a few units

	feNativeFee  = 5_000 // ubze, above the 2000 minimum at feSendGas
	feGldTxFee   = 1_000 // gld, large enough to be swapped on its own
	feSmallGld   = 10    // gld, too small to swap
	feGrowGld    = 600   // gld, makes feSmallGld swappable
	fePulFee     = 10    // pul, too small to swap
	feTreSwap    = 1_000_000
	feNswSwap    = 1_000_000
	feTreFee     = 5_000 // tre, swappable
	feMakerSell  = 10_000
	feTakerBuy   = 1_000
	feOrderPrice = "1"

	feGrantUbze = 10_000
	feGrantGld  = 1_000
	feGrantFee  = 2_000

	// step 13 proposals pay above the raised minimum
	feGovPrices = "0.03" + chain.Denom
)

// TestFees follows every fee to its final account and proves the minimum gas
// price: native fees to the validators through distribution, the
// community-pool leg (create fees and treasury swap fees, swapped or
// forwarded to the burner), the burner leg of a taker fee captured in a
// non-native denom, non-native tx fees that wait until they can be swapped or
// are forwarded as-is once their pool is gone, the ubze and non-native minimum
// gas prices, the ante refusals, simulation, fee grants and the
// txfeecollector params by gov. Subtests run in order and share state.
func TestFees(t *testing.T) {
	if testing.Short() {
		t.Skip("e2e: skipped in short mode")
	}
	ctx := context.Background()
	net := chain.Start(t, ctx, chain.WithMinGasPrices(feNodeMinGas))
	s := setupFees(t, ctx, net.Chain)

	t.Run("1 native fee to the validators", s.nativeFee)
	t.Run("2 community-pool leg", s.communityPoolLeg)
	t.Run("3 non-native coins in the community-pool collector", s.communityPoolNonNative)
	t.Run("4 burner leg in a non-native denom", s.burnerLeg)
	t.Run("5 too small to swap", s.tooSmallToSwap)
	t.Run("6 non-swappable in the txfeecollector", s.nonSwappable)
	t.Run("7 minimum gas price in ubze", s.minGasNative)
	t.Run("8 minimum gas price in a non-native denom", s.minGasNonNative)
	t.Run("9 ante refusals", s.anteRefusals)
	t.Run("10 simulation", s.simulation)
	t.Run("11 threshold boundary", s.thresholdBoundary)
	t.Run("12 fee grants", s.feeGrants)
	t.Run("13 params by gov", s.paramsByGov)
}

type feesSuite struct {
	*ammSuite

	admin, u1, taker, grantee, granter, poor ibc.Wallet
	gld, tre, pul, nsw, tok                  string
	gldPool, trePool, pulPool, nswPool       string
	tokMarket                                string

	gov, valoper             string
	distr, mintAcc, tfAcc    string
	tax, commissionRate      sdkmath.LegacyDec
	poolFee, minNative       sdkmath.LegacyDec
	minLiquidity             sdkmath.Int
	paramsBefore             feTxfeeParams
	createDenomFee           sdk.Coin
	pulDepositLp, pulLpDenom string
}

func setupFees(t *testing.T, ctx context.Context, bze *cosmos.CosmosChain) *feesSuite {
	s := &feesSuite{ammSuite: &ammSuite{ctx: ctx, bze: bze}, poolFee: sdkmath.LegacyMustNewDecFromStr(ammPoolFee)}

	s.admin = interchaintest.GetAndFundTestUsers(t, ctx, "fe-admin", sdkmath.NewInt(feAdminFunds), bze)[0]
	users := interchaintest.GetAndFundTestUsers(t, ctx, "fe", sdkmath.NewInt(feUserFunds), bze, bze, bze, bze)
	s.u1, s.taker, s.grantee, s.granter = users[0], users[1], users[2], users[3]
	s.poor = interchaintest.GetAndFundTestUsers(t, ctx, "fe-poor", sdkmath.NewInt(fePoorFunds), bze)[0]
	require.NoError(t, testutil.WaitForBlocks(ctx, 2, bze))

	for name, addr := range map[string]*string{
		"tradebin":              &s.tradebin,
		"burner":                &s.burner,
		"burner_black_hole":     &s.blackHole,
		"fee_collector":         &s.feeCollector,
		"txfeecollector":        &s.txFees,
		"txfeecollector_burner": &s.burnerFees,
		"txfeecollector_cp":     &s.cpFees,
		"distribution":          &s.distr,
		"mint":                  &s.mintAcc,
		"tokenfactory":          &s.tfAcc,
	} {
		var err error
		*addr, err = helpers.ModuleAddress(ctx, bze, name)
		require.NoError(t, err)
	}
	var err error
	s.gov, err = helpers.GovAuthority(ctx, bze)
	require.NoError(t, err)

	s.params = s.queryParams(t)
	s.paramsBefore = s.txfeeParams(t)
	s.minNative = s.paramsBefore.minGasPrice(t)
	require.Equal(t, sdkmath.LegacyMustNewDecFromStr("0.01"), s.minNative, "ValidatorMinGasFee")
	require.EqualValues(t, 100, s.paramsBefore.MaxBalanceIterations, "MaxBalanceIterations")
	s.minLiquidity = s.minNativeLiquidity(t)
	s.createDenomFee = s.tokenfactoryFee(t)
	s.valoper, s.commissionRate = s.validator(t)
	s.tax = s.communityTax(t)
	t.Logf("community tax %s, commission rate %s, min native liquidity %s", s.tax, s.commissionRate, s.minLiquidity)

	factory := "factory/" + s.admin.FormattedAddress() + "/"
	s.gld, s.tre, s.pul, s.nsw, s.tok = factory+"gld", factory+"tre", factory+"pul", factory+"nsw", factory+"tok"
	for _, d := range []string{s.gld, s.tre, s.pul, s.nsw} {
		s.mustExec(t, s.admin, "tokenfactory", "create-denom", strings.TrimPrefix(d, factory))
		s.mustExec(t, s.admin, "tokenfactory", "mint", coinStr(feMint, d))
	}
	for _, send := range []struct {
		to    ibc.Wallet
		coins sdk.Coins
	}{
		{s.u1, sdk.NewCoins(sdk.NewInt64Coin(s.tre, feUserTre), sdk.NewInt64Coin(s.nsw, feUserNsw), sdk.NewInt64Coin(s.gld, feUserGld), sdk.NewInt64Coin(s.pul, feUserPul))},
		{s.taker, coins(feUserGld, s.gld)},
		{s.grantee, coins(feUserGld, s.gld)},
		{s.granter, coins(feUserGld, s.gld)},
	} {
		s.mustExec(t, s.admin, "bank", "send", s.admin.FormattedAddress(), send.to.FormattedAddress(), send.coins.String())
	}

	// pool ids order the denoms alphabetically: "factory/…" sorts before
	// "ubze", and ".../gld" before ".../nsw"
	s.gldPool, s.trePool, s.pulPool = s.gld+"_"+chain.Denom, s.tre+"_"+chain.Denom, s.pul+"_"+chain.Denom
	s.nswPool = s.gld + "_" + s.nsw
	s.createPool(t, s.gld, chain.Denom, feProvidersDest, feGldPoolGld, feGldPoolBze)
	s.createPool(t, s.tre, chain.Denom, feTreasuryDest, feTrePoolTre, feTrePoolBze)
	s.createPool(t, s.pul, chain.Denom, feProvidersDest, fePulPoolPul, fePulPoolBze)

	// deepen the pul pool; step 6 removes exactly this deposit again
	pul := s.pool(t, s.pulPool)
	s.pulLpDenom = pul.LpDenom
	before := s.balance(t, s.admin.FormattedAddress(), s.pulLpDenom)
	s.mustExec(t, s.admin, "tradebin", "add-liquidity", s.pulPool, strconv.Itoa(fePulDepositPul), strconv.Itoa(fePulDepositBze), "1")
	s.pulDepositLp = s.balance(t, s.admin.FormattedAddress(), s.pulLpDenom).Sub(before).String()
	pul = s.pool(t, s.pulPool)
	require.True(t, pul.reserveQuote().GT(s.minLiquidity), "pul pool deep: %s", pul.reserveQuote())
	return s
}

// 1. A ubze tx fee lands in the fee collector and stays there to the end of
// its block. The next BeginBlock adds the block's inflation and distribution
// takes all of it: the validator's outstanding rewards grow by the
// non-taxed share (its commission by its rate of that) and the community
// pool by the community tax.
func (s *feesSuite) nativeFee(t *testing.T) {
	require.True(t, s.addrBalances(t, s.txFees, 0).IsZero(), "txfeecollector empty")
	fee := coins(feNativeFee, chain.Denom)
	res := s.mustExecWith(t, s.u1, []helpers.TxOption{helpers.WithFees(fee.String()), helpers.WithGas(feSendGas)},
		"bank", "send", s.u1.FormattedAddress(), s.admin.FormattedAddress(), coinStr(1, chain.Denom))

	require.Equal(t, fee, s.transfers(t, res.Events, s.u1.FormattedAddress(), s.feeCollector), "the fee goes to the fee collector")
	require.Equal(t, s.u1.FormattedAddress(), s.feePayer(t, res), "fee payer")
	require.Equal(t, fee, s.addrBalances(t, s.feeCollector, res.Height), "fee collector at the end of the tx's block")

	next := res.Height + 1
	s.waitForHeight(t, next)
	minted := s.transfers(t, s.blockEvents(t, next), s.mintAcc, s.feeCollector)
	require.True(t, minted.AmountOf(chain.Denom).IsPositive(), "inflation minted at %d", next)
	alloc := s.requireAllocation(t, next, nil)
	require.Equal(t, fee.Add(minted...), alloc, "distribution takes the fee and the block's inflation")
	require.True(t, s.addrBalances(t, s.feeCollector, next).IsZero(), "fee collector emptied")
}

// 2. The tokenfactory create-denom fee and the tradebin create-pool fee go
// to the community-pool collector, which funds the community pool with them
// in the same block's EndBlock.
func (s *feesSuite) communityPoolLeg(t *testing.T) {
	require.True(t, s.addrBalances(t, s.cpFees, 0).IsZero(), "community-pool collector empty")

	res := s.mustExec(t, s.admin, "tokenfactory", "create-denom", "tok")
	fee := sdk.NewCoins(s.createDenomFee)
	require.Equal(t, fee, s.transfers(t, res.Events, s.tfAcc, s.cpFees), "create-denom fee to the community-pool collector")
	s.requireFunded(t, res.Height, fee)

	res = s.mustExecWith(t, s.admin, []helpers.TxOption{helpers.WithGas(feTxGas)},
		"tradebin", "create-liquidity-pool", s.gld, s.nsw, ammPoolFee, feTreasuryDest, "false",
		strconv.Itoa(feNswPoolGld), strconv.Itoa(feNswPoolNsw))
	fee = sdk.NewCoins(s.params.CreateMarketFee)
	require.Equal(t, fee, s.transfers(t, res.Events, s.tradebin, s.cpFees), "create-pool fee to the community-pool collector")
	s.requireFunded(t, res.Height, fee)
	require.Equal(t, s.gld, s.pool(t, s.nswPool).Base)
}

// 3. A swap on a treasury pool sends its fee, in the input denom, to the
// community-pool collector. tre has a deep ubze pool: the EndBlock swaps it
// and funds the community pool with the ubze, and the treasury share of that
// swap's own fee comes back to the collector, too small to swap. nsw has no
// ubze pool: its fee is forwarded to the burner.
func (s *feesSuite) communityPoolNonNative(t *testing.T) {
	t.Run("swappable", func(t *testing.T) {
		pool := s.pool(t, s.trePool)
		in := sdkmath.NewInt(feTreSwap)
		q := s.quote(in, pool.reserveBase(), pool.reserveQuote())
		res := s.mustExec(t, s.u1, "tradebin", "multi-swap", s.trePool, sdk.NewCoin(s.tre, in).String(), coinStr(1, chain.Denom))
		require.Equal(t, sdk.NewCoins(sdk.NewCoin(s.tre, q.fee)), s.transfers(t, res.Events, s.tradebin, s.cpFees), "swap fee in tre to the collector")

		// the user's swap left tre + (in - fee), ubze - out
		treR, bzeR := pool.reserveBase().Add(in).Sub(q.fee), pool.reserveQuote().Sub(q.out)
		cq := s.quote(q.fee, treR, bzeR)
		require.True(t, cq.fee.IsPositive() && cq.fee.LT(sdkmath.NewInt(500)), "the collector's own swap fee is positive and too small to swap: %s", cq.fee)

		block := s.blockEvents(t, res.Height)
		s.requireSwap(t, block, s.cpFees, s.trePool, sdk.NewCoin(s.tre, q.fee), sdk.NewCoin(chain.Denom, cq.out))
		require.Equal(t, sdk.NewCoins(sdk.NewCoin(s.tre, cq.fee), sdk.NewCoin(chain.Denom, cq.out)), s.transfers(t, block, s.tradebin, s.cpFees), "treasury share and swap output back to the collector")
		require.Equal(t, sdk.NewCoins(sdk.NewCoin(s.tre, cq.fee)), s.addrBalances(t, s.cpFees, res.Height), "the treasury share waits in the collector")
		s.requireFunded(t, res.Height, sdk.NewCoins(sdk.NewCoin(chain.Denom, cq.out)))

		after := s.pool(t, s.trePool)
		require.Equal(t, treR.Add(q.fee).Sub(cq.fee), after.reserveBase(), "tre reserve")
		require.Equal(t, bzeR.Sub(cq.out), after.reserveQuote(), "ubze reserve")
	})

	t.Run("not swappable", func(t *testing.T) {
		left := s.addrBalances(t, s.cpFees, 0)
		pool := s.pool(t, s.nswPool)
		in := sdkmath.NewInt(feNswSwap)
		q := s.quote(in, pool.reserveQuote(), pool.reserveBase())
		res := s.mustExec(t, s.u1, "tradebin", "multi-swap", s.nswPool, sdk.NewCoin(s.nsw, in).String(), coinStr(1, s.gld))
		fee := sdk.NewCoins(sdk.NewCoin(s.nsw, q.fee))
		require.Equal(t, fee, s.transfers(t, res.Events, s.tradebin, s.cpFees), "swap fee in nsw to the collector")

		block := s.blockEvents(t, res.Height)
		require.Equal(t, fee, s.transfers(t, block, s.cpFees, s.burner), "nsw has no ubze pool: forwarded to the burner")
		require.Empty(t, s.swapsBy(block, s.cpFees), "no swap by the collector")
		require.True(t, s.transfers(t, block, s.cpFees, s.distr).IsZero(), "nothing funded")
		require.Equal(t, left, s.addrBalances(t, s.cpFees, res.Height), "only the waiting tre is left")
	})
}

// 4. A taker paying its tx fee in gld pays the 0.1 BZE taker fee in gld too.
// The gld that buys the taker fee is about 101 units, whose swap fee rounds
// to zero, so the chain captures it as-is for the burner fee collector, which
// keeps it until enough accumulated to swap (500 units): the fifth taker's
// EndBlock swaps all of it and the burner gets the ubze. Each taker's tx fee
// (1000 gld) is swapped by the txfeecollector first, in the same EndBlock.
func (s *feesSuite) burnerLeg(t *testing.T) {
	s.tokMarket = s.tok + "/" + chain.Denom
	s.mustExec(t, s.admin, "tokenfactory", "mint", coinStr(feMint, s.tok))
	s.mustExecWith(t, s.admin, []helpers.TxOption{helpers.WithGas(feTxGas)}, "tradebin", "create-market", s.tok, chain.Denom)
	s.mustExec(t, s.admin, "tradebin", "create-order", "sell", strconv.Itoa(feMakerSell), feOrderPrice, s.tokMarket)
	require.True(t, s.addrBalances(t, s.burnerFees, 0).IsZero(), "burner fee collector empty")
	require.True(t, s.addrBalances(t, s.txFees, 0).IsZero(), "txfeecollector empty")

	takerFee := s.params.MarketTakerFee
	acc := sdkmath.ZeroInt()
	for round := 1; ; round++ {
		require.LessOrEqual(t, round, 10, "the burner fee collector never reached a swappable amount")
		pool := s.pool(t, s.gldPool)
		req := s.inputFor(takerFee.Amount, pool.reserveBase(), pool.reserveQuote())
		require.True(t, s.swapFee(req).IsZero(), "the taker fee in gld (%s) is too small to swap", req)
		before := s.addrBalances(t, s.taker.FormattedAddress(), 0)

		res := s.mustExecWith(t, s.taker, []helpers.TxOption{helpers.WithFees(coinStr(feGldTxFee, s.gld)), helpers.WithGas(feTxGas)},
			"tradebin", "create-order", "buy", strconv.Itoa(feTakerBuy), feOrderPrice, s.tokMarket)
		require.Equal(t, sdk.NewCoins(sdk.NewCoin(s.gld, req)), s.transfers(t, res.Events, s.tradebin, s.burnerFees), "round %d: taker fee captured as-is in gld", round)
		acc = acc.Add(req)

		cost := sdkmath.NewInt(feTakerBuy) // price 1
		want := before.Add(sdk.NewInt64Coin(s.tok, feTakerBuy)).Sub(sdk.NewCoin(s.gld, req.AddRaw(feGldTxFee)), sdk.NewCoin(chain.Denom, cost))
		require.Equal(t, want, s.addrBalances(t, s.taker.FormattedAddress(), 0), "round %d: taker pays no ubze fee", round)

		// EndBlock: the txfeecollector swaps the tx fee first
		block := s.blockEvents(t, res.Height)
		txq := s.quote(sdkmath.NewInt(feGldTxFee), pool.reserveBase(), pool.reserveQuote())
		s.requireSwap(t, block, s.txFees, s.gldPool, sdk.NewInt64Coin(s.gld, feGldTxFee), sdk.NewCoin(chain.Denom, txq.out))
		require.Equal(t, sdk.NewCoins(sdk.NewCoin(chain.Denom, txq.out)), s.transfers(t, block, s.txFees, s.feeCollector), "round %d: swapped tx fee to the fee collector", round)
		gldR, bzeR := pool.reserveBase().AddRaw(feGldTxFee), pool.reserveQuote().Sub(txq.out)

		if s.swapFee(acc).IsZero() {
			require.Equal(t, sdk.NewCoins(sdk.NewCoin(s.gld, acc)), s.addrBalances(t, s.burnerFees, res.Height), "round %d: the gld waits in the burner fee collector", round)
			require.Empty(t, s.swapsBy(block, s.burnerFees), "round %d: no swap by the burner fee collector", round)
			continue
		}

		bq := s.quote(acc, gldR, bzeR)
		s.requireSwap(t, block, s.burnerFees, s.gldPool, sdk.NewCoin(s.gld, acc), sdk.NewCoin(chain.Denom, bq.out))
		require.Equal(t, sdk.NewCoins(sdk.NewCoin(chain.Denom, bq.out)), s.transfers(t, block, s.burnerFees, s.burner), "swapped taker fees to the burner")
		require.True(t, s.addrBalances(t, s.burnerFees, res.Height).IsZero(), "burner fee collector emptied")
		after := s.pool(t, s.gldPool)
		require.Equal(t, gldR.Add(acc), after.reserveBase(), "gld reserve")
		require.Equal(t, bzeR.Sub(bq.out), after.reserveQuote(), "ubze reserve")
		t.Logf("swapped %s gld of taker fees after %d rounds", acc, round)
		return
	}
}

// 5. A gld tx fee too small to swap stays in the txfeecollector (gld has
// deep liquidity, so it is not forwarded); the next gld fee makes the sum
// swappable and the EndBlock swaps all of it.
func (s *feesSuite) tooSmallToSwap(t *testing.T) {
	require.True(t, s.addrBalances(t, s.txFees, 0).IsZero(), "txfeecollector empty")
	res := s.mustExecWith(t, s.u1, []helpers.TxOption{helpers.WithFees(coinStr(feSmallGld, s.gld)), helpers.WithGas(feSendGas)},
		"bank", "send", s.u1.FormattedAddress(), s.admin.FormattedAddress(), coinStr(1, chain.Denom))
	require.Equal(t, coins(feSmallGld, s.gld), s.transfers(t, res.Events, s.u1.FormattedAddress(), s.txFees))
	require.Empty(t, s.swapsBy(s.blockEvents(t, res.Height), s.txFees), "nothing swapped")
	require.Equal(t, coins(feSmallGld, s.gld), s.addrBalances(t, s.txFees, res.Height), "the fee waits")

	pool := s.pool(t, s.gldPool)
	res = s.mustExecWith(t, s.u1, []helpers.TxOption{helpers.WithFees(coinStr(feGrowGld, s.gld)), helpers.WithGas(feSendGas)},
		"bank", "send", s.u1.FormattedAddress(), s.admin.FormattedAddress(), coinStr(1, chain.Denom))
	total := sdkmath.NewInt(feSmallGld + feGrowGld)
	q := s.quote(total, pool.reserveBase(), pool.reserveQuote())
	block := s.blockEvents(t, res.Height)
	s.requireSwap(t, block, s.txFees, s.gldPool, sdk.NewCoin(s.gld, total), sdk.NewCoin(chain.Denom, q.out))
	require.Equal(t, sdk.NewCoins(sdk.NewCoin(chain.Denom, q.out)), s.transfers(t, block, s.txFees, s.feeCollector), "swapped fees to the fee collector")
	require.True(t, s.addrBalances(t, s.txFees, res.Height).IsZero(), "txfeecollector emptied")
}

// 6. A pul fee too small to swap waits in the txfeecollector. Once the pul
// pool's liquidity is pulled below the threshold the fee can never be
// swapped: the EndBlock of that block forwards it as-is to the fee collector
// and the next block distributes it to the validator.
func (s *feesSuite) nonSwappable(t *testing.T) {
	res := s.mustExecWith(t, s.u1, []helpers.TxOption{helpers.WithFees(coinStr(fePulFee, s.pul)), helpers.WithGas(feSendGas)},
		"bank", "send", s.u1.FormattedAddress(), s.admin.FormattedAddress(), coinStr(1, chain.Denom))
	require.Equal(t, coins(fePulFee, s.pul), s.addrBalances(t, s.txFees, res.Height), "deep liquidity: the pul fee waits")

	res = s.mustExec(t, s.admin, "tradebin", "remove-liquidity", s.pulPool, s.pulDepositLp, "1", "1")
	require.True(t, s.pool(t, s.pulPool).reserveQuote().LT(s.minLiquidity), "pul pool below the threshold")
	block := s.blockEvents(t, res.Height)
	require.Equal(t, coins(fePulFee, s.pul), s.transfers(t, block, s.txFees, s.feeCollector), "forwarded as-is to the fee collector")
	require.Empty(t, s.swapsBy(block, s.txFees), "nothing swapped")
	require.True(t, s.addrBalances(t, s.txFees, res.Height).IsZero(), "txfeecollector emptied")

	next := res.Height + 1
	s.waitForHeight(t, next)
	alloc := s.requireAllocation(t, next, nil)
	require.Equal(t, sdkmath.NewInt(fePulFee), alloc.AmountOf(s.pul), "pul distributed")
}

// 7. The ubze minimum is ValidatorMinGasFee × gas, rounded up, even though
// the node accepts ten times less: one ubze under it is refused by CheckTx,
// exactly the minimum is accepted.
func (s *feesSuite) minGasNative(t *testing.T) {
	required := s.minFee(s.minNative, feSendGas)
	nodeMin := s.minFee(sdkmath.LegacyMustNewDecFromStr(feNodeMinGasPrice), feSendGas)
	require.True(t, nodeMin.LT(required), "the node minimum (%s) is below the param floor (%s)", nodeMin, required)
	s.requireMinFee(t, chain.Denom, required, feSendGas)
}

// 8. A non-native minimum is the ubze minimum divided by the denom's spot
// price on its ubze pool.
func (s *feesSuite) minGasNonNative(t *testing.T) {
	pool := s.pool(t, s.gldPool)
	spot := sdkmath.LegacyNewDecFromInt(pool.reserveQuote()).Quo(sdkmath.LegacyNewDecFromInt(pool.reserveBase()))
	required := s.minFee(s.minNative.Quo(spot), feMinGasGas)
	require.True(t, required.GT(sdkmath.OneInt()), "the gld minimum is more than one unit: %s", required)
	s.requireMinFee(t, s.gld, required, feMinGasGas)
}

// 9. Fees the ante handlers refuse before anything is charged.
func (s *feesSuite) anteRefusals(t *testing.T) {
	send := func(user ibc.Wallet, opts ...helpers.TxOption) (helpers.TxResult, error) {
		return helpers.ExecTxWith(s.ctx, s.bze, user, opts, "bank", "send", user.FormattedAddress(), s.admin.FormattedAddress(), coinStr(1, chain.Denom))
	}
	for _, c := range []struct {
		name      string
		user      ibc.Wallet
		opts      []helpers.TxOption
		codespace string
		code      uint32
		reason    string
	}{
		{"two fee denoms", s.u1, []helpers.TxOption{helpers.WithFees(coinStr(feNativeFee, chain.Denom) + "," + coinStr(feGldTxFee, s.gld)), helpers.WithGas(feSendGas)},
			"store", 7, "multiple denominations for same transaction fee are not supported"},
		{"zero fee", s.u1, []helpers.TxOption{helpers.WithFees(coinStr(0, chain.Denom)), helpers.WithGas(feSendGas)},
			"store", 7, "no fee supplied"},
		{"gas 0", s.u1, []helpers.TxOption{helpers.WithFees(coinStr(feNativeFee, chain.Denom)), helpers.WithFlags("--gas", "0")},
			"sdk", 11, "out of gas"},
		{"fee denom without deep liquidity", s.u1, []helpers.TxOption{helpers.WithFees(coinStr(fePulFee, s.pul)), helpers.WithGas(feSendGas)},
			"store", 7, "can be used to pay for fees only if enough liquidity is available"},
		{"payer can not cover the fee", s.poor, []helpers.TxOption{helpers.WithFees(coinStr(feNativeFee, chain.Denom)), helpers.WithGas(feSendGas)},
			"sdk", 5, "insufficient funds"},
	} {
		t.Run(c.name, func(t *testing.T) {
			before := s.addrBalances(t, c.user.FormattedAddress(), 0)
			res, err := send(c.user, c.opts...)
			s.requireRefused(t, res, err, c.codespace, c.code, c.reason)
			require.Zero(t, res.Height, "refused by CheckTx")
			require.Equal(t, before, s.addrBalances(t, c.user.FormattedAddress(), 0), "nothing charged")
		})
	}
}

// 10. Simulation (--gas auto) accepts an empty fee; the tx built from it then
// has no fee and CheckTx refuses it. A zero fee in a denom without deep
// liquidity is refused by the simulation itself.
func (s *feesSuite) simulation(t *testing.T) {
	t.Run("empty fee", func(t *testing.T) {
		res, err := helpers.ExecTxWith(s.ctx, s.bze, s.u1, []helpers.TxOption{helpers.WithFlags("--gas-prices", "")},
			"bank", "send", s.u1.FormattedAddress(), s.admin.FormattedAddress(), coinStr(1, chain.Denom))
		require.NoError(t, err, "the simulation accepts an empty fee")
		s.requireRefused(t, res, nil, "store", 7, "no fee supplied")
		require.Zero(t, res.Height)
	})
	t.Run("zero fee in a shallow denom", func(t *testing.T) {
		res, err := helpers.ExecTxWith(s.ctx, s.bze, s.u1, []helpers.TxOption{helpers.WithGasPrices("0.01" + s.pul)},
			"bank", "send", s.u1.FormattedAddress(), s.admin.FormattedAddress(), coinStr(1, chain.Denom))
		var simErr *helpers.SimulationError
		require.ErrorAs(t, err, &simErr, "refused by the simulation, res %+v", res)
		require.Contains(t, simErr.Output, "can be used to pay for fees only if enough liquidity is available")
	})
}

// 11. A fee pool whose ubze side is exactly MinNativeLiquidityForModuleSwap
// is not deep: the module swap needs strictly more, so the ante refuses its
// denom as a fee instead of accepting a fee the EndBlock could not convert.
func (s *feesSuite) thresholdBoundary(t *testing.T) {
	require.True(t, s.minLiquidity.IsInt64(), "threshold %s", s.minLiquidity)
	factory := "factory/" + s.admin.FormattedAddress() + "/"
	thr := factory + "thr"
	s.mustExec(t, s.admin, "tokenfactory", "create-denom", "thr")
	s.mustExec(t, s.admin, "tokenfactory", "mint", coinStr(feMint, thr))
	s.mustExec(t, s.admin, "bank", "send", s.admin.FormattedAddress(), s.u1.FormattedAddress(), coinStr(feUserThr, thr))
	s.createPool(t, thr, chain.Denom, feProvidersDest, feThrPoolThr, s.minLiquidity.Int64())
	require.Equal(t, s.minLiquidity, s.pool(t, thr+"_"+chain.Denom).reserveQuote(), "ubze side exactly at the threshold")

	before := s.addrBalances(t, s.u1.FormattedAddress(), 0)
	res, err := helpers.ExecTxWith(s.ctx, s.bze, s.u1, []helpers.TxOption{helpers.WithFees(coinStr(feThrFee, thr)), helpers.WithGas(feSendGas)},
		"bank", "send", s.u1.FormattedAddress(), s.admin.FormattedAddress(), coinStr(1, chain.Denom))
	s.requireRefused(t, res, err, "store", 7, "can be used to pay for fees only if enough liquidity is available")
	require.Zero(t, res.Height, "refused by CheckTx")
	require.Equal(t, before, s.addrBalances(t, s.u1.FormattedAddress(), 0), "nothing charged")
}

// 12. A granter pays the grantee's fees within the spend limit, in ubze and
// in gld; the tx event names the granter as fee payer. Above the remaining
// limit, or without a grant, the tx is refused.
func (s *feesSuite) feeGrants(t *testing.T) {
	granter, grantee := s.granter.FormattedAddress(), s.grantee.FormattedAddress()
	limit := sdk.NewCoins(sdk.NewInt64Coin(chain.Denom, feGrantUbze), sdk.NewInt64Coin(s.gld, feGrantGld))
	s.mustExec(t, s.granter, "feegrant", "grant", granter, grantee, "--spend-limit", limit.String())

	sendWithGrant := func(user ibc.Wallet, fee string) (helpers.TxResult, error) {
		return helpers.ExecTxWith(s.ctx, s.bze, user, []helpers.TxOption{helpers.WithFees(fee), helpers.WithGas(feSendGas)},
			"bank", "send", user.FormattedAddress(), s.admin.FormattedAddress(), coinStr(1, chain.Denom), "--fee-granter", granter)
	}

	t.Run("native fee", func(t *testing.T) {
		gr, ge := s.addrBalances(t, granter, 0), s.addrBalances(t, grantee, 0)
		fee := coins(feGrantFee, chain.Denom)
		res, err := sendWithGrant(s.grantee, fee.String())
		require.NoError(t, err)
		require.Zero(t, res.Code, res.RawLog)
		require.Equal(t, granter, s.feePayer(t, res), "fee payer")
		require.Equal(t, fee, s.transfers(t, res.Events, granter, s.feeCollector), "the granter pays")
		require.Equal(t, gr.Sub(fee...), s.addrBalances(t, granter, 0), "granter balances")
		require.Equal(t, ge.Sub(coins(1, chain.Denom)...), s.addrBalances(t, grantee, 0), "grantee pays no fee")
	})

	t.Run("non-native fee", func(t *testing.T) {
		waiting := s.addrBalances(t, s.txFees, 0).AmountOf(s.gld)
		gr, ge := s.addrBalances(t, granter, 0), s.addrBalances(t, grantee, 0)
		fee := coins(feGrowGld, s.gld)
		pool := s.pool(t, s.gldPool)
		res, err := sendWithGrant(s.grantee, fee.String())
		require.NoError(t, err)
		require.Zero(t, res.Code, res.RawLog)
		require.Equal(t, granter, s.feePayer(t, res), "fee payer")
		require.Equal(t, fee, s.transfers(t, res.Events, granter, s.txFees), "the granter pays, to the txfeecollector")
		require.Equal(t, gr.Sub(fee...), s.addrBalances(t, granter, 0), "granter balances")
		require.Equal(t, ge.Sub(coins(1, chain.Denom)...), s.addrBalances(t, grantee, 0), "grantee pays no fee")

		total := waiting.AddRaw(feGrowGld)
		q := s.quote(total, pool.reserveBase(), pool.reserveQuote())
		block := s.blockEvents(t, res.Height)
		s.requireSwap(t, block, s.txFees, s.gldPool, sdk.NewCoin(s.gld, total), sdk.NewCoin(chain.Denom, q.out))
		require.Equal(t, sdk.NewCoins(sdk.NewCoin(chain.Denom, q.out)), s.transfers(t, block, s.txFees, s.feeCollector), "swapped to the fee collector")
	})

	t.Run("above the spend limit", func(t *testing.T) {
		// 8000 ubze of the limit is left
		before := s.addrBalances(t, granter, 0)
		res, err := sendWithGrant(s.grantee, coinStr(feGrantUbze, chain.Denom))
		s.requireRefused(t, res, err, "feegrant", 2, "does not allow to pay fees for")
		require.Equal(t, before, s.addrBalances(t, granter, 0), "granter balances")
	})

	t.Run("without a grant", func(t *testing.T) {
		before := s.addrBalances(t, granter, 0)
		res, err := sendWithGrant(s.u1, coinStr(feGrantFee, chain.Denom))
		s.requireRefused(t, res, err, "sdk", 38, "fee-grant not found")
		require.Equal(t, before, s.addrBalances(t, granter, 0), "granter balances")
	})
}

// 13. The txfeecollector params change only through gov. A higher
// ValidatorMinGasFee refuses a fee that was enough before; MaxBalanceIterations
// 1 stops the EndBlock at the first balance (a gld fee too small to swap), so
// a swappable tre fee waits until the cap is lifted.
func (s *feesSuite) paramsByGov(t *testing.T) {
	t.Run("signed by a user", func(t *testing.T) {
		before := s.addrBalances(t, s.u1.FormattedAddress(), 0)
		res, err := helpers.ExecMsgs(s.ctx, s.bze, s.u1, []helpers.TxOption{helpers.WithGas(feTxGas)}, s.paramsMsg(s.u1.FormattedAddress(), "0.02", 100))
		s.requireRefused(t, res, err, "txfeecollector", 1100, "invalid authority")
		require.Equal(t, before.Sub(sdk.NewCoin(chain.Denom, s.txFee(t, res))), s.addrBalances(t, s.u1.FormattedAddress(), 0), "u1 pays only the tx fee")
	})
	for _, c := range []struct {
		name, amount, denom string
		iterations          uint64
		reason              string
	}{
		{"denom other than ubze", "0.01", s.gld, 100, "validator min gas fee denom must be ubze"},
		{"negative min gas fee", "-0.01", chain.Denom, 100, "validator min gas fee amount cannot be negative"},
		{"zero iterations", "0.01", chain.Denom, 0, "max balance iterations must be greater than 0"},
	} {
		t.Run(c.name, func(t *testing.T) {
			msg := s.paramsMsg(s.gov, c.amount, c.iterations)
			msg["params"].(map[string]any)["ValidatorMinGasFee"] = map[string]string{"denom": c.denom, "amount": c.amount}
			_, err := helpers.SubmitAndPassProposalWith(s.ctx, s.bze, s.admin, s.govOpts(), helpers.Proposal{
				Messages: []any{msg}, Title: "invalid txfeecollector params", Summary: "e2e: " + c.name,
			})
			require.ErrorContains(t, err, c.reason)
			require.Equal(t, s.paramsBefore, s.txfeeParams(t), "params unchanged")
		})
	}

	t.Run("raised minimum gas price", func(t *testing.T) {
		s.setParams(t, "0.02", 100)
		require.Equal(t, sdkmath.LegacyMustNewDecFromStr("0.02"), s.txfeeParams(t).minGasPrice(t))
		old := s.minFee(s.minNative, feSendGas)
		raised := s.minFee(sdkmath.LegacyMustNewDecFromStr("0.02"), feSendGas)
		res, err := helpers.ExecTxWith(s.ctx, s.bze, s.u1, []helpers.TxOption{helpers.WithFees(sdk.NewCoin(chain.Denom, old).String()), helpers.WithGas(feSendGas)},
			"bank", "send", s.u1.FormattedAddress(), s.admin.FormattedAddress(), coinStr(1, chain.Denom))
		s.requireRefused(t, res, err, "sdk", 13, fmt.Sprintf("insufficient fees; got: %s required: %s", sdk.NewCoin(chain.Denom, old), sdk.NewCoin(chain.Denom, raised)))
		s.mustExecWith(t, s.u1, []helpers.TxOption{helpers.WithFees(sdk.NewCoin(chain.Denom, raised).String()), helpers.WithGas(feSendGas)},
			"bank", "send", s.u1.FormattedAddress(), s.admin.FormattedAddress(), coinStr(1, chain.Denom))

		s.setParams(t, "0.01", 100)
		require.Equal(t, s.paramsBefore, s.txfeeParams(t), "params restored")
	})

	t.Run("max balance iterations", func(t *testing.T) {
		require.True(t, s.addrBalances(t, s.txFees, 0).IsZero(), "txfeecollector empty")
		s.mustExecWith(t, s.u1, []helpers.TxOption{helpers.WithFees(coinStr(feSmallGld, s.gld)), helpers.WithGas(feSendGas)},
			"bank", "send", s.u1.FormattedAddress(), s.admin.FormattedAddress(), coinStr(1, chain.Denom))
		s.setParams(t, "0.01", 1)

		// balances iterate by denom: gld (too small, skipped) uses the only
		// iteration, so the swappable tre is never reached
		res := s.mustExecWith(t, s.u1, []helpers.TxOption{helpers.WithFees(coinStr(feTreFee, s.tre)), helpers.WithGas(feSendGas)},
			"bank", "send", s.u1.FormattedAddress(), s.admin.FormattedAddress(), coinStr(1, chain.Denom))
		waiting := sdk.NewCoins(sdk.NewInt64Coin(s.gld, feSmallGld), sdk.NewInt64Coin(s.tre, feTreFee))
		require.Empty(t, s.swapsBy(s.blockEvents(t, res.Height), s.txFees), "nothing swapped")
		require.Equal(t, waiting, s.addrBalances(t, s.txFees, res.Height), "the tre fee waits behind the gld one")

		pool := s.pool(t, s.trePool)
		executed := s.setParams(t, "0.01", 100)
		require.Equal(t, s.paramsBefore, s.txfeeParams(t), "params restored")
		// gov's EndBlock runs before the txfeecollector's: the same block converts the tre
		q := s.quote(sdkmath.NewInt(feTreFee), pool.reserveBase(), pool.reserveQuote())
		block := s.blockEvents(t, executed)
		s.requireSwap(t, block, s.txFees, s.trePool, sdk.NewInt64Coin(s.tre, feTreFee), sdk.NewCoin(chain.Denom, q.out))
		require.Equal(t, coins(feSmallGld, s.gld), s.addrBalances(t, s.txFees, executed), "only the gld waits")
	})
}

// requireMinFee asserts a bank send with gas limit gas paying one unit less
// than required in denom is refused by CheckTx, and paying required is
// accepted and charged exactly.
func (s *feesSuite) requireMinFee(t *testing.T, denom string, required sdkmath.Int, gas uint64) {
	t.Helper()
	under := sdk.NewCoin(denom, required.SubRaw(1))
	exact := sdk.NewCoin(denom, required)
	before := s.addrBalances(t, s.u1.FormattedAddress(), 0)
	res, err := helpers.ExecTxWith(s.ctx, s.bze, s.u1, []helpers.TxOption{helpers.WithFees(under.String()), helpers.WithGas(gas)},
		"bank", "send", s.u1.FormattedAddress(), s.admin.FormattedAddress(), coinStr(1, chain.Denom))
	s.requireRefused(t, res, err, "sdk", 13, fmt.Sprintf("insufficient fees; got: %s required: %s", under, exact))
	require.Equal(t, before, s.addrBalances(t, s.u1.FormattedAddress(), 0), "nothing charged")

	res = s.mustExecWith(t, s.u1, []helpers.TxOption{helpers.WithFees(exact.String()), helpers.WithGas(gas)},
		"bank", "send", s.u1.FormattedAddress(), s.admin.FormattedAddress(), coinStr(1, chain.Denom))
	paid, err := helpers.TxFee(res, denom)
	require.NoError(t, err)
	require.Equal(t, required, paid)
	require.Equal(t, before.Sub(sdk.NewCoins(exact).Add(sdk.NewInt64Coin(chain.Denom, 1))...), s.addrBalances(t, s.u1.FormattedAddress(), 0), "u1 pays the minimum")
}

// requireFunded asserts the community-pool collector funded the community
// pool with exactly want in the EndBlock of height and kept nothing but
// coins too small to swap.
func (s *feesSuite) requireFunded(t *testing.T, height int64, want sdk.Coins) {
	t.Helper()
	require.Equal(t, want, s.transfers(t, s.blockEvents(t, height), s.cpFees, s.distr), "community-pool collector to distribution")
	require.True(t, s.addrBalances(t, s.cpFees, height).AmountOf(chain.Denom).IsZero(), "no ubze left in the collector")
	s.requireAllocation(t, height, want)
}

// requireAllocation asserts the BeginBlock distribution of height: the fee
// collector's whole balance went to distribution, the single validator's
// outstanding rewards grew by its non-taxed share (rounded down like the
// chain, at 18 decimals) and its commission by its rate of that, and the
// community pool by the rest plus funded (the community-pool collector's
// EndBlock funding). It returns the allocated coins.
func (s *feesSuite) requireAllocation(t *testing.T, height int64, funded sdk.Coins) sdk.Coins {
	t.Helper()
	alloc := s.transfers(t, s.blockEvents(t, height), s.feeCollector, s.distr)
	require.False(t, alloc.Empty(), "nothing allocated at %d", height)
	all := sdk.NewDecCoinsFromCoins(alloc...)
	reward := all.MulDecTruncate(sdkmath.LegacyOneDec().Sub(s.tax))

	requireDecDelta(t, "outstanding rewards", s.decCoinsAt(t, height-1, "validator-outstanding-rewards"), s.decCoinsAt(t, height, "validator-outstanding-rewards"), reward)
	requireDecDelta(t, "commission", s.decCoinsAt(t, height-1, "commission"), s.decCoinsAt(t, height, "commission"), reward.MulDec(s.commissionRate))
	requireDecDelta(t, "community pool", s.decCoinsAt(t, height-1, "community-pool"), s.decCoinsAt(t, height, "community-pool"),
		all.Sub(reward).Add(sdk.NewDecCoinsFromCoins(funded...)...))
	return alloc
}

func requireDecDelta(t *testing.T, name string, before, after, delta sdk.DecCoins) {
	t.Helper()
	want := before.Add(delta...)
	require.True(t, want.Equal(after), "%s: before %s, want +%s = %s, got %s", name, before, delta, want, after)
}

// decCoinsAt reads a distribution query of the validator (or the community
// pool) at height. The CLI prints DecCoins as strings ("1.5ubze").
func (s *feesSuite) decCoinsAt(t *testing.T, height int64, query string) sdk.DecCoins {
	t.Helper()
	args := []string{"distribution", query}
	if query != "community-pool" {
		args = append(args, s.valoper)
	}
	args = append(args, "--height", strconv.FormatInt(height, 10))
	raw, err := helpers.Query[map[string]json.RawMessage](s.ctx, s.bze, args...)
	require.NoError(t, err)

	// community-pool: {"pool": [...]}; outstanding: {"rewards": {"rewards": [...]}};
	// commission: {"commission": {"commission": [...]}}
	var list json.RawMessage
	switch query {
	case "community-pool":
		list = raw["pool"]
	case "validator-outstanding-rewards":
		list = nested(t, raw["rewards"], "rewards")
	case "commission":
		list = nested(t, raw["commission"], "commission")
	}
	var items []string
	if len(list) > 0 {
		require.NoError(t, json.Unmarshal(list, &items), "%s: %s", query, list)
	}
	out := sdk.NewDecCoins()
	for _, it := range items {
		c, err := sdk.ParseDecCoin(it)
		require.NoError(t, err, "%s: %q", query, it)
		out = out.Add(c)
	}
	return out
}

func nested(t *testing.T, raw json.RawMessage, key string) json.RawMessage {
	t.Helper()
	if len(raw) == 0 {
		return nil
	}
	var m map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(raw, &m), "%s", raw)
	return m[key]
}

// legacyDec reads a cosmos.Dec the CLI printed, with or without its decimal
// point (a raw value has 18 implied decimals).
func legacyDec(t *testing.T, v string) sdkmath.LegacyDec {
	t.Helper()
	if strings.Contains(v, ".") {
		d, err := sdkmath.LegacyNewDecFromStr(v)
		require.NoError(t, err, "%q", v)
		return d
	}
	return sdkmath.LegacyNewDecFromBigIntWithPrec(parseInt(v).BigInt(), sdkmath.LegacyPrecision)
}

func (s *feesSuite) validator(t *testing.T) (string, sdkmath.LegacyDec) {
	t.Helper()
	res, err := helpers.Query[struct {
		Validators []struct {
			OperatorAddress string `json:"operator_address"`
			Commission      struct {
				CommissionRates struct {
					Rate string `json:"rate"`
				} `json:"commission_rates"`
			} `json:"commission"`
		} `json:"validators"`
	}](s.ctx, s.bze, "staking", "validators")
	require.NoError(t, err)
	require.Len(t, res.Validators, 1, "the suite's distribution math assumes one validator")
	v := res.Validators[0]
	return v.OperatorAddress, legacyDec(t, v.Commission.CommissionRates.Rate)
}

func (s *feesSuite) communityTax(t *testing.T) sdkmath.LegacyDec {
	t.Helper()
	res, err := helpers.Query[struct {
		Params struct {
			CommunityTax string `json:"community_tax"`
		} `json:"params"`
	}](s.ctx, s.bze, "distribution", "params")
	require.NoError(t, err)
	return legacyDec(t, res.Params.CommunityTax)
}

func (s *feesSuite) tokenfactoryFee(t *testing.T) sdk.Coin {
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
	fee := sdk.NewCoin(res.Params.CreateDenomFee.Denom, parseInt(res.Params.CreateDenomFee.Amount))
	require.Equal(t, chain.Denom, fee.Denom)
	require.True(t, fee.IsPositive(), "create-denom fee")
	return fee
}

func (s *feesSuite) minNativeLiquidity(t *testing.T) sdkmath.Int {
	t.Helper()
	res, err := helpers.Query[struct {
		Params struct {
			MinNativeLiquidityForModuleSwap string `json:"minNativeLiquidityForModuleSwap"`
		} `json:"params"`
	}](s.ctx, s.bze, "tradebin", "params")
	require.NoError(t, err)
	return parseInt(res.Params.MinNativeLiquidityForModuleSwap)
}

type feTxfeeParams struct {
	MinGasDenom          string
	MinGasAmount         string
	MaxBalanceIterations uint64
}

func (p feTxfeeParams) minGasPrice(t *testing.T) sdkmath.LegacyDec {
	t.Helper()
	require.Equal(t, chain.Denom, p.MinGasDenom)
	return legacyDec(t, p.MinGasAmount)
}

// txfeeParams reads the txfeecollector params; the CLI prints their names
// as proto field names or snake_case depending on the encoder, so keys are
// matched like the IBC suite does (normalizeKey).
func (s *feesSuite) txfeeParams(t *testing.T) feTxfeeParams {
	t.Helper()
	res, err := helpers.Query[struct {
		Params map[string]json.RawMessage `json:"params"`
	}](s.ctx, s.bze, "txfeecollector", "params")
	require.NoError(t, err)
	byName := map[string]json.RawMessage{}
	for k, v := range res.Params {
		byName[normalizeKey(k)] = v
	}

	var p feTxfeeParams
	var iterations helpers.Int64
	require.NoError(t, json.Unmarshal(byName["maxbalanceiterations"], &iterations), "params %v", res.Params)
	p.MaxBalanceIterations = uint64(iterations)

	// a DecCoin prints as "0.01ubze" (autocli) or as {"denom", "amount"}
	raw := byName["validatormingasfee"]
	var str string
	if json.Unmarshal(raw, &str) == nil {
		c, err := sdk.ParseDecCoin(str)
		require.NoError(t, err, "%q", str)
		p.MinGasDenom, p.MinGasAmount = c.Denom, c.Amount.String()
		return p
	}
	var c struct {
		Denom  string `json:"denom"`
		Amount string `json:"amount"`
	}
	require.NoError(t, json.Unmarshal(raw, &c), "params %v", res.Params)
	p.MinGasDenom, p.MinGasAmount = c.Denom, legacyDec(t, c.Amount).String()
	return p
}

// paramsMsg is a txfeecollector MsgUpdateParams; the proposal JSON wants the
// params by proto field name.
func (s *feesSuite) paramsMsg(authority, minGas string, iterations uint64) map[string]any {
	return map[string]any{
		"@type":     "/bze.txfeecollector.MsgUpdateParams",
		"authority": authority,
		"params": map[string]any{
			"ValidatorMinGasFee":   map[string]string{"denom": chain.Denom, "amount": minGas},
			"MaxBalanceIterations": strconv.FormatUint(iterations, 10),
			"BlockedIbcInbound":    []any{},
		},
	}
}

// govOpts pay the proposal and the votes above any minimum the suite sets.
func (s *feesSuite) govOpts() []helpers.TxOption {
	return []helpers.TxOption{helpers.WithGas(feTxGas), helpers.WithGasPrices(feGovPrices)}
}

// setParams passes a proposal setting the txfeecollector params and returns
// the height of the block whose gov EndBlock executed it.
func (s *feesSuite) setParams(t *testing.T, minGas string, iterations uint64) int64 {
	t.Helper()
	from, err := s.bze.Height(s.ctx)
	require.NoError(t, err)
	id, err := helpers.SubmitAndPassProposalWith(s.ctx, s.bze, s.admin, s.govOpts(), helpers.Proposal{
		Messages: []any{s.paramsMsg(s.gov, minGas, iterations)},
		Title:    "txfeecollector params",
		Summary:  fmt.Sprintf("e2e: min gas %s, iterations %d", minGas, iterations),
	})
	require.NoError(t, err)
	require.EqualValues(t, iterations, s.txfeeParams(t).MaxBalanceIterations)

	to, err := s.bze.Height(s.ctx)
	require.NoError(t, err)
	attrs := map[string]string{"proposal_id": strconv.FormatUint(id, 10), "proposal_result": "proposal_passed"}
	for h := from; h <= to; h++ {
		events, err := helpers.BlockEvents(s.ctx, s.bze, h, "active_proposal")
		require.NoError(t, err)
		if len(helpers.EventsWith(events, attrs)) > 0 {
			return h
		}
	}
	require.FailNow(t, "no active_proposal event", "proposal %d between heights %d and %d", id, from, to)
	return 0
}

func (s *feesSuite) createPool(t *testing.T, base, quote, dest string, baseAmount, quoteAmount int64) {
	t.Helper()
	s.mustExecWith(t, s.admin, []helpers.TxOption{helpers.WithGas(feTxGas)},
		"tradebin", "create-liquidity-pool", base, quote, ammPoolFee, dest, "false",
		strconv.FormatInt(baseAmount, 10), strconv.FormatInt(quoteAmount, 10))
	pool := s.pool(t, base+"_"+quote)
	require.Equal(t, base, pool.Base)
	require.Equal(t, sdkmath.NewInt(baseAmount), pool.reserveBase())
	require.Equal(t, sdkmath.NewInt(quoteAmount), pool.reserveQuote())
}

// feQuote is one swap computed like the chain: the pool fee is
// ⌊in × 0.002⌋ (treasury pools send it to the community-pool collector,
// provider pools keep it in the input reserve) and
// out = ⌊outReserve × (in − fee) / (inReserve + in − fee)⌋ at 18 decimals.
type feQuote struct {
	fee, out sdkmath.Int
}

func (s *feesSuite) swapFee(in sdkmath.Int) sdkmath.Int {
	return sdkmath.LegacyNewDecFromInt(in).Mul(s.poolFee).TruncateInt()
}

func (s *feesSuite) quote(in, inReserve, outReserve sdkmath.Int) feQuote {
	fee := s.swapFee(in)
	net := in.Sub(fee)
	out := sdkmath.LegacyNewDecFromInt(outReserve.Mul(net)).Quo(sdkmath.LegacyNewDecFromInt(inReserve.Add(net))).TruncateInt()
	return feQuote{fee: fee, out: out}
}

// inputFor is the input that buys out, like tradebin's
// CalculateOptimalInputForOutput: ⌈out × inReserve / (outReserve − out) / (1 − fee)⌉.
func (s *feesSuite) inputFor(out, inReserve, outReserve sdkmath.Int) sdkmath.Int {
	o := sdkmath.LegacyNewDecFromInt(out)
	net := o.Mul(sdkmath.LegacyNewDecFromInt(inReserve)).Quo(sdkmath.LegacyNewDecFromInt(outReserve).Sub(o))
	return net.Quo(sdkmath.LegacyOneDec().Sub(s.poolFee)).Ceil().TruncateInt()
}

// minFee is ⌈price × gas⌉, like the ante handler's required fee.
func (s *feesSuite) minFee(price sdkmath.LegacyDec, gas uint64) sdkmath.Int {
	return price.Mul(sdkmath.LegacyNewDec(int64(gas))).Ceil().RoundInt()
}

func (s *feesSuite) swapsBy(events []helpers.Event, creator string) []helpers.Event {
	return helpers.EventsWith(filterType(events, tradebinEventPrefix+"SwapEvent"), map[string]string{"creator": creator})
}

func (s *feesSuite) feePayer(t *testing.T, res helpers.TxResult) string {
	t.Helper()
	for _, e := range helpers.TxEvents(res, "tx") {
		if v, ok := e.Attributes["fee_payer"]; ok {
			return v
		}
	}
	require.FailNow(t, "no fee_payer attribute", "tx %s", res.TxHash)
	return ""
}

// addrBalances returns every balance of addr at height (0 for the latest).
func (s *feesSuite) addrBalances(t *testing.T, addr string, height int64) sdk.Coins {
	t.Helper()
	args := []string{"bank", "balances", addr}
	if height > 0 {
		args = append(args, "--height", strconv.FormatInt(height, 10))
	}
	res, err := helpers.Query[struct {
		Balances []struct {
			Denom  string `json:"denom"`
			Amount string `json:"amount"`
		} `json:"balances"`
	}](s.ctx, s.bze, args...)
	require.NoError(t, err)
	out := sdk.NewCoins()
	for _, b := range res.Balances {
		out = out.Add(sdk.NewCoin(b.Denom, parseInt(b.Amount)))
	}
	return out
}

// transfers sums every coin moved from one address to another by the bank
// transfer events in events.
func (s *feesSuite) transfers(t *testing.T, events []helpers.Event, from, to string) sdk.Coins {
	t.Helper()
	out := sdk.NewCoins()
	for _, e := range helpers.EventsWith(filterType(events, "transfer"), map[string]string{"sender": from, "recipient": to}) {
		c, err := sdk.ParseCoinsNormalized(e.Attributes["amount"])
		require.NoError(t, err)
		out = out.Add(c...)
	}
	return out
}

func (s *feesSuite) waitForHeight(t *testing.T, h int64) {
	t.Helper()
	_, err := helpers.WaitForHeight(s.ctx, s.bze, h)
	require.NoError(t, err)
}

// requireRefused asserts a refusal with codespace, code and the chain's message.
func (s *feesSuite) requireRefused(t *testing.T, res helpers.TxResult, err error, codespace string, code uint32, reason string) {
	t.Helper()
	got, err := helpers.RejectionReason(res, err)
	require.NoError(t, err)
	require.Contains(t, got, reason)
	require.EqualValues(t, code, res.Code, got)
	require.Equal(t, codespace, res.Codespace, got)
}
