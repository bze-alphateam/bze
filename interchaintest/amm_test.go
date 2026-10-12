package interchaintest_test

import (
	"context"
	"encoding/json"
	"math/big"
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
)

// TestAmm runs the tradebin AMM and the fee path on one network: pools,
// liquidity, single and multi-hop swaps with exact outputs, a tx fee paid in a
// factory token and swapped to ubze in the txfeecollector EndBlock, the taker
// fee charged in that token and burned, --gas auto with --gas-prices in ubze
// and in a factory token, and the message_id that links the order events of a
// MsgFillOrders to the EndBlock engine events. Subtests run in order and share
// state; the step numbers match the scenario of the suite's design.
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
}

type ammSuite struct {
	ctx context.Context
	bze *cosmos.CosmosChain

	lp, trader, maker, taker ibc.Wallet
	usd, abc                 string
	usdPool, abcPool         string
	market                   string
	params                   tradebinParams

	// module account addresses
	tradebin, burner, blackHole, feeCollector, txFees, burnerFees, cpFees string
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

	for name, addr := range map[string]*string{
		"tradebin":              &s.tradebin,
		"burner":                &s.burner,
		"burner_black_hole":     &s.blackHole,
		"fee_collector":         &s.feeCollector,
		"txfeecollector":        &s.txFees,
		"txfeecollector_burner": &s.burnerFees,
		"txfeecollector_cp":     &s.cpFees,
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
