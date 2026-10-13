package interchaintest_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"testing"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/interchaintest/v10"
	"github.com/cosmos/interchaintest/v10/chain/cosmos"
	"github.com/cosmos/interchaintest/v10/ibc"
	"github.com/cosmos/interchaintest/v10/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bze-alphateam/bze/interchaintest/chain"
	"github.com/bze-alphateam/bze/interchaintest/helpers"
)

// Pool amounts are chosen so the initial LP is exact: the chain mints
// sqrt(base × quote) × 10^6 LP and both products below are perfect squares.
// Liquidity is added and removed in exact tenths of a reserve or of the LP
// supply, so the chain's decimal ratios never round. Swap outputs are not
// round numbers; the suite computes them with the formulas of quoteSwap and
// inputFor from the reserves it queried before the tx.
const (
	ammLpFunds   = 1_000_000_000_000 // 1,000,000 BZE: two denoms, two pools, a market, liquidity
	ammUserFunds = 10_000_000_000    // 10,000 BZE each

	ammPoolFee = "0.002"
	// the burner gets half of every swap fee, the LPs keep the other half
	ammFeeDest = `{"treasury":"0","burner":"0.5","providers":"0.5"}`

	// usd/ubze: 4 ubze per usd; sqrt(5e10 × 2e11) = 1e11
	usdPoolUsd = 50_000_000_000
	usdPoolBze = 200_000_000_000
	// abc/usd: 0.25 usd per abc; sqrt(4e10 × 1e10) = 2e10
	abcPoolAbc = 40_000_000_000
	abcPoolUsd = 10_000_000_000
	lpScale    = 1_000_000

	ammMint      = 100_000_000_000 // usd and abc minted by lp
	ammUserUsd   = 1_000_000_000   // usd sent to trader and taker
	ammMakerAbc  = 100_000_000     // abc sent to maker
	ammSwapInput = 1_000_000_000   // 1,000 BZE swapped in steps 3 and 4

	// a tx fee in usd, well above the minimum gas price converted at the pool
	// price (0.01 ubze per gas ≈ 0.0025 usd per gas)
	ammUsdFee  = 20_000
	ammSendGas = 200_000
	ammTxGas   = 500_000
	ammFillGas = 1_000_000

	ammOrderAmount = 1_000

	tradebinEventPrefix = "bze.tradebin."
	ammCodespace        = "tradebin"
	ammMaxRoutes        = 5 // tradebin MaxRoutes

	// steps 10, 11 and 14
	ammBaseSwapInput = 100_000_000   // 100 usd swapped to ubze
	ammExcessBze     = 1_000_000_000 // ubze of the excess-base deposit
	ammGuardUsd      = 400_000_007   // usd of the slippage-guard deposit

	// step 12: rev/ubze sent as ubze/rev; sqrt(4e9 × 1e9) = 2e9
	revPoolBze    = 4_000_000_000
	revPoolRev    = 1_000_000_000
	ammMaxPoolFee = "0.05"

	// step 15: abc/ubze with a treasury share; sqrt(1e9 × 4e9) = 2e9
	cpPoolAbc            = 1_000_000_000
	cpPoolBze            = 4_000_000_000
	ammMinPoolFee        = "0.001"
	ammTreasuryFeeDest   = `{"treasury":"0.3","burner":"0.2","providers":"0.5"}`
	ammTreasurySwapInput = 123_456_789
)

// TestAmm runs the tradebin AMM and the fee path on one network: pools,
// liquidity, single and multi-hop swaps with exact outputs, a tx fee paid in a
// factory token and swapped to ubze in the txfeecollector EndBlock, the taker
// fee charged in that token and burned, --gas auto with --gas-prices in ubze
// and in a factory token, the message_id that links the order events of a
// MsgFillOrders to the EndBlock engine events, then the swap and deposit
// directions the first steps leave out, slippage guards, a pool fee share to
// the community pool, LP denom metadata and the create-pool, swap and
// create-market refusals. Subtests run in order and share state; the step
// numbers match the scenario of the suite's design.
func TestAmm(t *testing.T) {
	if testing.Short() {
		t.Skip("e2e: skipped in short mode")
	}
	ctx := context.Background()
	net := chain.Start(t, ctx)
	s := setupAmm(t, ctx, net.Chain)

	t.Run("1 create pools", s.createPools)
	t.Run("2 add liquidity", s.addLiquidity)
	t.Run("3 single swap", s.singleSwap)
	t.Run("4 multi-hop swap and min output", s.multiHopSwap)
	t.Run("5 remove liquidity", s.removeLiquidity)
	t.Run("6 tx fee in a non-native denom", s.txFeeInUsd)
	t.Run("7 taker fee in the fee denom", s.takerFeeInUsd)
	t.Run("8 gas auto with gas prices", s.gasAuto)
	t.Run("9 fill orders and message ids", s.fillOrders)
	t.Run("10 swap base to quote", s.swapBaseToQuote)
	t.Run("11 add liquidity with excess base", s.addExcessBase)
	t.Run("12 pool with base and quote reversed", s.reversedPool)
	t.Run("13 LP denom metadata", s.lpMetadata)
	t.Run("14 slippage guards", s.slippageGuards)
	t.Run("15 pool fee treasury share", s.treasuryShare)
	t.Run("16 create pool refusals", s.createPoolRefusals)
	t.Run("17 swap refusals", s.swapRefusals)
	t.Run("18 create market refusals", s.createMarketRefusals)
}

type ammSuite struct {
	ctx context.Context
	bze *cosmos.CosmosChain

	lp, trader, maker, taker ibc.Wallet
	usd, abc                 string
	usdPool, abcPool         string
	market                   string
	params                   tradebinParams

	// steps 12 and 15: rev/ubze created as ubze/rev, abc/ubze with a treasury share
	rev, revPool, cpPool string

	// module account addresses
	tradebin, burner, blackHole, feeCollector, txFees, burnerFees, cpFees, distribution string
}

func setupAmm(t *testing.T, ctx context.Context, bze *cosmos.CosmosChain) *ammSuite {
	s := &ammSuite{ctx: ctx, bze: bze}

	s.lp = interchaintest.GetAndFundTestUsers(t, ctx, "amm-lp", sdkmath.NewInt(ammLpFunds), bze)[0]
	users := interchaintest.GetAndFundTestUsers(t, ctx, "amm", sdkmath.NewInt(ammUserFunds), bze, bze, bze)
	s.trader, s.maker, s.taker = users[0], users[1], users[2]
	require.NoError(t, testutil.WaitForBlocks(ctx, 2, bze))

	factory := "factory/" + s.lp.FormattedAddress() + "/"
	s.usd, s.abc = factory+"usd", factory+"abc"
	for _, sub := range []string{"usd", "abc"} {
		s.mustExec(t, s.lp, "tokenfactory", "create-denom", sub)
		s.mustExec(t, s.lp, "tokenfactory", "mint", coinStr(ammMint, factory+sub))
	}
	for _, send := range []struct {
		to   ibc.Wallet
		coin string
	}{
		{s.trader, coinStr(ammUserUsd, s.usd)},
		{s.taker, coinStr(ammUserUsd, s.usd)},
		{s.maker, coinStr(ammMakerAbc, s.abc)},
	} {
		s.mustExec(t, s.lp, "bank", "send", s.lp.FormattedAddress(), send.to.FormattedAddress(), send.coin)
	}

	// pool ids order the denoms alphabetically: "factory/…" sorts before "ubze"
	s.usdPool = s.usd + "_" + chain.Denom
	s.abcPool = s.abc + "_" + s.usd
	s.market = s.abc + "/" + s.usd
	s.rev = factory + "rev"
	s.revPool = s.rev + "_" + chain.Denom
	s.cpPool = s.abc + "_" + chain.Denom

	for name, addr := range map[string]*string{
		"tradebin":              &s.tradebin,
		"burner":                &s.burner,
		"burner_black_hole":     &s.blackHole,
		"fee_collector":         &s.feeCollector,
		"txfeecollector":        &s.txFees,
		"txfeecollector_burner": &s.burnerFees,
		"txfeecollector_cp":     &s.cpFees,
		"distribution":          &s.distribution,
	} {
		var err error
		*addr, err = helpers.ModuleAddress(ctx, bze, name)
		require.NoError(t, err)
	}

	s.params = s.queryParams(t)
	return s
}

// 1. Both pools start at their initial reserves; the initial LP is minted to
// the black hole (locked forever) and the creator pays the create fee.
func (s *ammSuite) createPools(t *testing.T) {
	for _, p := range []struct {
		base, quote          string
		initBase, initQuote  int64
		poolID, wantBase     string
		wantBaseR, wantQuote int64
	}{
		{s.usd, chain.Denom, usdPoolUsd, usdPoolBze, s.usdPool, s.usd, usdPoolUsd, usdPoolBze},
		{s.abc, s.usd, abcPoolAbc, abcPoolUsd, s.abcPool, s.abc, abcPoolAbc, abcPoolUsd},
	} {
		before := s.balances(t, s.lp, chain.Denom, s.usd, s.abc)
		res := s.mustExecWith(t, s.lp, []helpers.TxOption{helpers.WithGas(ammTxGas)},
			"tradebin", "create-liquidity-pool", p.base, p.quote, ammPoolFee, ammFeeDest, "false",
			sdkmath.NewInt(p.initBase).String(), sdkmath.NewInt(p.initQuote).String())

		ev := s.requireTxEvent(t, res, "PoolCreatedEvent", map[string]string{"creator": s.lp.FormattedAddress(), "base": p.base, "quote": p.quote})
		pool := s.pool(t, p.poolID)
		require.Equal(t, p.wantBase, pool.Base)
		require.Equal(t, ev.Attributes["lp_denom"], pool.LpDenom)
		require.Equal(t, sdkmath.NewInt(p.wantBaseR), pool.reserveBase(), "base reserve")
		require.Equal(t, sdkmath.NewInt(p.wantQuote), pool.reserveQuote(), "quote reserve")

		// initial LP = sqrt(base × quote) × 10^6, all of it locked in the black hole
		wantLP := sqrtInt(sdkmath.NewInt(p.initBase).MulRaw(p.initQuote)).MulRaw(lpScale)
		require.Equal(t, wantLP, s.supply(t, pool.LpDenom), "LP supply")
		require.Equal(t, wantLP, s.transferred(t, res.Events, s.tradebin, s.blackHole, pool.LpDenom), "initial LP to the black hole")
		require.Equal(t, s.params.CreateMarketFee.Amount, s.transferred(t, res.Events, s.tradebin, s.cpFees, chain.Denom), "create fee to the community pool collector")

		// the creator pays the reserves, the create fee and the tx fee
		after := s.balances(t, s.lp, chain.Denom, s.usd, s.abc)
		spent := sdk.NewCoins().Add(sdk.NewInt64Coin(p.base, p.initBase), sdk.NewInt64Coin(p.quote, p.initQuote), s.params.CreateMarketFee, sdk.NewCoin(chain.Denom, s.txFee(t, res)))
		require.Equal(t, before.Sub(spent...), after, "lp balances")
	}
}

// 2. An exactly proportional deposit mints the same share of the LP supply.
// A deposit with too much ubze takes only what the pool ratio needs; the rest
// never leaves the user's account.
func (s *ammSuite) addLiquidity(t *testing.T) {
	for _, add := range []struct {
		name      string
		usd, bze  int64
		takenBze  int64
		wantShare int64 // tenths of the pool
	}{
		{"proportional", usdPoolUsd / 10, usdPoolBze / 10, usdPoolBze / 10, 1},
		// reserves are now 5.5e10 usd / 2.2e11 ubze: 5.5e9 usd needs 2.2e10 ubze
		{"too much ubze", usdPoolUsd * 11 / 100, 30_000_000_000, usdPoolBze * 11 / 100, 1},
	} {
		t.Logf("deposit: %s", add.name)
		pool := s.pool(t, s.usdPool)
		supply := s.supply(t, pool.LpDenom)
		before := s.balances(t, s.lp, chain.Denom, s.usd, pool.LpDenom)

		res := s.mustExec(t, s.lp, "tradebin", "add-liquidity", s.usdPool,
			sdkmath.NewInt(add.usd).String(), sdkmath.NewInt(add.bze).String(), "1")

		// minted = supply × deposit / reserve, a tenth of the supply here
		wantMinted := supply.MulRaw(add.wantShare).QuoRaw(10)
		require.Zero(t, supply.MulRaw(add.wantShare).ModRaw(10).Int64(), "the scenario needs an exact tenth")
		s.requireTxEvent(t, res, "LiquidityAddedEvent", map[string]string{
			"pool_id":      s.usdPool,
			"baseAmount":   sdkmath.NewInt(add.usd).String(),
			"quoteAmount":  sdkmath.NewInt(add.takenBze).String(),
			"mintedAmount": wantMinted.String(),
		})

		after := s.pool(t, s.usdPool)
		require.Equal(t, pool.reserveBase().AddRaw(add.usd), after.reserveBase())
		require.Equal(t, pool.reserveQuote().AddRaw(add.takenBze), after.reserveQuote())
		require.Equal(t, supply.Add(wantMinted), s.supply(t, pool.LpDenom))

		want := before.Sub(sdk.NewInt64Coin(s.usd, add.usd), sdk.NewCoin(chain.Denom, s.txFee(t, res).AddRaw(add.takenBze))).
			Add(sdk.NewCoin(pool.LpDenom, wantMinted))
		require.Equal(t, want, s.balances(t, s.lp, chain.Denom, s.usd, pool.LpDenom), "lp balances")
	}
}

// 3. ubze → usd through the usd/ubze pool: exact output, half the pool fee
// burned, the other half left in the reserves, the taker fee to the burner.
func (s *ammSuite) singleSwap(t *testing.T) {
	pool := s.pool(t, s.usdPool)
	in := sdkmath.NewInt(ammSwapInput)
	q := quoteSwap(in, pool.reserveQuote(), pool.reserveBase())
	before := s.balances(t, s.trader, chain.Denom, s.usd)

	res := s.mustExec(t, s.trader, "tradebin", "multi-swap", s.usdPool, sdk.NewCoin(chain.Denom, in).String(), coinStr(1, s.usd))

	s.requireSwap(t, res.Events, s.trader.FormattedAddress(), s.usdPool, sdk.NewCoin(chain.Denom, in), sdk.NewCoin(s.usd, q.out))
	after := s.pool(t, s.usdPool)
	require.Equal(t, pool.reserveQuote().Add(q.inReserveAdd), after.reserveQuote(), "ubze reserve")
	require.Equal(t, pool.reserveBase().Sub(q.out), after.reserveBase(), "usd reserve")

	want := before.Add(sdk.NewCoin(s.usd, q.out)).Sub(sdk.NewCoin(chain.Denom, in.Add(s.params.MarketTakerFee.Amount).Add(s.txFee(t, res))))
	require.Equal(t, want, s.balances(t, s.trader, chain.Denom, s.usd), "trader balances")

	require.Equal(t, q.burned, s.transferred(t, res.Events, s.tradebin, s.burner, chain.Denom), "swap fee to the burner")
	s.requireTakerFeeBurned(t, res)
}

// 4. ubze → usd → abc in one MsgMultiSwap: each hop is exact on its own pool.
// A min output one unit above the result is rejected and moves nothing.
func (s *ammSuite) multiHopSwap(t *testing.T) {
	usdPool, abcPool := s.pool(t, s.usdPool), s.pool(t, s.abcPool)
	in := sdkmath.NewInt(ammSwapInput)
	hop1 := quoteSwap(in, usdPool.reserveQuote(), usdPool.reserveBase())
	hop2 := quoteSwap(hop1.out, abcPool.reserveQuote(), abcPool.reserveBase())
	routes := s.usdPool + "," + s.abcPool
	inCoin := sdk.NewCoin(chain.Denom, in).String()

	before := s.balances(t, s.trader, chain.Denom, s.usd, s.abc)
	res, err := helpers.ExecTxWith(s.ctx, s.bze, s.trader, []helpers.TxOption{helpers.WithGas(ammTxGas)},
		"tradebin", "multi-swap", routes, inCoin, sdk.NewCoin(s.abc, hop2.out.AddRaw(1)).String())
	reason, err := helpers.RejectionReason(res, err)
	require.NoError(t, err)
	require.Contains(t, reason, "expected minimum")
	require.Equal(t, usdPool, s.pool(t, s.usdPool), "usd/ubze pool moved on a rejected swap")
	require.Equal(t, abcPool, s.pool(t, s.abcPool), "abc/usd pool moved on a rejected swap")
	require.Equal(t, before.Sub(sdk.NewCoin(chain.Denom, s.txFee(t, res))), s.balances(t, s.trader, chain.Denom, s.usd, s.abc), "a rejected swap only costs the tx fee")

	before = s.balances(t, s.trader, chain.Denom, s.usd, s.abc)
	res = s.mustExec(t, s.trader, "tradebin", "multi-swap", routes, inCoin, sdk.NewCoin(s.abc, hop2.out).String())

	s.requireSwap(t, res.Events, s.trader.FormattedAddress(), s.usdPool, sdk.NewCoin(chain.Denom, in), sdk.NewCoin(s.usd, hop1.out))
	s.requireSwap(t, res.Events, s.trader.FormattedAddress(), s.abcPool, sdk.NewCoin(s.usd, hop1.out), sdk.NewCoin(s.abc, hop2.out))
	afterUsd, afterAbc := s.pool(t, s.usdPool), s.pool(t, s.abcPool)
	require.Equal(t, usdPool.reserveQuote().Add(hop1.inReserveAdd), afterUsd.reserveQuote())
	require.Equal(t, usdPool.reserveBase().Sub(hop1.out), afterUsd.reserveBase())
	require.Equal(t, abcPool.reserveQuote().Add(hop2.inReserveAdd), afterAbc.reserveQuote())
	require.Equal(t, abcPool.reserveBase().Sub(hop2.out), afterAbc.reserveBase())

	// the intermediate usd never reaches the trader
	want := before.Add(sdk.NewCoin(s.abc, hop2.out)).Sub(sdk.NewCoin(chain.Denom, in.Add(s.params.MarketTakerFee.Amount).Add(s.txFee(t, res))))
	require.Equal(t, want, s.balances(t, s.trader, chain.Denom, s.usd, s.abc), "trader balances")

	require.Equal(t, hop1.burned, s.transferred(t, res.Events, s.tradebin, s.burner, chain.Denom), "hop 1 fee to the burner")
	require.Equal(t, hop2.burned, s.transferred(t, res.Events, s.tradebin, s.burner, s.usd), "hop 2 fee to the burner")
	s.requireTakerFeeBurned(t, res)
}

// 5. Burning a tenth of the LP supply pays out a tenth of each reserve,
// rounded down.
func (s *ammSuite) removeLiquidity(t *testing.T) {
	pool := s.pool(t, s.usdPool)
	supply := s.supply(t, pool.LpDenom)
	require.Zero(t, supply.ModRaw(10).Int64(), "the scenario needs an exact tenth of the LP supply")
	burn := supply.QuoRaw(10)
	// out = ⌊reserve × burn / supply⌋
	wantUsd := pool.reserveBase().Mul(burn).Quo(supply)
	wantBze := pool.reserveQuote().Mul(burn).Quo(supply)
	before := s.balances(t, s.lp, chain.Denom, s.usd, pool.LpDenom)

	res := s.mustExec(t, s.lp, "tradebin", "remove-liquidity", s.usdPool, burn.String(), wantUsd.String(), wantBze.String())

	s.requireTxEvent(t, res, "LiquidityRemovedEvent", map[string]string{
		"pool_id": s.usdPool, "baseAmount": wantUsd.String(), "quoteAmount": wantBze.String(),
	})
	after := s.pool(t, s.usdPool)
	require.Equal(t, pool.reserveBase().Sub(wantUsd), after.reserveBase())
	require.Equal(t, pool.reserveQuote().Sub(wantBze), after.reserveQuote())
	require.Equal(t, supply.Sub(burn), s.supply(t, pool.LpDenom), "LP supply")

	want := before.Add(sdk.NewCoin(s.usd, wantUsd), sdk.NewCoin(chain.Denom, wantBze)).Sub(sdk.NewCoin(pool.LpDenom, burn), sdk.NewCoin(chain.Denom, s.txFee(t, res)))
	require.Equal(t, want, s.balances(t, s.lp, chain.Denom, s.usd, pool.LpDenom), "lp balances")
}

// 6. A tx fee in usd goes to the txfeecollector, which swaps all of it to
// ubze through the usd/ubze pool in the same block's EndBlock and sends the
// ubze to the validators' fee collector. A denom without a ubze pool can not
// pay fees.
func (s *ammSuite) txFeeInUsd(t *testing.T) {
	require.True(t, s.balance(t, s.txFees, s.usd).IsZero(), "txfeecollector holds usd before the test")
	pool := s.pool(t, s.usdPool)
	fee := sdkmath.NewInt(ammUsdFee)
	q := quoteSwap(fee, pool.reserveBase(), pool.reserveQuote())
	before := s.balances(t, s.trader, chain.Denom, s.usd)

	res := s.mustExecWith(t, s.trader, []helpers.TxOption{helpers.WithFees(sdk.NewCoin(s.usd, fee).String()), helpers.WithGas(ammSendGas)},
		"bank", "send", s.trader.FormattedAddress(), s.lp.FormattedAddress(), coinStr(1, chain.Denom))

	paid, err := helpers.TxFee(res, s.usd)
	require.NoError(t, err)
	require.Equal(t, fee, paid)
	require.Equal(t, fee, s.transferred(t, res.Events, s.trader.FormattedAddress(), s.txFees, s.usd), "the usd fee goes to the txfeecollector")
	require.Equal(t, before.Sub(sdk.NewCoin(s.usd, fee), sdk.NewInt64Coin(chain.Denom, 1)), s.balances(t, s.trader, chain.Denom, s.usd), "trader pays no ubze fee")

	block := s.blockEvents(t, res.Height)
	s.requireSwap(t, block, s.txFees, s.usdPool, sdk.NewCoin(s.usd, fee), sdk.NewCoin(chain.Denom, q.out))
	require.Equal(t, q.out, s.transferred(t, block, s.txFees, s.feeCollector, chain.Denom), "swapped ubze to the fee collector")
	require.Equal(t, q.burned, s.transferred(t, block, s.tradebin, s.burner, s.usd), "swap fee to the burner")
	require.True(t, s.balance(t, s.txFees, s.usd).IsZero(), "txfeecollector keeps usd after EndBlock")

	after := s.pool(t, s.usdPool)
	require.Equal(t, pool.reserveBase().Add(q.inReserveAdd), after.reserveBase(), "usd reserve")
	require.Equal(t, pool.reserveQuote().Sub(q.out), after.reserveQuote(), "ubze reserve")

	// abc has a pool with usd only, none with ubze
	res, err = helpers.ExecTxWith(s.ctx, s.bze, s.trader, []helpers.TxOption{helpers.WithFees(coinStr(ammUsdFee, s.abc)), helpers.WithGas(ammSendGas)},
		"bank", "send", s.trader.FormattedAddress(), s.lp.FormattedAddress(), coinStr(1, chain.Denom))
	reason, err := helpers.RejectionReason(res, err)
	require.NoError(t, err)
	require.Contains(t, reason, "can be used to pay for fees only if enough liquidity is available")
}

// 7. A taker paying its tx fee in usd also pays the 0.1 BZE taker fee in
// usd: the chain takes the usd that buys exactly the taker fee from the
// usd/ubze pool, swaps it and sends the ubze to the burner fee collector,
// which hands it to the burner in EndBlock. The tx fee is swapped after it.
func (s *ammSuite) takerFeeInUsd(t *testing.T) {
	s.mustExecWith(t, s.lp, []helpers.TxOption{helpers.WithGas(ammTxGas)}, "tradebin", "create-market", s.abc, s.usd)
	const price = 5
	s.mustExec(t, s.maker, "tradebin", "create-order", "sell", sdkmath.NewInt(ammOrderAmount).String(), sdkmath.NewInt(price).String(), s.market)
	makerUsd := s.balance(t, s.maker.FormattedAddress(), s.usd)

	pool := s.pool(t, s.usdPool)
	takerFee := s.params.MarketTakerFee.Amount
	req := inputFor(takerFee, pool.reserveBase(), pool.reserveQuote())
	feeSwap := quoteSwap(req, pool.reserveBase(), pool.reserveQuote())
	// then the EndBlock swaps the tx fee, on the reserves the fee swap left
	txFee := sdkmath.NewInt(ammUsdFee)
	txSwap := quoteSwap(txFee, pool.reserveBase().Add(feeSwap.inReserveAdd), pool.reserveQuote().Sub(feeSwap.out))
	before := s.balances(t, s.taker, chain.Denom, s.usd, s.abc)

	res := s.mustExecWith(t, s.taker, []helpers.TxOption{helpers.WithFees(sdk.NewCoin(s.usd, txFee).String()), helpers.WithGas(ammTxGas)},
		"tradebin", "create-order", "buy", sdkmath.NewInt(ammOrderAmount).String(), sdkmath.NewInt(price).String(), s.market)

	cost := sdkmath.NewInt(ammOrderAmount * price)
	want := before.Add(sdk.NewInt64Coin(s.abc, ammOrderAmount)).Sub(sdk.NewCoin(s.usd, txFee.Add(req).Add(cost)))
	require.Equal(t, want, s.balances(t, s.taker, chain.Denom, s.usd, s.abc), "taker balances (no ubze spent)")
	require.Equal(t, makerUsd.Add(cost), s.balance(t, s.maker.FormattedAddress(), s.usd), "maker paid in usd")

	require.Equal(t, feeSwap.out, s.transferred(t, res.Events, s.tradebin, s.burnerFees, chain.Denom), "swapped taker fee to the burner fee collector")
	require.Equal(t, feeSwap.burned, s.transferred(t, res.Events, s.tradebin, s.burner, s.usd), "fee-swap pool fee to the burner")

	block := s.blockEvents(t, res.Height)
	require.Equal(t, feeSwap.out, s.transferred(t, block, s.burnerFees, s.burner, chain.Denom), "burner fee collector to the burner")
	s.requireSwap(t, block, s.txFees, s.usdPool, sdk.NewCoin(s.usd, txFee), sdk.NewCoin(chain.Denom, txSwap.out))
	require.Equal(t, txSwap.out, s.transferred(t, block, s.txFees, s.feeCollector, chain.Denom), "swapped tx fee to the fee collector")
	require.True(t, s.balance(t, s.burnerFees, chain.Denom).IsZero(), "burner fee collector keeps ubze after EndBlock")

	after := s.pool(t, s.usdPool)
	require.Equal(t, pool.reserveBase().Add(feeSwap.inReserveAdd).Add(txSwap.inReserveAdd), after.reserveBase(), "usd reserve")
	require.Equal(t, pool.reserveQuote().Sub(feeSwap.out).Sub(txSwap.out), after.reserveQuote(), "ubze reserve")
}

// 8. `--gas auto` simulates with a zero fee; the txfeecollector must accept
// that (fixed in #117) and, for a fee in a factory token, run the same
// fee-denom checks as the real tx so the estimate covers their gas.
func (s *ammSuite) gasAuto(t *testing.T) {
	for _, c := range []struct {
		name, prices string
	}{
		{"ubze", chain.GasPrices},
		{"factory token", "0.01" + s.usd},
	} {
		t.Run(c.name, func(t *testing.T) {
			res, err := helpers.ExecTxWith(s.ctx, s.bze, s.trader, []helpers.TxOption{helpers.WithGasPrices(c.prices)},
				"bank", "send", s.trader.FormattedAddress(), s.lp.FormattedAddress(), coinStr(1, chain.Denom))
			require.NoError(t, err, "gas auto with --gas-prices %s", c.prices)
			t.Logf("gas wanted %d (estimate × %.1f), used %d", res.GasWanted, chain.GasAdjustment, res.GasUsed)
			require.Zero(t, res.Code, res.RawLog)

			// the fee is the estimate at the given prices, rounded up
			wantFee, err := helpers.FeeForGas(c.prices, uint64(res.GasWanted))
			require.NoError(t, err)
			for _, fee := range wantFee {
				paid, err := helpers.TxFee(res, fee.Denom)
				require.NoError(t, err)
				require.Equal(t, fee.Amount, paid, "fee in %s", fee.Denom)
			}
		})
	}
}

// 9. One MsgFillOrders filling three sell orders queues three messages, each
// with its own message_id; the EndBlock executes them and every
// OrderExecutedEvent carries the id of the message it executed. A cancel
// carries its message id from the tx to the EndBlock cancel too.
func (s *ammSuite) fillOrders(t *testing.T) {
	prices := []int64{2, 3, 4}
	makerBefore := s.balances(t, s.maker, chain.Denom, s.usd, s.abc)
	makerFees := sdkmath.ZeroInt()
	for _, p := range prices {
		res := s.mustExec(t, s.maker, "tradebin", "create-order", "sell", sdkmath.NewInt(ammOrderAmount).String(), sdkmath.NewInt(p).String(), s.market)
		makerFees = makerFees.Add(s.params.MarketMakerFee.Amount).Add(s.txFee(t, res))
	}

	orders := make([]map[string]string, 0, len(prices))
	cost := sdkmath.ZeroInt()
	for _, p := range prices {
		orders = append(orders, map[string]string{"price": sdkmath.NewInt(p).String(), "amount": sdkmath.NewInt(ammOrderAmount).String()})
		cost = cost.AddRaw(ammOrderAmount * p)
	}
	takerBefore := s.balances(t, s.taker, chain.Denom, s.usd, s.abc)
	res, err := helpers.ExecMsgs(s.ctx, s.bze, s.taker, []helpers.TxOption{helpers.WithGas(ammFillGas)}, map[string]any{
		"@type":      "/bze.tradebin.MsgFillOrders",
		"creator":    s.taker.FormattedAddress(),
		"market_id":  s.market,
		"order_type": "sell",
		"orders":     orders,
	})
	require.NoError(t, err)
	require.Zero(t, res.Code, res.RawLog)

	// filling sell orders queues buy messages, one per order, with distinct ids
	created := helpers.TxEvents(res, tradebinEventPrefix+"OrderCreateMessageEvent")
	require.Len(t, created, len(prices))
	ids := map[string]string{} // price → message id
	for _, e := range created {
		assert.Equal(t, "buy", e.Attributes["order_type"])
		assert.Equal(t, s.market, e.Attributes["market_id"])
		require.Len(t, e.Attributes["message_id"], 24, "message_id is 24 zero-filled digits")
		ids[e.Attributes["price"]] = e.Attributes["message_id"]
	}
	require.Len(t, ids, len(prices), "one message per price")
	require.Len(t, uniq(ids), len(prices), "message ids are distinct")

	executed := helpers.EventsWith(s.blockEvents(t, res.Height), map[string]string{"market_id": s.market, "taker": s.taker.FormattedAddress()})
	executed = filterType(executed, tradebinEventPrefix+"OrderExecutedEvent")
	require.Len(t, executed, len(prices), "one execution per filled order")
	for _, e := range executed {
		assert.Equal(t, ids[e.Attributes["price"]], e.Attributes["message_id"], "execution at %s carries its message id", e.Attributes["price"])
		assert.Equal(t, sdkmath.NewInt(ammOrderAmount).String(), e.Attributes["amount"])
		assert.Equal(t, s.maker.FormattedAddress(), e.Attributes["maker"])
	}

	takerFee := s.params.MarketTakerFee.Amount.Add(s.txFee(t, res))
	wantTaker := takerBefore.Add(sdk.NewInt64Coin(s.abc, ammOrderAmount*int64(len(prices)))).Sub(sdk.NewCoin(s.usd, cost), sdk.NewCoin(chain.Denom, takerFee))
	require.Equal(t, wantTaker, s.balances(t, s.taker, chain.Denom, s.usd, s.abc), "taker balances")
	wantMaker := makerBefore.Add(sdk.NewCoin(s.usd, cost)).Sub(sdk.NewInt64Coin(s.abc, ammOrderAmount*int64(len(prices))), sdk.NewCoin(chain.Denom, makerFees))
	require.Equal(t, wantMaker, s.balances(t, s.maker, chain.Denom, s.usd, s.abc), "maker balances")

	s.createAndCancel(t)
}

// createAndCancel places a sell order nobody fills and cancels it: the
// cancel's message id goes from the tx event to the EndBlock event.
func (s *ammSuite) createAndCancel(t *testing.T) {
	before := s.balance(t, s.maker.FormattedAddress(), s.abc)
	res := s.mustExec(t, s.maker, "tradebin", "create-order", "sell", sdkmath.NewInt(ammOrderAmount).String(), "10", s.market)
	createID := s.requireTxEvent(t, res, "OrderCreateMessageEvent", map[string]string{"market_id": s.market, "price": "10"}).Attributes["message_id"]
	saved := helpers.EventsWith(filterType(s.blockEvents(t, res.Height), tradebinEventPrefix+"OrderSavedEvent"), map[string]string{"message_id": createID})
	require.Len(t, saved, 1, "the order is saved by the EndBlock that processed message %s", createID)
	orderID := saved[0].Attributes["id"]
	require.Equal(t, before.SubRaw(ammOrderAmount), s.balance(t, s.maker.FormattedAddress(), s.abc), "order amount locked")

	res = s.mustExec(t, s.maker, "tradebin", "cancel-order", s.market, orderID, "sell")
	cancelID := s.requireTxEvent(t, res, "OrderCancelMessageEvent", map[string]string{"orderId": orderID}).Attributes["message_id"]
	// ids are unique only while queued: the counter restarts once the queue
	// empties, so the cancel may reuse the create's id
	require.Len(t, cancelID, 24)
	canceled := helpers.EventsWith(filterType(s.blockEvents(t, res.Height), tradebinEventPrefix+"OrderCanceledEvent"), map[string]string{"id": orderID})
	require.Len(t, canceled, 1)
	assert.Equal(t, cancelID, canceled[0].Attributes["message_id"])
	require.Equal(t, before, s.balance(t, s.maker.FormattedAddress(), s.abc), "cancel refunds the order amount")
}

// 10. usd → ubze through the usd/ubze pool, the base-to-quote direction:
// exact output, half the pool fee (in usd) burned, the taker fee to the burner.
func (s *ammSuite) swapBaseToQuote(t *testing.T) {
	pool := s.pool(t, s.usdPool)
	in := sdkmath.NewInt(ammBaseSwapInput)
	q := quoteSwap(in, pool.reserveBase(), pool.reserveQuote())
	before := s.balances(t, s.trader, chain.Denom, s.usd)

	res := s.mustExec(t, s.trader, "tradebin", "multi-swap", s.usdPool, sdk.NewCoin(s.usd, in).String(), coinStr(1, chain.Denom))

	s.requireSwap(t, res.Events, s.trader.FormattedAddress(), s.usdPool, sdk.NewCoin(s.usd, in), sdk.NewCoin(chain.Denom, q.out))
	after := s.pool(t, s.usdPool)
	require.Equal(t, pool.reserveBase().Add(q.inReserveAdd), after.reserveBase(), "usd reserve")
	require.Equal(t, pool.reserveQuote().Sub(q.out), after.reserveQuote(), "ubze reserve")

	want := before.Add(sdk.NewCoin(chain.Denom, q.out)).Sub(sdk.NewCoin(s.usd, in), sdk.NewCoin(chain.Denom, s.params.MarketTakerFee.Amount.Add(s.txFee(t, res))))
	require.Equal(t, want, s.balances(t, s.trader, chain.Denom, s.usd), "trader balances")

	require.Equal(t, q.burned, s.transferred(t, res.Events, s.tradebin, s.burner, s.usd), "swap fee to the burner")
	s.requireTakerFeeBurned(t, res)
}

// 11. A deposit with twice the usd the ubze needs: the chain takes
// ⌊ubze × usd reserve / ubze reserve⌋ usd and all the ubze, mints on the
// smaller of the two ratios and leaves the excess usd with the user.
func (s *ammSuite) addExcessBase(t *testing.T) {
	pool := s.pool(t, s.usdPool)
	supply := s.supply(t, pool.LpDenom)
	bze := sdkmath.NewInt(ammExcessBze)
	usd := bze.Mul(pool.reserveBase()).Quo(pool.reserveQuote()).MulRaw(2).AddRaw(12_345)
	d := ammDeposit(usd, bze, pool.reserveBase(), pool.reserveQuote(), supply)
	require.Equal(t, bze, d.quote, "all the ubze is taken")
	require.True(t, d.base.LT(usd), "only part of the usd is taken")
	before := s.balances(t, s.lp, chain.Denom, s.usd, pool.LpDenom)

	res := s.mustExec(t, s.lp, "tradebin", "add-liquidity", s.usdPool, usd.String(), bze.String(), "1")

	s.requireTxEvent(t, res, "LiquidityAddedEvent", map[string]string{
		"pool_id":      s.usdPool,
		"baseAmount":   d.base.String(),
		"quoteAmount":  d.quote.String(),
		"mintedAmount": d.minted.String(),
	})
	after := s.pool(t, s.usdPool)
	require.Equal(t, pool.reserveBase().Add(d.base), after.reserveBase())
	require.Equal(t, pool.reserveQuote().Add(d.quote), after.reserveQuote())
	require.Equal(t, supply.Add(d.minted), s.supply(t, pool.LpDenom))
	require.Equal(t, d.base, s.transferred(t, res.Events, s.lp.FormattedAddress(), s.tradebin, s.usd), "usd taken")

	want := before.Sub(sdk.NewCoin(s.usd, d.base), sdk.NewCoin(chain.Denom, s.txFee(t, res).Add(d.quote))).
		Add(sdk.NewCoin(pool.LpDenom, d.minted))
	require.Equal(t, want, s.balances(t, s.lp, chain.Denom, s.usd, pool.LpDenom), "lp balances: the excess usd stays")
}

// 12. A pool created as ubze/rev is stored as rev/ubze (denoms sorted, the
// amounts swapped with them), under that id only; the 0.05 fee bound is
// accepted.
func (s *ammSuite) reversedPool(t *testing.T) {
	s.mustExec(t, s.lp, "tokenfactory", "create-denom", "rev")
	s.mustExec(t, s.lp, "tokenfactory", "mint", coinStr(ammMint, s.rev))

	s.createPool(t, ammPoolArgs{
		base: chain.Denom, quote: s.rev, initBase: revPoolBze, initQuote: revPoolRev,
		fee: ammMaxPoolFee, feeDest: ammFeeDest,
		wantID: s.revPool, wantBase: s.rev, wantQuote: chain.Denom, wantReserveBase: revPoolRev, wantReserveQuote: revPoolBze,
	})
	s.requireNoPool(t, chain.Denom+"_"+s.rev)
}

// 13. The LP denom of a new pool is ulp/<uppercase hex sha256(pool id)>, with
// bank metadata: base unit ulp/<hash> at exponent 0, display unit lp/<hash>
// at exponent 6, no name or symbol.
func (s *ammSuite) lpMetadata(t *testing.T) {
	sum := sha256.Sum256([]byte(s.revPool))
	hash := strings.ToUpper(hex.EncodeToString(sum[:]))
	base, display := "ulp/"+hash, "lp/"+hash
	require.Len(t, base, 68)
	require.Equal(t, base, s.pool(t, s.revPool).LpDenom, "pool lp denom")

	res, err := helpers.Query[struct {
		Metadata struct {
			Name       string `json:"name"`
			Symbol     string `json:"symbol"`
			Display    string `json:"display"`
			Base       string `json:"base"`
			DenomUnits []struct {
				Denom    string        `json:"denom"`
				Exponent helpers.Int64 `json:"exponent"`
			} `json:"denom_units"`
		} `json:"metadata"`
	}](s.ctx, s.bze, "bank", "denom-metadata", base)
	require.NoError(t, err)
	md := res.Metadata
	assert.Equal(t, base, md.Base)
	assert.Equal(t, display, md.Display)
	assert.Empty(t, md.Name)
	assert.Empty(t, md.Symbol)
	require.Len(t, md.DenomUnits, 2, "%+v", md.DenomUnits)
	assert.Equal(t, base, md.DenomUnits[0].Denom)
	assert.EqualValues(t, 0, md.DenomUnits[0].Exponent)
	assert.Equal(t, display, md.DenomUnits[1].Denom)
	assert.EqualValues(t, 6, md.DenomUnits[1].Exponent)
}

// 14. Slippage guards: a MinLpTokens one above what the deposit mints and a
// MinBase or MinQuote one above what the burn pays are refused and move
// nothing but the tx fee; the exact amounts then pass. The chain's decimal
// math is replayed with LegacyDec (ammDeposit, ammWithdraw).
func (s *ammSuite) slippageGuards(t *testing.T) {
	addMsg := func(base, quote, minLp sdkmath.Int) map[string]any {
		return map[string]any{
			"@type": "/bze.tradebin.MsgAddLiquidity", "creator": s.lp.FormattedAddress(), "pool_id": s.usdPool,
			"base_amount": base.String(), "quote_amount": quote.String(), "min_lp_tokens": minLp.String(),
		}
	}
	removeMsg := func(lp, minBase, minQuote sdkmath.Int) map[string]any {
		return map[string]any{
			"@type": "/bze.tradebin.MsgRemoveLiquidity", "creator": s.lp.FormattedAddress(), "pool_id": s.usdPool,
			"lp_tokens": lp.String(), "min_base": minBase.String(), "min_quote": minQuote.String(),
		}
	}

	pool := s.pool(t, s.usdPool)
	supply := s.supply(t, pool.LpDenom)
	usd := sdkmath.NewInt(ammGuardUsd)
	bze := usd.Mul(pool.reserveQuote()).Quo(pool.reserveBase()).AddRaw(1_000)
	d := ammDeposit(usd, bze, pool.reserveBase(), pool.reserveQuote(), supply)
	over := d.minted.AddRaw(1)
	s.requireRefusals(t, s.lp, []string{chain.Denom, s.usd, pool.LpDenom}, []ammRefusal{
		{"add, min LP one above the minted", addMsg(usd, bze, over), 4014, ammCodespace,
			fmt.Sprintf("Minted %s, expected minimum: %s", d.minted, over), false},
	})
	before := s.balances(t, s.lp, chain.Denom, s.usd, pool.LpDenom)
	res := s.mustExecMsgs(t, s.lp, addMsg(usd, bze, d.minted))
	s.requireTxEvent(t, res, "LiquidityAddedEvent", map[string]string{
		"pool_id": s.usdPool, "baseAmount": d.base.String(), "quoteAmount": d.quote.String(), "mintedAmount": d.minted.String(),
	})
	want := before.Sub(sdk.NewCoin(s.usd, d.base), sdk.NewCoin(chain.Denom, d.quote.Add(s.txFee(t, res)))).Add(sdk.NewCoin(pool.LpDenom, d.minted))
	require.Equal(t, want, s.balances(t, s.lp, chain.Denom, s.usd, pool.LpDenom), "lp balances")

	pool = s.pool(t, s.usdPool)
	supply = s.supply(t, pool.LpDenom)
	burn := s.balance(t, s.lp.FormattedAddress(), pool.LpDenom).QuoRaw(7)
	outUsd, outBze := ammWithdraw(burn, supply, pool.reserveBase(), pool.reserveQuote())
	s.requireRefusals(t, s.lp, []string{chain.Denom, s.usd, pool.LpDenom}, []ammRefusal{
		{"remove, min base one above the paid", removeMsg(burn, outUsd.AddRaw(1), outBze), 4014, ammCodespace,
			fmt.Sprintf("base amount too low, got %s, minimum %s", outUsd, outUsd.AddRaw(1)), false},
		{"remove, min quote one above the paid", removeMsg(burn, outUsd, outBze.AddRaw(1)), 4014, ammCodespace,
			fmt.Sprintf("quote amount too low, got %s, minimum %s", outBze, outBze.AddRaw(1)), false},
	})
	before = s.balances(t, s.lp, chain.Denom, s.usd, pool.LpDenom)
	res = s.mustExecMsgs(t, s.lp, removeMsg(burn, outUsd, outBze))
	s.requireTxEvent(t, res, "LiquidityRemovedEvent", map[string]string{
		"pool_id": s.usdPool, "baseAmount": outUsd.String(), "quoteAmount": outBze.String(),
	})
	after := s.pool(t, s.usdPool)
	require.Equal(t, pool.reserveBase().Sub(outUsd), after.reserveBase())
	require.Equal(t, pool.reserveQuote().Sub(outBze), after.reserveQuote())
	require.Equal(t, supply.Sub(burn), s.supply(t, pool.LpDenom), "LP supply")
	want = before.Add(sdk.NewCoin(s.usd, outUsd), sdk.NewCoin(chain.Denom, outBze)).Sub(sdk.NewCoin(pool.LpDenom, burn), sdk.NewCoin(chain.Denom, s.txFee(t, res)))
	require.Equal(t, want, s.balances(t, s.lp, chain.Denom, s.usd, pool.LpDenom), "lp balances")
}

// 15. abc/ubze with fee 0.001 (the lower bound) and fee destination
// treasury 0.3 / burner 0.2 / providers 0.5. A ubze → abc swap sends
// ⌊⌊in × 0.001⌋ × 0.3⌋ ubze to txfeecollector_cp, which funds the community
// pool with it in EndBlock, and ⌊⌊in × 0.001⌋ × 0.2⌋ to the burner; the rest
// of the fee stays in the reserves.
func (s *ammSuite) treasuryShare(t *testing.T) {
	pool := s.createPool(t, ammPoolArgs{
		base: s.abc, quote: chain.Denom, initBase: cpPoolAbc, initQuote: cpPoolBze,
		fee: ammMinPoolFee, feeDest: ammTreasuryFeeDest,
		wantID: s.cpPool, wantBase: s.abc, wantQuote: chain.Denom, wantReserveBase: cpPoolAbc, wantReserveQuote: cpPoolBze,
	})

	in := sdkmath.NewInt(ammTreasurySwapInput)
	q := ammQuoteSwap(in, pool.reserveQuote(), pool.reserveBase(), ammMinPoolFee, "0.3", "0.2")
	// the scenario's numbers: fee 123,456, treasury 37,036 (37,036.8), burner 24,691 (24,691.2)
	require.Equal(t, sdkmath.NewInt(123_456), q.fee)
	require.Equal(t, sdkmath.NewInt(37_036), q.treasury)
	require.Equal(t, sdkmath.NewInt(24_691), q.burned)
	before := s.balances(t, s.trader, chain.Denom, s.abc)

	res := s.mustExec(t, s.trader, "tradebin", "multi-swap", s.cpPool, sdk.NewCoin(chain.Denom, in).String(), coinStr(1, s.abc))

	s.requireSwap(t, res.Events, s.trader.FormattedAddress(), s.cpPool, sdk.NewCoin(chain.Denom, in), sdk.NewCoin(s.abc, q.out))
	require.Equal(t, q.treasury, s.transferred(t, res.Events, s.tradebin, s.cpFees, chain.Denom), "treasury share to txfeecollector_cp")
	require.Equal(t, q.burned, s.transferred(t, res.Events, s.tradebin, s.burner, chain.Denom), "burner share to the burner")
	s.requireTakerFeeBurned(t, res)

	after := s.pool(t, s.cpPool)
	require.Equal(t, pool.reserveQuote().Add(q.inReserveAdd), after.reserveQuote(), "ubze reserve keeps the providers share")
	require.Equal(t, pool.reserveBase().Sub(q.out), after.reserveBase(), "abc reserve")
	want := before.Add(sdk.NewCoin(s.abc, q.out)).Sub(sdk.NewCoin(chain.Denom, in.Add(s.params.MarketTakerFee.Amount).Add(s.txFee(t, res))))
	require.Equal(t, want, s.balances(t, s.trader, chain.Denom, s.abc), "trader balances")

	block := s.blockEvents(t, res.Height)
	require.Equal(t, q.treasury, s.transferred(t, block, s.cpFees, s.distribution, chain.Denom), "treasury share to the community pool in EndBlock")
	require.True(t, s.balance(t, s.cpFees, chain.Denom).IsZero(), "txfeecollector_cp keeps ubze after EndBlock")
}

// 16. Pools the msg server or ValidateBasic refuse: nothing moves but the tx
// fee (none when CheckTx refused it), no pool appears. A refused fee or fee
// destination is tried on a pair with no pool, so the duplicate check passes.
func (s *ammSuite) createPoolRefusals(t *testing.T) {
	noSupply := "factory/" + s.lp.FormattedAddress() + "/nosupply"
	pool := func(base, quote, fee, feeDest string, stable bool, initBase int64) map[string]any {
		return map[string]any{
			"@type": "/bze.tradebin.MsgCreateLiquidityPool", "creator": s.lp.FormattedAddress(),
			"base": base, "quote": quote, "fee": fee, "fee_dest": feeDest, "stable": stable,
			"initial_base": sdkmath.NewInt(initBase).String(), "initial_quote": sdkmath.NewInt(revPoolRev).String(),
		}
	}
	fresh := func(fee, feeDest string) map[string]any {
		return pool(s.rev, s.usd, fee, feeDest, false, revPoolRev)
	}
	s.requireRefusals(t, s.lp, []string{chain.Denom, s.usd, s.rev}, []ammRefusal{
		{"duplicate pool", pool(s.usd, chain.Denom, ammPoolFee, ammFeeDest, false, revPoolRev), 4000, ammCodespace, "market already exists", false},
		{"duplicate pool, reversed", pool(chain.Denom, s.usd, ammPoolFee, ammFeeDest, false, revPoolRev), 4000, ammCodespace, "market already exists", false},
		{"fee below 0.001", fresh("0.0009", ammFeeDest), 10, "sdk", "fee must be at least 0.001 (0.1%)", false},
		{"fee above 0.05", fresh("0.0501", ammFeeDest), 10, "sdk", "fee must be at most 0.05 (5%)", false},
		{"fee destination sums to 0.9", fresh(ammPoolFee, `{"treasury":"0","burner":"0.5","providers":"0.4"}`), 4012, ammCodespace, "invalid fee destination", false},
		{"fee destination sums to 1.1", fresh(ammPoolFee, `{"treasury":"0.1","burner":"0.5","providers":"0.5"}`), 4012, ammCodespace, "invalid fee destination", false},
		{"negative fee destination", fresh(ammPoolFee, `{"treasury":"-0.1","burner":"0.6","providers":"0.5"}`), 4013, ammCodespace, "negative fee destination", false},
		{"stable pool", pool(s.rev, s.usd, ammPoolFee, ammFeeDest, true, revPoolRev), 18, "sdk", "stable pools are not supported yet", true},
		{"denom without supply", pool(noSupply, s.usd, ammPoolFee, ammFeeDest, false, revPoolRev), 4001, ammCodespace, "no supply found for the provided denom", false},
		{"base equals quote", pool(s.usd, s.usd, ammPoolFee, ammFeeDest, false, revPoolRev), 18, "sdk", "base and quote cannot be the same", true},
		{"no initial liquidity", pool(s.rev, s.usd, ammPoolFee, ammFeeDest, false, 0), 18, "sdk", "missing initial liquidity", true},
	})
	s.requireNoPool(t, s.rev+"_"+s.usd)
	s.requireNoPool(t, noSupply+"_"+s.usd)
}

// 17. Swaps the msg server or ValidateBasic refuse: the input never leaves
// the trader, no pool moves and no taker fee is charged.
func (s *ammSuite) swapRefusals(t *testing.T) {
	swap := func(routes []string, in, minOut sdk.Coin) map[string]any {
		return map[string]any{
			"@type": "/bze.tradebin.MsgMultiSwap", "creator": s.trader.FormattedAddress(), "routes": routes,
			"input":      map[string]string{"denom": in.Denom, "amount": in.Amount.String()},
			"min_output": map[string]string{"denom": minOut.Denom, "amount": minOut.Amount.String()},
		}
	}
	bze := func(amount int64) sdk.Coin { return sdk.NewInt64Coin(chain.Denom, amount) }
	oneUsd := sdk.NewInt64Coin(s.usd, 1)
	tooMany := []string{s.usdPool, s.abcPool, s.revPool, s.cpPool, "x_" + chain.Denom, "y_" + chain.Denom}
	require.Len(t, tooMany, ammMaxRoutes+1)
	s.requireRefusals(t, s.trader, []string{chain.Denom, s.usd, s.abc}, []ammRefusal{
		// ⌊1 × 0.002⌋ = 0: a swap must pay a positive pool fee
		{"amount too small", swap([]string{s.usdPool}, bze(1), oneUsd), 4016, ammCodespace, "amount is too low to be traded", false},
		{"input denom not in the first pool", swap([]string{s.abcPool}, bze(1_000_000), oneUsd), 4016, ammCodespace,
			fmt.Sprintf("denom %s does not exist in pool %s", chain.Denom, s.abcPool), false},
		{"output denom is not the min output denom", swap([]string{s.usdPool}, bze(1_000_000), sdk.NewInt64Coin(s.abc, 1)), 4016, ammCodespace,
			fmt.Sprintf("expected %s output, got %s", s.abc, s.usd), false},
		{"unknown pool", swap([]string{"x_" + chain.Denom}, bze(1_000_000), oneUsd), 4015, ammCodespace, "pool x_" + chain.Denom + " not found", false},
		{"duplicate pool in routes", swap([]string{s.usdPool, s.usdPool}, bze(1_000_000), oneUsd), 4015, ammCodespace, "duplicate route " + s.usdPool, true},
		{"more than MaxRoutes", swap(tooMany, bze(1_000_000), oneUsd), 4015, ammCodespace, "routes length must be between 1 and 5", true},
		{"no routes", swap([]string{}, bze(1_000_000), oneUsd), 4015, ammCodespace, "routes length must be between 1 and 5", true},
		{"insufficient balance", swap([]string{s.usdPool}, bze(ammLpFunds), oneUsd), 10, "sdk", "could not capture user input coins", false},
	})
}

// 18. Markets the msg server or ValidateBasic refuse: the existing abc/usd,
// also asked as usd/abc (the alias index), base equal to quote and a denom
// without supply. No market appears and no create fee is charged.
func (s *ammSuite) createMarketRefusals(t *testing.T) {
	market := func(base, quote string) map[string]any {
		return map[string]any{"@type": "/bze.tradebin.MsgCreateMarket", "creator": s.lp.FormattedAddress(), "base": base, "quote": quote}
	}
	s.requireRefusals(t, s.lp, []string{chain.Denom, s.usd, s.abc}, []ammRefusal{
		{"duplicate market", market(s.abc, s.usd), 4000, ammCodespace, "market already exists", false},
		{"duplicate market, reversed", market(s.usd, s.abc), 4000, ammCodespace, "market already exists", false},
		{"base equals quote", market(s.usd, s.usd), 4010, ammCodespace, "invalid denom provided", true},
		{"denom without supply", market("factory/"+s.lp.FormattedAddress()+"/nosupply", s.usd), 4001, ammCodespace, "no supply found for the provided denom", false},
	})
}

// ammPoolArgs is a create-liquidity-pool as sent and the pool the chain
// should store for it.
type ammPoolArgs struct {
	base, quote                       string
	initBase, initQuote               int64
	fee, feeDest                      string
	wantID, wantBase, wantQuote       string
	wantReserveBase, wantReserveQuote int64
}

// createPool creates a pool as the lp and asserts it like step 1: id, sorted
// denoms, reserves, fee and fee destination, the initial LP locked in the
// black hole, the create fee to the community pool collector and the lp's
// spending.
func (s *ammSuite) createPool(t *testing.T, a ammPoolArgs) poolInfo {
	t.Helper()
	// one of the pair is ubze in this suite; balances must not count it twice
	denoms := []string{chain.Denom}
	for _, d := range []string{a.base, a.quote} {
		if d != chain.Denom {
			denoms = append(denoms, d)
		}
	}
	before := s.balances(t, s.lp, denoms...)
	res := s.mustExecWith(t, s.lp, []helpers.TxOption{helpers.WithGas(ammTxGas)},
		"tradebin", "create-liquidity-pool", a.base, a.quote, a.fee, a.feeDest, "false",
		sdkmath.NewInt(a.initBase).String(), sdkmath.NewInt(a.initQuote).String())

	ev := s.requireTxEvent(t, res, "PoolCreatedEvent", map[string]string{"creator": s.lp.FormattedAddress(), "base": a.wantBase, "quote": a.wantQuote})
	pool := s.pool(t, a.wantID)
	require.Equal(t, a.wantBase, pool.Base)
	require.Equal(t, a.wantQuote, pool.Quote)
	require.Equal(t, ev.Attributes["lp_denom"], pool.LpDenom)
	require.Equal(t, sdkmath.NewInt(a.wantReserveBase), pool.reserveBase(), "base reserve")
	require.Equal(t, sdkmath.NewInt(a.wantReserveQuote), pool.reserveQuote(), "quote reserve")
	ammRequireDec(t, a.fee, pool.Fee, "pool fee")
	var dest map[string]string
	require.NoError(t, json.Unmarshal([]byte(a.feeDest), &dest))
	ammRequireDec(t, dest["treasury"], pool.FeeDest.Treasury, "treasury share")
	ammRequireDec(t, dest["burner"], pool.FeeDest.Burner, "burner share")
	ammRequireDec(t, dest["providers"], pool.FeeDest.Providers, "providers share")

	wantLP := sqrtInt(sdkmath.NewInt(a.initBase).MulRaw(a.initQuote)).MulRaw(lpScale)
	require.Equal(t, wantLP, s.supply(t, pool.LpDenom), "LP supply")
	require.Equal(t, wantLP, s.transferred(t, res.Events, s.tradebin, s.blackHole, pool.LpDenom), "initial LP to the black hole")
	require.Equal(t, s.params.CreateMarketFee.Amount, s.transferred(t, res.Events, s.tradebin, s.cpFees, chain.Denom), "create fee to the community pool collector")

	spent := sdk.NewCoins().Add(sdk.NewInt64Coin(a.base, a.initBase), sdk.NewInt64Coin(a.quote, a.initQuote), s.params.CreateMarketFee, sdk.NewCoin(chain.Denom, s.txFee(t, res)))
	require.Equal(t, before.Sub(spent...), s.balances(t, s.lp, denoms...), "lp balances")
	return pool
}

// ammRequireDec compares two decimals whatever their printed precision (the
// chain prints LegacyDec with 18 decimals). An omitted zero reads as "".
func ammRequireDec(t *testing.T, want, got, what string) {
	t.Helper()
	dec := func(v string) sdkmath.LegacyDec {
		if v == "" {
			return sdkmath.LegacyZeroDec()
		}
		d, err := sdkmath.LegacyNewDecFromStr(v)
		require.NoError(t, err, "%s: %q", what, v)
		return d
	}
	w, g := dec(want), dec(got)
	require.True(t, w.Equal(g), "%s: want %s, got %s", what, w, g)
}

// ammRefusal is a message the chain must refuse with code in codespace and a
// raw log containing reason; checkTx tells whether ValidateBasic refuses it
// in CheckTx (never included, no fee) or the msg server in DeliverTx.
type ammRefusal struct {
	name      string
	msg       map[string]any
	code      uint32
	codespace string
	reason    string
	checkTx   bool
}

// requireRefusals sends each refused message as its own tx from user (signed
// and broadcast, so the CLI's own ValidateBasic does not stop it) and asserts
// the code, the reason and that pools, LP supplies, markets and the user's
// denoms are unchanged but for the tx fee.
func (s *ammSuite) requireRefusals(t *testing.T, user ibc.Wallet, denoms []string, cases []ammRefusal) {
	t.Helper()
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			state := s.state(t)
			before := s.balances(t, user, denoms...)
			res, err := helpers.ExecMsgs(s.ctx, s.bze, user, []helpers.TxOption{helpers.WithGas(ammTxGas)}, c.msg)
			reason, err := helpers.RejectionReason(res, err)
			require.NoError(t, err)
			require.Contains(t, reason, c.reason)
			require.EqualValues(t, c.code, res.Code, reason)
			require.Equal(t, c.codespace, res.Codespace, reason)
			require.Equal(t, c.checkTx, res.Height == 0, "refused by CheckTx: want %v, got height %d", c.checkTx, res.Height)

			require.Equal(t, state, s.state(t), "state moved on a refused tx")
			want := before
			if res.Height > 0 {
				want = before.Sub(sdk.NewCoin(chain.Denom, s.txFee(t, res)))
			}
			require.Equal(t, want, s.balances(t, user, denoms...), "a refused tx only costs the tx fee")
		})
	}
}

func (s *ammSuite) mustExecMsgs(t *testing.T, user ibc.Wallet, msgs ...any) helpers.TxResult {
	t.Helper()
	res, err := helpers.ExecMsgs(s.ctx, s.bze, user, []helpers.TxOption{helpers.WithGas(ammTxGas)}, msgs...)
	require.NoError(t, err)
	require.Zero(t, res.Code, res.RawLog)
	return res
}

// ammState is what a refused tx must not change: every pool as the chain
// prints it, every pool's LP supply and every market.
type ammState struct {
	Pools    string
	Supplies map[string]string
	Markets  string
}

func (s *ammSuite) state(t *testing.T) ammState {
	t.Helper()
	pools, err := helpers.Query[struct {
		List json.RawMessage `json:"list"`
	}](s.ctx, s.bze, "tradebin", "all-liquidity-pools")
	require.NoError(t, err)
	var list []poolInfo
	require.NoError(t, json.Unmarshal(pools.List, &list))
	require.NotEmpty(t, list)
	supplies := map[string]string{}
	for _, p := range list {
		supplies[p.LpDenom] = s.supply(t, p.LpDenom).String()
	}
	markets, err := helpers.Query[struct {
		Market json.RawMessage `json:"market"`
	}](s.ctx, s.bze, "tradebin", "all-markets")
	require.NoError(t, err)
	return ammState{Pools: string(pools.List), Supplies: supplies, Markets: string(markets.Market)}
}

func (s *ammSuite) requireNoPool(t *testing.T, id string) {
	t.Helper()
	_, err := helpers.Query[json.RawMessage](s.ctx, s.bze, "tradebin", "liquidity-pool", id)
	require.Error(t, err, "pool %s exists", id)
	require.Contains(t, err.Error(), "not found")
}

// ammSwap is one swap replayed with the chain's decimal math, for any pool
// fee and fee destination:
//
//	fee = ⌊in × poolFee⌋, treasury = ⌊fee × t⌋, burned = ⌊fee × b⌋
//	net = in − fee, out = ⌊outReserve × net / (inReserve + net)⌋
//	the input reserve grows by in − treasury − burned
type ammSwap struct {
	fee, treasury, burned, out, inReserveAdd sdkmath.Int
}

func ammQuoteSwap(in, inReserve, outReserve sdkmath.Int, poolFee, treasury, burner string) ammSwap {
	fee := sdkmath.LegacyNewDecFromInt(in).Mul(sdkmath.LegacyMustNewDecFromStr(poolFee)).TruncateInt()
	feeDec := sdkmath.LegacyNewDecFromInt(fee)
	tr := feeDec.Mul(sdkmath.LegacyMustNewDecFromStr(treasury)).TruncateInt()
	burned := feeDec.Mul(sdkmath.LegacyMustNewDecFromStr(burner)).TruncateInt()
	net := in.Sub(fee)
	out := sdkmath.LegacyNewDecFromInt(outReserve.Mul(net)).Quo(sdkmath.LegacyNewDecFromInt(inReserve.Add(net))).TruncateInt()
	return ammSwap{fee: fee, treasury: tr, burned: burned, out: out, inReserveAdd: in.Sub(tr).Sub(burned)}
}

// ammDepositResult is what an add-liquidity takes and mints.
type ammDepositResult struct {
	base, quote, minted sdkmath.Int
}

// ammDeposit replays the chain's add-liquidity: the side that would need
// more than was offered of the other is cut to the pool ratio (rounded down),
// and the LP minted is the smaller of the two deposit/reserve ratios times
// the supply, in LegacyDec, truncated.
func ammDeposit(base, quote, reserveBase, reserveQuote, supply sdkmath.Int) ammDepositResult {
	d := ammDepositResult{base: base, quote: base.Mul(reserveQuote).Quo(reserveBase)}
	if d.quote.GT(quote) {
		d.base, d.quote = quote.Mul(reserveBase).Quo(reserveQuote), quote
	}
	baseRatio := sdkmath.LegacyNewDecFromInt(d.base).Quo(sdkmath.LegacyNewDecFromInt(reserveBase))
	quoteRatio := sdkmath.LegacyNewDecFromInt(d.quote).Quo(sdkmath.LegacyNewDecFromInt(reserveQuote))
	d.minted = sdkmath.LegacyMinDec(baseRatio, quoteRatio).Mul(sdkmath.LegacyNewDecFromInt(supply)).TruncateInt()
	return d
}

// ammWithdraw replays the chain's remove-liquidity: each reserve times
// burn / supply, in LegacyDec, truncated.
func ammWithdraw(burn, supply, reserveBase, reserveQuote sdkmath.Int) (base, quote sdkmath.Int) {
	share := sdkmath.LegacyNewDecFromInt(burn).Quo(sdkmath.LegacyNewDecFromInt(supply))
	return sdkmath.LegacyNewDecFromInt(reserveBase).Mul(share).TruncateInt(),
		sdkmath.LegacyNewDecFromInt(reserveQuote).Mul(share).TruncateInt()
}

// swapQuote is one constant-product swap with the pool fee taken from the
// input, rounded down like the chain (fee 0.002, half of it burned):
//
//	fee = ⌊in × 0.002⌋, burned = ⌊fee × 0.5⌋, net = in − fee
//	out = ⌊outReserve × net / (inReserve + net)⌋
//	the input reserve grows by in − burned (the LPs keep the other half)
type swapQuote struct {
	fee, burned, out, inReserveAdd sdkmath.Int
}

func quoteSwap(in, inReserve, outReserve sdkmath.Int) swapQuote {
	fee := in.MulRaw(2).QuoRaw(1000)
	burned := fee.QuoRaw(2)
	net := in.Sub(fee)
	return swapQuote{
		fee:          fee,
		burned:       burned,
		out:          outReserve.Mul(net).Quo(inReserve.Add(net)),
		inReserveAdd: in.Sub(burned),
	}
}

// inputFor is the input that buys exactly out, rounded up (the chain charges a
// module fee in the user's fee denom with it):
//
//	⌈out × inReserve / ((outReserve − out) × (1 − 0.002))⌉
func inputFor(out, inReserve, outReserve sdkmath.Int) sdkmath.Int {
	num := out.Mul(inReserve).MulRaw(1000)
	den := outReserve.Sub(out).MulRaw(998)
	return num.Add(den).SubRaw(1).Quo(den)
}

// sqrtInt is the integer square root (the suite only uses perfect squares).
func sqrtInt(v sdkmath.Int) sdkmath.Int {
	return sdkmath.NewIntFromBigInt(new(big.Int).Sqrt(v.BigInt()))
}

func uniq(m map[string]string) map[string]struct{} {
	out := map[string]struct{}{}
	for _, v := range m {
		out[v] = struct{}{}
	}
	return out
}

func filterType(events []helpers.Event, typ string) []helpers.Event {
	var out []helpers.Event
	for _, e := range events {
		if e.Type == typ {
			out = append(out, e)
		}
	}
	return out
}

func (s *ammSuite) mustExec(t *testing.T, user ibc.Wallet, args ...string) helpers.TxResult {
	t.Helper()
	return s.mustExecWith(t, user, nil, args...)
}

func (s *ammSuite) mustExecWith(t *testing.T, user ibc.Wallet, opts []helpers.TxOption, args ...string) helpers.TxResult {
	t.Helper()
	res, err := helpers.ExecTxWith(s.ctx, s.bze, user, opts, args...)
	require.NoError(t, err)
	require.Zero(t, res.Code, "tx %v failed: %s", args, res.RawLog)
	return res
}

func (s *ammSuite) requireTxEvent(t *testing.T, res helpers.TxResult, name string, attrs map[string]string) helpers.Event {
	t.Helper()
	events := helpers.EventsWith(helpers.TxEvents(res, tradebinEventPrefix+name), attrs)
	require.Len(t, events, 1, "want one %s with %v in tx %s, got %+v", name, attrs, res.TxHash, helpers.TxEvents(res, tradebinEventPrefix+name))
	return events[0]
}

// requireSwap asserts exactly one SwapEvent of creator on the pool, with the
// exact input and output coins.
func (s *ammSuite) requireSwap(t *testing.T, events []helpers.Event, creator, poolID string, in, out sdk.Coin) {
	t.Helper()
	swaps := helpers.EventsWith(filterType(events, tradebinEventPrefix+"SwapEvent"), map[string]string{"creator": creator, "pool_id": poolID})
	require.Len(t, swaps, 1, "want one swap of %s on %s, got %+v", creator, poolID, filterType(events, tradebinEventPrefix+"SwapEvent"))
	assert.Equal(t, in, eventCoin(t, swaps[0], "in"), "swap input on %s", poolID)
	assert.Equal(t, out, eventCoin(t, swaps[0], "out"), "swap output on %s", poolID)
}

// requireTakerFeeBurned asserts the taker fee, paid in ubze, went to the
// burner fee collector in the tx and from there to the burner in EndBlock.
func (s *ammSuite) requireTakerFeeBurned(t *testing.T, res helpers.TxResult) {
	t.Helper()
	fee := s.params.MarketTakerFee.Amount
	require.Equal(t, fee, s.transferred(t, res.Events, s.tradebin, s.burnerFees, chain.Denom), "taker fee to the burner fee collector")
	require.Equal(t, fee, s.transferred(t, s.blockEvents(t, res.Height), s.burnerFees, s.burner, chain.Denom), "taker fee to the burner in EndBlock")
}

func eventCoin(t *testing.T, e helpers.Event, key string) sdk.Coin {
	t.Helper()
	var c struct {
		Denom  string `json:"denom"`
		Amount string `json:"amount"`
	}
	require.NoError(t, json.Unmarshal([]byte(e.Attributes[key]), &c), "attribute %s = %q", key, e.Attributes[key])
	return sdk.NewCoin(c.Denom, parseInt(c.Amount))
}

func (s *ammSuite) transferred(t *testing.T, events []helpers.Event, from, to, denom string) sdkmath.Int {
	t.Helper()
	got, err := helpers.Transferred(events, from, to, denom)
	require.NoError(t, err)
	return got
}

// blockEvents returns every finalize-block event at height (BeginBlock and
// EndBlock), typed events and bank transfers alike.
func (s *ammSuite) blockEvents(t *testing.T, height int64) []helpers.Event {
	t.Helper()
	var all []helpers.Event
	for _, typ := range []string{
		"transfer",
		tradebinEventPrefix + "SwapEvent",
		tradebinEventPrefix + "OrderExecutedEvent",
		tradebinEventPrefix + "OrderSavedEvent",
		tradebinEventPrefix + "OrderCanceledEvent",
	} {
		events, err := helpers.BlockEvents(s.ctx, s.bze, height, typ)
		require.NoError(t, err)
		all = append(all, events...)
	}
	return all
}

func (s *ammSuite) txFee(t *testing.T, res helpers.TxResult) sdkmath.Int {
	t.Helper()
	fee, err := helpers.TxFee(res, chain.Denom)
	require.NoError(t, err)
	return fee
}

func (s *ammSuite) balance(t *testing.T, addr, denom string) sdkmath.Int {
	t.Helper()
	b, err := helpers.Balance(s.ctx, s.bze, addr, denom)
	require.NoError(t, err)
	return b
}

// balances returns the user's balances of denoms as coins (zeros dropped), so
// whole before/after states compare in one assertion.
func (s *ammSuite) balances(t *testing.T, user ibc.Wallet, denoms ...string) sdk.Coins {
	t.Helper()
	out := sdk.NewCoins()
	for _, d := range denoms {
		if b := s.balance(t, user.FormattedAddress(), d); b.IsPositive() {
			out = out.Add(sdk.NewCoin(d, b))
		}
	}
	return out
}

func (s *ammSuite) supply(t *testing.T, denom string) sdkmath.Int {
	t.Helper()
	res, err := helpers.Query[struct {
		Amount struct {
			Amount string `json:"amount"`
		} `json:"amount"`
	}](s.ctx, s.bze, "bank", "total-supply-of", denom)
	require.NoError(t, err)
	return parseInt(res.Amount.Amount)
}

type poolInfo struct {
	ID           string `json:"id"`
	Base         string `json:"base"`
	Quote        string `json:"quote"`
	LpDenom      string `json:"lp_denom"`
	Fee          string `json:"fee"`
	ReserveBase  string `json:"reserve_base"`
	ReserveQuote string `json:"reserve_quote"`
	FeeDest      struct {
		Treasury  string `json:"treasury"`
		Burner    string `json:"burner"`
		Providers string `json:"providers"`
	} `json:"fee_dest"`
}

func (p poolInfo) reserveBase() sdkmath.Int  { return parseInt(p.ReserveBase) }
func (p poolInfo) reserveQuote() sdkmath.Int { return parseInt(p.ReserveQuote) }

func (s *ammSuite) pool(t *testing.T, id string) poolInfo {
	t.Helper()
	res, err := helpers.Query[struct {
		Pool poolInfo `json:"pool"`
	}](s.ctx, s.bze, "tradebin", "liquidity-pool", id)
	require.NoError(t, err)
	require.Equal(t, id, res.Pool.ID, "pool %s not found", id)
	return res.Pool
}

// tradebinParams are the fee params the suite charges; the module's Params
// proto uses camelCase field names.
type tradebinParams struct {
	CreateMarketFee     sdk.Coin
	MarketMakerFee      sdk.Coin
	MarketTakerFee      sdk.Coin
	TakerFeeDestination string
	MakerFeeDestination string
}

func (s *ammSuite) queryParams(t *testing.T) tradebinParams {
	t.Helper()
	type coin struct {
		Denom  string `json:"denom"`
		Amount string `json:"amount"`
	}
	res, err := helpers.Query[struct {
		Params struct {
			CreateMarketFee     coin   `json:"createMarketFee"`
			MarketMakerFee      coin   `json:"marketMakerFee"`
			MarketTakerFee      coin   `json:"marketTakerFee"`
			MakerFeeDestination string `json:"makerFeeDestination"`
			TakerFeeDestination string `json:"takerFeeDestination"`
		} `json:"params"`
	}](s.ctx, s.bze, "tradebin", "params")
	require.NoError(t, err)
	toCoin := func(name string, c coin) sdk.Coin {
		require.Equal(t, chain.Denom, c.Denom, "param %s is not in %s: %+v", name, chain.Denom, res.Params)
		return sdk.NewCoin(c.Denom, parseInt(c.Amount))
	}
	p := tradebinParams{
		CreateMarketFee:     toCoin("createMarketFee", res.Params.CreateMarketFee),
		MarketMakerFee:      toCoin("marketMakerFee", res.Params.MarketMakerFee),
		MarketTakerFee:      toCoin("marketTakerFee", res.Params.MarketTakerFee),
		MakerFeeDestination: res.Params.MakerFeeDestination,
		TakerFeeDestination: res.Params.TakerFeeDestination,
	}
	// the suite follows trading fees to the burner
	require.Equal(t, "burner", p.TakerFeeDestination)
	require.Equal(t, "burner", p.MakerFeeDestination)
	require.True(t, p.MarketTakerFee.IsPositive() && p.MarketMakerFee.IsPositive() && p.CreateMarketFee.IsPositive(), "fees: %+v", p)
	return p
}
