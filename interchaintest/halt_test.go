package interchaintest_test

import (
	"context"
	"encoding/json"
	"strconv"
	"testing"
	"time"

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

// The ubze side of hlt/ubze and usd/ubze is above the tradebin
// minNativeLiquidityForModuleSwap param (100,000 BZE by default), so both
// tokens qualify as fee denoms while they are not halted.
const (
	haltAdminFunds  = 1_000_000_000_000 // 1,000,000 BZE: three denoms, three pools, a market, proposals
	haltTraderFunds = 50_000_000_000    // 50,000 BZE: enough for the create fees it is refused before paying

	haltMint = 1_000_000_000_000
	// hlt/ubze and usd/ubze: 4 ubze per token
	haltPoolToken = 50_000_000_000
	haltPoolBze   = 200_000_000_000
	// hlt/usd: 1 usd per hlt
	haltPoolPair = 10_000_000_000

	haltTraderTokens = 1_000_000_000 // hlt, usd and oth sent to trader
	// trader's deposit to hlt/ubze, at the pool price
	haltLpHlt = 100_000_000
	haltLpBze = 400_000_000

	haltOrderAmount = 1_000
	haltOrderPrice  = "10" // ubze per hlt, far above the pool price: the order rests
	haltSwapInput   = 1_000_000
	haltFee         = 20_000 // a tx fee in hlt or usd
	haltTxGas       = 500_000
	haltSendGas     = 200_000

	haltCodespace = "tradebin"
	haltCode      = 4017 // ErrDenomHalted

	haltUserFunds = 10_000_000_000 // 10,000 BZE each for queuer, maker and taker

	// step 2: one tx of haltBacklogOrders sell orders, drained one per block
	// (orderBookPerBlockMessages lowered to 1) while the halt proposal of step
	// 3 is voted (about 25 blocks): the margin keeps it draining past the halt.
	// The price is above every other sell, so nothing crosses or fills them.
	haltBacklogOrders = 60
	haltBacklogAmount = 10
	haltBacklogPrice  = "15"
	haltBacklogGas    = 40_000_000 // about 9M of order gas plus 5,000 per queued message scanned
	haltPerBlockCap   = 1

	// maker's hlt sell at haltOrderPrice, resting through the halt (step 10)
	haltMakerAmount = 500
	// maker's usd sell on usd/ubze during the halt (step 8)
	haltPairAmount = 1_000
	haltPairPrice  = "4"
)

// TestHaltedDenoms halts a factory denom through a real governance proposal
// carrying MsgHaltDenoms and proves the tradebin msg servers refuse every
// trading path that touches it (code 4017, nothing moves but the tx fee),
// while cancelling an order, removing liquidity and bank sends keep working,
// and the ante handler no longer takes it as a fee denom. Messages queued
// before the halt still execute after it, and other pairs keep trading. A
// second proposal unhalts it, trading works again and an order that rested
// through the halt fills. One proposal halts and unhalts two denoms at once.
// Subtests run in order and share state; the step numbers match the scenario
// of the suite's design.
func TestHaltedDenoms(t *testing.T) {
	if testing.Short() {
		t.Skip("e2e: skipped in short mode")
	}
	ctx := context.Background()
	net := chain.Start(t, ctx)
	s := setupHalt(t, ctx, net.Chain)

	t.Run("1 before the halt", s.beforeHalt)
	t.Run("2 queue a backlog", s.queueBacklog)
	t.Run("3 halt by governance", s.halt)
	t.Run("4 queued messages execute after the halt", s.backlogAfterHalt)
	t.Run("5 trading refused", s.tradingRefused)
	t.Run("6 exits still work", s.exits)
	t.Run("7 fee denom", s.feeDenom)
	t.Run("8 other pairs trade during the halt", s.otherPairs)
	t.Run("9 unhalt by governance", s.unhalt)
	t.Run("10 resting order fills after the unhalt", s.restingFills)
	t.Run("11 several denoms in one proposal", s.severalDenoms)
	t.Run("12 native denom can not be halted", s.haltNative)
}

type haltSuite struct {
	// the AMM suite's query and tx helpers (pool, balances, supply, events)
	*ammSuite

	admin, trader ibc.Wallet
	queuer        ibc.Wallet // step 2's backlog of queued orders
	maker, taker  ibc.Wallet // the resting order of step 10 and the trades of step 8
	gov           string

	hlt, usd, oth           string
	hltPool, usdPool, pair  string // hlt/ubze, usd/ubze, hlt/usd
	hltLp                   string
	market                  string // hlt/ubze
	usdMarket               string // usd/ubze
	orderID                 string // trader's resting sell order, placed before the halt
	makerOrderID            string // maker's sell on hlt/ubze, placed before the halt, filled in step 10
	minNativeLiquidityParam sdkmath.Int
	defaults                map[string]json.RawMessage // tradebin params before the suite changed any

	backlog       []string  // step 2's queue message ids, in queue order
	backlogHeight int64     // the block of step 2's tx
	queuerAfter   sdk.Coins // queuer's ubze and hlt once step 2's orders were paid
	haltHeight    int64     // the block whose EndBlock halted hlt
}

func setupHalt(t *testing.T, ctx context.Context, bze *cosmos.CosmosChain) *haltSuite {
	s := &haltSuite{ammSuite: &ammSuite{ctx: ctx, bze: bze}}

	s.admin = interchaintest.GetAndFundTestUsers(t, ctx, "halt-admin", sdkmath.NewInt(haltAdminFunds), bze)[0]
	s.trader = interchaintest.GetAndFundTestUsers(t, ctx, "halt-trader", sdkmath.NewInt(haltTraderFunds), bze)[0]
	users := interchaintest.GetAndFundTestUsers(t, ctx, "halt-user", sdkmath.NewInt(haltUserFunds), bze, bze, bze)
	s.queuer, s.maker, s.taker = users[0], users[1], users[2]
	require.NoError(t, testutil.WaitForBlocks(ctx, 2, bze))

	var err error
	s.gov, err = helpers.GovAuthority(ctx, bze)
	require.NoError(t, err)
	for name, addr := range map[string]*string{
		"tradebin":              &s.tradebin,
		"txfeecollector":        &s.txFees,
		"burner":                &s.burner,
		"txfeecollector_burner": &s.burnerFees,
	} {
		*addr, err = helpers.ModuleAddress(ctx, bze, name)
		require.NoError(t, err)
	}
	s.params = s.queryParams(t)
	s.defaults = s.orderBook().rawParams(t)
	s.minNativeLiquidityParam = s.minNativeLiquidity(t)
	require.True(t, sdkmath.NewInt(haltPoolBze).GTE(s.minNativeLiquidityParam), "the pools must qualify their tokens as fee denoms")

	factory := "factory/" + s.admin.FormattedAddress() + "/"
	s.hlt, s.usd, s.oth = factory+"hlt", factory+"usd", factory+"oth"
	for _, sub := range []string{"hlt", "usd", "oth"} {
		s.mustExec(t, s.admin, "tokenfactory", "create-denom", sub)
		s.mustExec(t, s.admin, "tokenfactory", "mint", coinStr(haltMint, factory+sub))
		s.mustExec(t, s.admin, "bank", "send", s.admin.FormattedAddress(), s.trader.FormattedAddress(), coinStr(haltTraderTokens, factory+sub))
	}

	// pool ids order the denoms alphabetically: "factory/…/hlt" < "factory/…/usd" < "ubze"
	s.hltPool = s.hlt + "_" + chain.Denom
	s.usdPool = s.usd + "_" + chain.Denom
	s.pair = s.hlt + "_" + s.usd
	s.market = s.hlt + "/" + chain.Denom
	for _, p := range []struct {
		base, quote         string
		initBase, initQuote int64
	}{
		{s.hlt, chain.Denom, haltPoolToken, haltPoolBze},
		{s.usd, chain.Denom, haltPoolToken, haltPoolBze},
		{s.hlt, s.usd, haltPoolPair, haltPoolPair},
	} {
		s.mustExecWith(t, s.admin, []helpers.TxOption{helpers.WithGas(haltTxGas)},
			"tradebin", "create-liquidity-pool", p.base, p.quote, ammPoolFee, ammFeeDest, "false",
			sdkmath.NewInt(p.initBase).String(), sdkmath.NewInt(p.initQuote).String())
	}
	s.hltLp = s.pool(t, s.hltPool).LpDenom
	s.mustExecWith(t, s.admin, []helpers.TxOption{helpers.WithGas(haltTxGas)}, "tradebin", "create-market", s.hlt, chain.Denom)

	// trader: LP shares of hlt/ubze and one resting sell order
	s.mustExec(t, s.trader, "tradebin", "add-liquidity", s.hltPool,
		sdkmath.NewInt(haltLpHlt).String(), sdkmath.NewInt(haltLpBze).String(), "1")
	require.True(t, s.balance(t, s.trader.FormattedAddress(), s.hltLp).IsPositive(), "trader holds hlt/ubze LP")
	s.orderID = s.placeSellOrder(t)

	// maker: a second sell on the hlt market at the trader's price, placed
	// after it; the queuer's hlt for step 2 and maker's usd for step 8
	for _, send := range []struct {
		to   ibc.Wallet
		coin string
	}{
		{s.maker, coinStr(haltMakerAmount, s.hlt)},
		{s.maker, coinStr(haltPairAmount, s.usd)},
		{s.queuer, coinStr(haltBacklogOrders*haltBacklogAmount, s.hlt)},
	} {
		s.mustExec(t, s.admin, "bank", "send", s.admin.FormattedAddress(), send.to.FormattedAddress(), send.coin)
	}
	s.makerOrderID = s.restSell(t, s.maker, s.market, s.hlt, haltMakerAmount, haltOrderPrice)

	// usd/ubze: a market the hlt halt does not touch
	s.usdMarket = s.usd + "/" + chain.Denom
	s.mustExecWith(t, s.admin, []helpers.TxOption{helpers.WithGas(haltTxGas)}, "tradebin", "create-market", s.usd, chain.Denom)
	return s
}

// 1. hlt is not halted and, with a deep ubze pool, pays tx fees: the fee goes
// to the txfeecollector, which swaps it to ubze through hlt/ubze in EndBlock.
func (s *haltSuite) beforeHalt(t *testing.T) {
	s.requireHaltedDenoms(t)
	require.False(t, s.denomHalted(t, s.hlt))

	res := s.mustExecWith(t, s.trader, []helpers.TxOption{helpers.WithFees(coinStr(haltFee, s.hlt)), helpers.WithGas(haltSendGas)},
		"bank", "send", s.trader.FormattedAddress(), s.admin.FormattedAddress(), coinStr(1, chain.Denom))
	paid, err := helpers.TxFee(res, s.hlt)
	require.NoError(t, err)
	require.Equal(t, sdkmath.NewInt(haltFee), paid, "fee paid in hlt")
	swaps := helpers.EventsWith(filterType(s.blockEvents(t, res.Height), tradebinEventPrefix+"SwapEvent"), map[string]string{"creator": s.txFees, "pool_id": s.hltPool})
	require.Len(t, swaps, 1, "the hlt fee is swapped to ubze in EndBlock")
	require.Equal(t, sdk.NewInt64Coin(s.hlt, haltFee), eventCoin(t, swaps[0], "in"))
}

// 2. With orderBookPerBlockMessages lowered to 1 by governance, one tx from
// the queuer queues haltBacklogOrders sell orders on the hlt market. The msg
// server takes the maker fees and locks the hlt at once; the EndBlock saves
// one order per block, so the queue is still draining when step 3's halt
// executes.
func (s *haltSuite) queueBacklog(t *testing.T) {
	s.orderBook().updateParams(t, "Order book cap 1", map[string]uint64{"orderBookPerBlockMessages": haltPerBlockCap})

	amount := strconv.Itoa(haltBacklogAmount)
	msgs := make([]any, 0, haltBacklogOrders)
	for range haltBacklogOrders {
		msgs = append(msgs, map[string]any{
			"@type":      "/bze.tradebin.MsgCreateOrder",
			"creator":    s.queuer.FormattedAddress(),
			"order_type": "sell",
			"amount":     amount,
			"price":      haltBacklogPrice,
			"market_id":  s.market,
		})
	}
	before := s.balances(t, s.queuer, chain.Denom, s.hlt)
	res, err := helpers.ExecMsgs(s.ctx, s.bze, s.queuer, []helpers.TxOption{helpers.WithGas(haltBacklogGas)}, msgs...)
	require.NoError(t, err)
	require.Zero(t, res.Code, res.RawLog)
	t.Logf("backlog tx at height %d: gas used %d of %d", res.Height, res.GasUsed, res.GasWanted)

	created := helpers.TxEvents(res, tradebinEventPrefix+"OrderCreateMessageEvent")
	require.Len(t, created, haltBacklogOrders, "one queued message per order")
	ids := make(map[string]string, len(created))
	s.backlog = make([]string, 0, len(created))
	for _, e := range created {
		require.Equal(t, []string{s.queuer.FormattedAddress(), s.market, "sell", amount, haltBacklogPrice},
			[]string{e.Attributes["creator"], e.Attributes["market_id"], e.Attributes["order_type"], e.Attributes["amount"], e.Attributes["price"]})
		id := e.Attributes["message_id"]
		s.backlog = append(s.backlog, id)
		ids[id] = id
	}
	require.Len(t, uniq(ids), haltBacklogOrders, "message ids are distinct")

	// no buy anywhere near: every order pays the maker fee
	fees := s.params.MarketMakerFee.Amount.MulRaw(haltBacklogOrders)
	locked := sdkmath.NewInt(haltBacklogOrders * haltBacklogAmount)
	require.Equal(t, fees, s.transferred(t, res.Events, s.queuer.FormattedAddress(), s.tradebin, chain.Denom), "maker fees to the tradebin module")
	require.Equal(t, locked, s.transferred(t, res.Events, s.queuer.FormattedAddress(), s.tradebin, s.hlt), "order amounts locked")
	require.Equal(t, fees, s.transferred(t, res.Events, s.tradebin, s.burnerFees, chain.Denom), "maker fees to the burner fee collector")
	s.queuerAfter = before.Sub(sdk.NewCoin(s.hlt, locked), sdk.NewCoin(chain.Denom, fees.Add(s.txFee(t, res))))
	require.Equal(t, s.queuerAfter, s.balances(t, s.queuer, chain.Denom, s.hlt), "queuer balances")

	// the tx's own block saves the first message only
	s.backlogHeight = res.Height
	saved := s.blockTypedEvents(t, res.Height, "OrderSavedEvent")
	require.Len(t, saved, haltPerBlockCap, "orders saved in the tx's block: %+v", saved)
	require.Equal(t, s.backlog[0], saved[0].Attributes["message_id"])
}

// 3. A proposal halts hlt. Its second message halts the already-halted denom
// again: a no-op without a second event. The same message signed by a user
// is refused.
func (s *haltSuite) halt(t *testing.T) {
	halt := s.haltMsg("MsgHaltDenoms", s.gov, s.hlt)
	height := s.passProposal(t, "Halt hlt", halt, halt)
	s.haltHeight = height

	events := s.blockTypedEvents(t, height, "DenomHaltedEvent")
	require.Len(t, events, 1, "one DenomHaltedEvent in the block that executed the proposal, none for the repeat: %+v", events)
	require.Equal(t, s.hlt, events[0].Attributes["denom"])
	require.True(t, s.denomHalted(t, s.hlt))
	s.requireHaltedDenoms(t, s.hlt)

	for _, typ := range []string{"MsgHaltDenoms", "MsgUnhaltDenoms"} {
		t.Run(typ+" signed by a user", func(t *testing.T) {
			res, err := helpers.ExecMsgs(s.ctx, s.bze, s.trader, []helpers.TxOption{helpers.WithGas(haltTxGas)},
				s.haltMsg(typ, s.trader.FormattedAddress(), s.hlt))
			reason, err := helpers.RejectionReason(res, err)
			require.NoError(t, err)
			require.Contains(t, reason, "invalid authority")
			require.True(t, s.denomHalted(t, s.hlt), "hlt still halted")
		})
	}
}

// 4. Messages accepted before the halt still execute after it: the EndBlock
// engine has no halt gate, only the msg servers refuse. The backlog keeps
// draining one message per block, in queue order, past the block that halted
// hlt; every message ends as a resting order on the halted market, and the
// queuer gets nothing back (no message was dropped and refunded).
func (s *haltSuite) backlogAfterHalt(t *testing.T) {
	require.NotZero(t, s.haltHeight, "step 3 did not halt hlt")
	last := s.backlogHeight + int64(len(s.backlog)) - 1
	if last <= s.haltHeight {
		t.Fatalf("the backlog drained at height %d, before the halt executed at height %d: nothing was queued across the halt, raise haltBacklogOrders", last, s.haltHeight)
	}
	ctx, cancel := context.WithTimeout(s.ctx, 2*time.Minute)
	defer cancel()
	_, err := helpers.WaitForHeight(ctx, s.bze, last+1)
	require.NoError(t, err)

	ids := make([]string, 0, len(s.backlog))
	afterHalt := 0
	for i, msgID := range s.backlog {
		h := s.backlogHeight + int64(i)
		saved := s.blockTypedEvents(t, h, "OrderSavedEvent")
		require.Len(t, saved, haltPerBlockCap, "orders saved at height %d: %+v", h, saved)
		e := saved[0].Attributes
		require.Equal(t,
			[]string{msgID, s.market, s.queuer.FormattedAddress(), "sell", strconv.Itoa(haltBacklogAmount), haltBacklogPrice},
			[]string{e["message_id"], e["market_id"], e["owner"], e["order_type"], e["amount"], e["price"]},
			"message %d of the backlog, saved at height %d", i, h)
		ids = append(ids, e["id"])
		if h > s.haltHeight {
			afterHalt++
		}
	}
	saved := s.blockTypedEvents(t, last+1, "OrderSavedEvent")
	require.Empty(t, saved, "the queue is empty once the backlog drained")
	t.Logf("halt at height %d; backlog saved at heights %d..%d, %d of them after the halt", s.haltHeight, s.backlogHeight, last, afterHalt)
	require.Positive(t, afterHalt, "orders saved after the halt")
	require.True(t, s.denomHalted(t, s.hlt), "hlt still halted")

	var onBook []string
	for _, o := range s.ordersOf(t, s.queuer, s.market) {
		onBook = append(onBook, o.ID)
	}
	require.ElementsMatch(t, ids, onBook, "every saved order is on the book")
	// the saved amounts were checked above; the level holds all of them
	require.Equal(t, strconv.Itoa(haltBacklogOrders*haltBacklogAmount), s.level(t, s.market, "sell", haltBacklogPrice), "aggregated sell level")
	require.Equal(t, s.queuerAfter, s.balances(t, s.queuer, chain.Denom, s.hlt), "queuer balances: every order keeps its locked hlt")
}

// 5. Every trading entry point that touches hlt is refused by its msg server
// with ErrDenomHalted. The tx is delivered (fixed gas) so the code is the
// msg server's; pools, markets, the order book and the trader's balances are
// unchanged apart from the ubze tx fee.
func (s *haltSuite) tradingRefused(t *testing.T) {
	cli := func(args ...string) func() (helpers.TxResult, error) {
		return func() (helpers.TxResult, error) {
			return helpers.ExecTxWith(s.ctx, s.bze, s.trader, []helpers.TxOption{helpers.WithGas(haltTxGas)}, args...)
		}
	}
	amount := sdkmath.NewInt(haltOrderAmount).String()
	for _, c := range []struct {
		name string
		tx   func() (helpers.TxResult, error)
	}{
		{"create order", cli("tradebin", "create-order", "buy", amount, haltOrderPrice, s.market)},
		{"swap hlt to ubze", cli("tradebin", "multi-swap", s.hltPool, coinStr(haltSwapInput, s.hlt), coinStr(1, chain.Denom))},
		{"swap ubze to hlt", cli("tradebin", "multi-swap", s.hltPool, coinStr(haltSwapInput, chain.Denom), coinStr(1, s.hlt))},
		{"multi-hop through hlt", cli("tradebin", "multi-swap", s.pair+","+s.hltPool, coinStr(haltSwapInput, s.usd), coinStr(1, chain.Denom))},
		{"add liquidity", cli("tradebin", "add-liquidity", s.hltPool, sdkmath.NewInt(haltLpHlt).String(), sdkmath.NewInt(haltLpBze).String(), "1")},
		{"create pool", cli("tradebin", "create-liquidity-pool", s.hlt, s.oth, ammPoolFee, ammFeeDest, "false",
			sdkmath.NewInt(haltSwapInput).String(), sdkmath.NewInt(haltSwapInput).String())},
		{"create market", cli("tradebin", "create-market", s.hlt, s.usd)},
		{"fill orders", func() (helpers.TxResult, error) {
			return helpers.ExecMsgs(s.ctx, s.bze, s.trader, []helpers.TxOption{helpers.WithGas(haltTxGas)}, map[string]any{
				"@type":      "/bze.tradebin.MsgFillOrders",
				"creator":    s.trader.FormattedAddress(),
				"market_id":  s.market,
				"order_type": "sell",
				"orders":     []map[string]string{{"price": haltOrderPrice, "amount": amount}},
			})
		}},
	} {
		t.Run(c.name, func(t *testing.T) {
			before := s.snapshot(t)
			res, err := c.tx()
			require.NoError(t, err)
			require.NotZero(t, res.Height, "refused by CheckTx instead of the msg server: %s", res.RawLog)
			require.Equal(t, haltCodespace, res.Codespace, res.RawLog)
			require.EqualValues(t, haltCode, res.Code, res.RawLog)
			require.Contains(t, res.RawLog, "halted")

			after := s.snapshot(t)
			require.Equal(t, before.pools, after.pools, "pools changed")
			require.Equal(t, before.markets, after.markets, "markets changed")
			require.Equal(t, before.orders, after.orders, "order book changed")
			require.Equal(t, before.balances.Sub(sdk.NewCoin(chain.Denom, s.txFee(t, res))), after.balances, "a refused tx only costs the tx fee")
		})
	}
	require.Equal(t, s.orderID, s.restingOrder(t).ID, "the resting order is still on the book")
}

// 6. Exits are never halted: the resting order is cancelled and refunded,
// the trader's LP shares are redeemed for both sides, and hlt still moves by
// bank send.
func (s *haltSuite) exits(t *testing.T) {
	t.Run("cancel order", func(t *testing.T) {
		before := s.balance(t, s.trader.FormattedAddress(), s.hlt)
		res := s.mustExec(t, s.trader, "tradebin", "cancel-order", s.market, s.orderID, "sell")
		canceled := helpers.EventsWith(filterType(s.blockEvents(t, res.Height), tradebinEventPrefix+"OrderCanceledEvent"), map[string]string{"id": s.orderID})
		require.Len(t, canceled, 1, "the cancel executes in EndBlock")
		require.Equal(t, before.AddRaw(haltOrderAmount), s.balance(t, s.trader.FormattedAddress(), s.hlt), "order amount refunded")
		require.Empty(t, s.traderOrders(t), "no order left on the book")
	})

	t.Run("remove liquidity", func(t *testing.T) {
		pool := s.pool(t, s.hltPool)
		supply := s.supply(t, s.hltLp)
		burn := s.balance(t, s.trader.FormattedAddress(), s.hltLp)
		// out = ⌊reserve × burn / supply⌋
		wantHlt := pool.reserveBase().Mul(burn).Quo(supply)
		wantBze := pool.reserveQuote().Mul(burn).Quo(supply)
		require.True(t, wantHlt.IsPositive() && wantBze.IsPositive())
		before := s.balances(t, s.trader, chain.Denom, s.hlt, s.hltLp)

		res := s.mustExec(t, s.trader, "tradebin", "remove-liquidity", s.hltPool, burn.String(), wantHlt.String(), wantBze.String())

		s.requireTxEvent(t, res, "LiquidityRemovedEvent", map[string]string{
			"pool_id": s.hltPool, "baseAmount": wantHlt.String(), "quoteAmount": wantBze.String(),
		})
		after := s.pool(t, s.hltPool)
		require.Equal(t, pool.reserveBase().Sub(wantHlt), after.reserveBase())
		require.Equal(t, pool.reserveQuote().Sub(wantBze), after.reserveQuote())
		want := before.Add(sdk.NewCoin(s.hlt, wantHlt), sdk.NewCoin(chain.Denom, wantBze)).
			Sub(sdk.NewCoin(s.hltLp, burn), sdk.NewCoin(chain.Denom, s.txFee(t, res)))
		require.Equal(t, want, s.balances(t, s.trader, chain.Denom, s.hlt, s.hltLp), "trader balances")
	})

	t.Run("bank send", func(t *testing.T) {
		before := s.balance(t, s.admin.FormattedAddress(), s.hlt)
		s.mustExec(t, s.trader, "bank", "send", s.trader.FormattedAddress(), s.admin.FormattedAddress(), coinStr(haltOrderAmount, s.hlt))
		require.Equal(t, before.AddRaw(haltOrderAmount), s.balance(t, s.admin.FormattedAddress(), s.hlt))
	})
}

// 7. The ante handler refuses a fee in a halted denom (a halted denom never
// has usable liquidity), before the tx reaches a block: nothing is charged.
// A fee in usd, whose pool is not halted, still works.
func (s *haltSuite) feeDenom(t *testing.T) {
	t.Run("hlt refused", func(t *testing.T) {
		before := s.balances(t, s.trader, chain.Denom, s.hlt)
		res, err := helpers.ExecTxWith(s.ctx, s.bze, s.trader, []helpers.TxOption{helpers.WithFees(coinStr(haltFee, s.hlt)), helpers.WithGas(haltSendGas)},
			"bank", "send", s.trader.FormattedAddress(), s.admin.FormattedAddress(), coinStr(1, chain.Denom))
		reason, err := helpers.RejectionReason(res, err)
		require.NoError(t, err)
		require.Contains(t, reason, s.hlt+" can be used to pay for fees only if enough liquidity is available")
		require.Zero(t, res.Height, "refused by CheckTx")
		require.Equal(t, before, s.balances(t, s.trader, chain.Denom, s.hlt), "nothing charged")
	})

	t.Run("usd accepted", func(t *testing.T) {
		res := s.mustExecWith(t, s.trader, []helpers.TxOption{helpers.WithFees(coinStr(haltFee, s.usd)), helpers.WithGas(haltSendGas)},
			"bank", "send", s.trader.FormattedAddress(), s.admin.FormattedAddress(), coinStr(1, chain.Denom))
		paid, err := helpers.TxFee(res, s.usd)
		require.NoError(t, err)
		require.Equal(t, sdkmath.NewInt(haltFee), paid, "fee paid in usd")
	})
}

// 8. The halt touches only hlt: while it is halted, a swap on usd/ubze and an
// order on the usd/ubze market, filled by a counter order, execute with exact
// amounts and fees.
func (s *haltSuite) otherPairs(t *testing.T) {
	require.True(t, s.denomHalted(t, s.hlt), "hlt halted")

	t.Run("swap", func(t *testing.T) {
		pool := s.pool(t, s.usdPool)
		in := sdkmath.NewInt(haltSwapInput)
		q := quoteSwap(in, pool.reserveQuote(), pool.reserveBase())
		before := s.balances(t, s.taker, chain.Denom, s.usd)

		res := s.mustExec(t, s.taker, "tradebin", "multi-swap", s.usdPool, sdk.NewCoin(chain.Denom, in).String(), coinStr(1, s.usd))

		s.requireSwap(t, res.Events, s.taker.FormattedAddress(), s.usdPool, sdk.NewCoin(chain.Denom, in), sdk.NewCoin(s.usd, q.out))
		after := s.pool(t, s.usdPool)
		require.Equal(t, pool.reserveQuote().Add(q.inReserveAdd), after.reserveQuote(), "ubze reserve")
		require.Equal(t, pool.reserveBase().Sub(q.out), after.reserveBase(), "usd reserve")
		want := before.Add(sdk.NewCoin(s.usd, q.out)).Sub(sdk.NewCoin(chain.Denom, in.Add(s.params.MarketTakerFee.Amount).Add(s.txFee(t, res))))
		require.Equal(t, want, s.balances(t, s.taker, chain.Denom, s.usd), "taker balances")
		require.Equal(t, q.burned, s.transferred(t, res.Events, s.tradebin, s.burner, chain.Denom), "swap fee to the burner")
		s.requireTakerFeeBurned(t, res)
	})

	t.Run("order", func(t *testing.T) {
		id := s.restSell(t, s.maker, s.usdMarket, s.usd, haltPairAmount, haltPairPrice)
		s.fillSell(t, s.maker, s.usdMarket, s.usd, id, haltPairAmount, haltPairPrice)
	})
}

// 9. A second proposal lifts the halt; its repeated message finds hlt no
// longer halted and emits nothing. Swaps, liquidity and orders work again.
func (s *haltSuite) unhalt(t *testing.T) {
	unhalt := s.haltMsg("MsgUnhaltDenoms", s.gov, s.hlt)
	height := s.passProposal(t, "Unhalt hlt", unhalt, unhalt)

	events := s.blockTypedEvents(t, height, "DenomUnhaltedEvent")
	require.Len(t, events, 1, "one DenomUnhaltedEvent, none for the repeat: %+v", events)
	require.Equal(t, s.hlt, events[0].Attributes["denom"])
	require.False(t, s.denomHalted(t, s.hlt))
	s.requireHaltedDenoms(t)

	t.Run("swap", func(t *testing.T) {
		pool := s.pool(t, s.hltPool)
		in := sdkmath.NewInt(haltSwapInput)
		q := quoteSwap(in, pool.reserveQuote(), pool.reserveBase())
		res := s.mustExec(t, s.trader, "tradebin", "multi-swap", s.hltPool, sdk.NewCoin(chain.Denom, in).String(), coinStr(1, s.hlt))
		s.requireSwap(t, res.Events, s.trader.FormattedAddress(), s.hltPool, sdk.NewCoin(chain.Denom, in), sdk.NewCoin(s.hlt, q.out))
	})

	t.Run("add liquidity", func(t *testing.T) {
		before := s.balance(t, s.trader.FormattedAddress(), s.hltLp)
		res := s.mustExec(t, s.trader, "tradebin", "add-liquidity", s.hltPool,
			sdkmath.NewInt(haltLpHlt).String(), sdkmath.NewInt(haltLpBze).String(), "1")
		s.requireTxEvent(t, res, "LiquidityAddedEvent", map[string]string{"pool_id": s.hltPool})
		require.True(t, s.balance(t, s.trader.FormattedAddress(), s.hltLp).GT(before), "LP minted")
	})

	t.Run("new order", func(t *testing.T) {
		id := s.placeSellOrder(t)
		require.Equal(t, id, s.restingOrder(t).ID)
	})
}

// 10. maker's sell, placed before the halt and left on the book through it,
// matches again once hlt is unhalted: a counter buy at its price fills it
// whole, ahead of the trader's newer order at the same price (FIFO), and pays
// the taker fee.
func (s *haltSuite) restingFills(t *testing.T) {
	require.False(t, s.denomHalted(t, s.hlt))
	require.Equal(t, strconv.Itoa(haltMakerAmount), s.orderAmount(t, s.market, "sell", s.makerOrderID), "maker's order untouched by the halt")
	newer := s.restingOrder(t).ID
	require.Equal(t, strconv.Itoa(haltMakerAmount+haltOrderAmount), s.level(t, s.market, "sell", haltOrderPrice), "maker's and trader's sells at one price")

	s.fillSell(t, s.maker, s.market, s.hlt, s.makerOrderID, haltMakerAmount, haltOrderPrice)

	require.Equal(t, newer, s.restingOrder(t).ID, "the trader's newer order is still on the book")
	require.Equal(t, strconv.Itoa(haltOrderAmount), s.level(t, s.market, "sell", haltOrderPrice), "only the trader's order is left at the price")
}

// 11. One MsgHaltDenoms halts usd and oth at once: one DenomHaltedEvent per
// denom, both are listed and a swap on usd/ubze is refused. One
// MsgUnhaltDenoms lifts both; the same proposal restores the tradebin params
// step 2 changed.
func (s *haltSuite) severalDenoms(t *testing.T) {
	height := s.passProposal(t, "Halt usd and oth", s.haltMsg("MsgHaltDenoms", s.gov, s.usd, s.oth))

	require.ElementsMatch(t, []string{s.usd, s.oth}, eventDenoms(s.blockTypedEvents(t, height, "DenomHaltedEvent")), "one DenomHaltedEvent per denom")
	s.requireHaltedDenoms(t, s.usd, s.oth)
	require.True(t, s.denomHalted(t, s.usd))
	require.True(t, s.denomHalted(t, s.oth))

	t.Run("swap on usd refused", func(t *testing.T) {
		pool := s.pool(t, s.usdPool)
		before := s.balances(t, s.taker, chain.Denom, s.usd)
		res, err := helpers.ExecTxWith(s.ctx, s.bze, s.taker, []helpers.TxOption{helpers.WithGas(haltTxGas)},
			"tradebin", "multi-swap", s.usdPool, coinStr(haltSwapInput, chain.Denom), coinStr(1, s.usd))
		require.NoError(t, err)
		require.NotZero(t, res.Height, "refused by CheckTx instead of the msg server: %s", res.RawLog)
		require.Equal(t, haltCodespace, res.Codespace, res.RawLog)
		require.EqualValues(t, haltCode, res.Code, res.RawLog)
		require.Equal(t, pool, s.pool(t, s.usdPool), "usd/ubze pool moved")
		require.Equal(t, before.Sub(sdk.NewCoin(chain.Denom, s.txFee(t, res))), s.balances(t, s.taker, chain.Denom, s.usd), "a refused tx only costs the tx fee")
	})

	height = s.passProposal(t, "Unhalt usd and oth, restore the order book cap",
		s.haltMsg("MsgUnhaltDenoms", s.gov, s.usd, s.oth),
		map[string]any{
			"@type":     "/bze.tradebin.MsgUpdateParams",
			"authority": s.gov,
			"params":    s.defaults,
		})

	require.ElementsMatch(t, []string{s.usd, s.oth}, eventDenoms(s.blockTypedEvents(t, height, "DenomUnhaltedEvent")), "one DenomUnhaltedEvent per denom")
	s.requireHaltedDenoms(t)
	require.False(t, s.denomHalted(t, s.usd))
	require.False(t, s.denomHalted(t, s.oth))
	require.Equal(t, s.defaults, s.orderBook().rawParams(t), "params restored")
}

// 12. Halting the native denom would halt every market and pool: the keeper
// refuses it, so the proposal passes the vote but fails on execution and
// nothing is halted.
func (s *haltSuite) haltNative(t *testing.T) {
	_, err := helpers.SubmitAndPassProposal(s.ctx, s.bze, s.admin, helpers.Proposal{
		Messages: []any{s.haltMsg("MsgHaltDenoms", s.gov, chain.Denom)},
		Title:    "Halt ubze",
		Summary:  "e2e: halting the native denom must fail",
	})
	require.ErrorContains(t, err, "PROPOSAL_STATUS_FAILED")
	require.ErrorContains(t, err, "the native denom "+chain.Denom+" cannot be halted")
	require.False(t, s.denomHalted(t, chain.Denom))
	s.requireHaltedDenoms(t)
}

func (s *haltSuite) haltMsg(typ, authority string, denoms ...string) map[string]any {
	return map[string]any{
		"@type":     "/bze.tradebin." + typ,
		"authority": authority,
		"denoms":    denoms,
	}
}

// passProposal submits msgs as one proposal from admin, passes it and
// returns the height of the block whose EndBlock executed it.
func (s *haltSuite) passProposal(t *testing.T, title string, msgs ...any) int64 {
	t.Helper()
	from, err := s.bze.Height(s.ctx)
	require.NoError(t, err)
	id, err := helpers.SubmitAndPassProposal(s.ctx, s.bze, s.admin, helpers.Proposal{
		Messages: msgs,
		Title:    title,
		Summary:  "e2e: " + title,
	})
	require.NoError(t, err)

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
	t.Fatalf("no active_proposal event for proposal %d between heights %d and %d", id, from, to)
	return 0
}

func (s *haltSuite) blockTypedEvents(t *testing.T, height int64, name string) []helpers.Event {
	t.Helper()
	events, err := helpers.BlockEvents(s.ctx, s.bze, height, tradebinEventPrefix+name)
	require.NoError(t, err)
	return events
}

func (s *haltSuite) denomHalted(t *testing.T, denom string) bool {
	t.Helper()
	res, err := helpers.Query[struct {
		Halted bool `json:"halted"`
	}](s.ctx, s.bze, "tradebin", "denom-halted", denom)
	require.NoError(t, err)
	return res.Halted
}

func (s *haltSuite) requireHaltedDenoms(t *testing.T, want ...string) {
	t.Helper()
	res, err := helpers.Query[struct {
		Denoms []string `json:"denoms"`
	}](s.ctx, s.bze, "tradebin", "halted-denoms")
	require.NoError(t, err)
	require.ElementsMatch(t, want, res.Denoms, "halted denoms")
}

// placeSellOrder places the trader's sell order on the hlt market at a price
// nobody fills and returns the id the EndBlock saved it under.
func (s *haltSuite) placeSellOrder(t *testing.T) string {
	t.Helper()
	before := s.balance(t, s.trader.FormattedAddress(), s.hlt)
	res := s.mustExec(t, s.trader, "tradebin", "create-order", "sell", sdkmath.NewInt(haltOrderAmount).String(), haltOrderPrice, s.market)
	msgID := s.requireTxEvent(t, res, "OrderCreateMessageEvent", map[string]string{"market_id": s.market}).Attributes["message_id"]
	saved := helpers.EventsWith(filterType(s.blockEvents(t, res.Height), tradebinEventPrefix+"OrderSavedEvent"), map[string]string{"message_id": msgID})
	require.Len(t, saved, 1, "the order is saved by the EndBlock that processed message %s", msgID)
	require.Equal(t, before.SubRaw(haltOrderAmount), s.balance(t, s.trader.FormattedAddress(), s.hlt), "order amount locked")
	return saved[0].Attributes["id"]
}

type haltOrder struct {
	ID        string `json:"id"`
	MarketID  string `json:"market_id"`
	OrderType string `json:"order_type"`
}

func (s *haltSuite) traderOrders(t *testing.T) []haltOrder {
	t.Helper()
	return s.ordersOf(t, s.trader, s.market)
}

func (s *haltSuite) ordersOf(t *testing.T, u ibc.Wallet, market string) []haltOrder {
	t.Helper()
	res, err := helpers.Query[struct {
		List []haltOrder `json:"list"`
	}](s.ctx, s.bze, "tradebin", "user-market-orders", u.FormattedAddress(), market)
	require.NoError(t, err)
	return res.List
}

// restingOrder is the trader's only order on the hlt market.
func (s *haltSuite) restingOrder(t *testing.T) haltOrder {
	t.Helper()
	orders := s.traderOrders(t)
	require.Len(t, orders, 1, "trader orders: %+v", orders)
	require.Equal(t, s.market, orders[0].MarketID)
	require.Equal(t, "sell", orders[0].OrderType)
	return orders[0]
}

// haltSnapshot is the state a refused trading tx must not touch.
type haltSnapshot struct {
	pools, markets, orders string
	balances               sdk.Coins
}

func (s *haltSuite) snapshot(t *testing.T) haltSnapshot {
	t.Helper()
	raw := func(args ...string) string {
		res, err := helpers.Query[json.RawMessage](s.ctx, s.bze, append([]string{"tradebin"}, args...)...)
		require.NoError(t, err)
		return string(res)
	}
	return haltSnapshot{
		pools:    raw("all-liquidity-pools"),
		markets:  raw("all-markets"),
		orders:   raw("market-aggregated-orders", s.market, "sell") + raw("market-aggregated-orders", s.market, "buy") + raw("user-market-orders", s.trader.FormattedAddress(), s.market),
		balances: s.balances(t, s.trader, chain.Denom, s.hlt, s.usd, s.oth, s.hltLp),
	}
}

func (s *haltSuite) minNativeLiquidity(t *testing.T) sdkmath.Int {
	t.Helper()
	res, err := helpers.Query[struct {
		Params struct {
			MinNativeLiquidityForModuleSwap string `json:"minNativeLiquidityForModuleSwap"`
		} `json:"params"`
	}](s.ctx, s.bze, "tradebin", "params")
	require.NoError(t, err)
	return parseInt(res.Params.MinNativeLiquidityForModuleSwap)
}

// orderBook lends the order-book suite's params helpers (rawParams,
// updateParams) to this suite: same chain, admin and gov authority.
func (s *haltSuite) orderBook() *orderBookSuite {
	return &orderBookSuite{ammSuite: s.ammSuite, admin: s.admin, gov: s.gov}
}

// restSell places u's sell order of amount base at price on market, asserts
// the EndBlock saved it, u paid the order amount, the maker fee (on to the
// burner fee collector) and the tx fee, and returns the order id.
func (s *haltSuite) restSell(t *testing.T, u ibc.Wallet, market, base string, amount int64, price string) string {
	t.Helper()
	amt := strconv.FormatInt(amount, 10)
	before := s.balances(t, u, chain.Denom, base)
	res := s.mustExec(t, u, "tradebin", "create-order", "sell", amt, price, market)
	msgID := s.requireTxEvent(t, res, "OrderCreateMessageEvent", map[string]string{"creator": u.FormattedAddress(), "market_id": market}).Attributes["message_id"]

	saved := helpers.EventsWith(s.blockTypedEvents(t, res.Height, "OrderSavedEvent"), map[string]string{"message_id": msgID})
	require.Len(t, saved, 1, "the order is saved by the EndBlock that processed message %s", msgID)
	e := saved[0].Attributes
	require.Equal(t, []string{market, u.FormattedAddress(), "sell", amt, price}, []string{e["market_id"], e["owner"], e["order_type"], e["amount"], e["price"]})

	fee := s.params.MarketMakerFee.Amount
	require.Equal(t, fee, s.transferred(t, res.Events, s.tradebin, s.burnerFees, chain.Denom), "maker fee to the burner fee collector")
	want := before.Sub(sdk.NewInt64Coin(base, amount), sdk.NewCoin(chain.Denom, fee.Add(s.txFee(t, res))))
	require.Equal(t, want, s.balances(t, u, chain.Denom, base), "order amount, maker fee and tx fee paid")
	return e["id"]
}

// fillSell has the taker buy maker's resting sell order whole at its price:
// one execution of that order and nothing saved; the taker pays the cost, the
// taker fee (on to the burner) and the tx fee and gets the base, the maker
// gets the cost and has no order left on the market.
func (s *haltSuite) fillSell(t *testing.T, maker ibc.Wallet, market, base, orderID string, amount int64, price string) {
	t.Helper()
	amt := strconv.FormatInt(amount, 10)
	cost := mulPrice(amount, price)
	makerBefore := s.balances(t, maker, chain.Denom, base)
	takerBefore := s.balances(t, s.taker, chain.Denom, base)

	res := s.mustExec(t, s.taker, "tradebin", "create-order", "buy", amt, price, market)
	msgID := s.requireTxEvent(t, res, "OrderCreateMessageEvent", map[string]string{"creator": s.taker.FormattedAddress(), "market_id": market}).Attributes["message_id"]

	fee := s.params.MarketTakerFee.Amount
	require.Equal(t, fee.Add(cost), s.transferred(t, res.Events, s.taker.FormattedAddress(), s.tradebin, chain.Denom), "taker fee and order cost to the tradebin module")
	require.Equal(t, fee, s.transferred(t, res.Events, s.tradebin, s.burnerFees, chain.Denom), "taker fee to the burner fee collector")
	block := s.blockEvents(t, res.Height)
	require.Equal(t, fee, s.transferred(t, block, s.burnerFees, s.burner, chain.Denom), "taker fee to the burner in EndBlock")

	executed := helpers.EventsWith(filterType(block, tradebinEventPrefix+"OrderExecutedEvent"), map[string]string{"message_id": msgID})
	require.Len(t, executed, 1, "executions of message %s: %+v", msgID, executed)
	e := executed[0].Attributes
	require.Equal(t,
		[]string{orderID, market, "sell", amt, price, maker.FormattedAddress(), s.taker.FormattedAddress()},
		[]string{e["id"], e["market_id"], e["order_type"], e["amount"], e["price"], e["maker"], e["taker"]})
	require.Empty(t, helpers.EventsWith(filterType(block, tradebinEventPrefix+"OrderSavedEvent"), map[string]string{"message_id": msgID}), "nothing left to save")

	wantTaker := takerBefore.Add(sdk.NewInt64Coin(base, amount)).Sub(sdk.NewCoin(chain.Denom, cost.Add(fee).Add(s.txFee(t, res))))
	require.Equal(t, wantTaker, s.balances(t, s.taker, chain.Denom, base), "taker balances")
	require.Equal(t, makerBefore.Add(sdk.NewCoin(chain.Denom, cost)), s.balances(t, maker, chain.Denom, base), "maker balances")
	require.Empty(t, s.ordersOf(t, maker, market), "maker's order left the book")
}

// orderAmount is the amount left on a resting order.
func (s *haltSuite) orderAmount(t *testing.T, market, typ, id string) string {
	t.Helper()
	res, err := helpers.Query[struct {
		Order struct {
			Amount string `json:"amount"`
		} `json:"order"`
	}](s.ctx, s.bze, "tradebin", "market-order", market, typ, id)
	require.NoError(t, err)
	return res.Order.Amount
}

// level is the aggregated amount at price on one side of market, "" when
// nothing rests there.
func (s *haltSuite) level(t *testing.T, market, typ, price string) string {
	t.Helper()
	res, err := helpers.Query[struct {
		List []struct {
			Amount string `json:"amount"`
			Price  string `json:"price"`
		} `json:"list"`
	}](s.ctx, s.bze, "tradebin", "market-aggregated-orders", market, typ)
	require.NoError(t, err)
	for _, l := range res.List {
		if l.Price == price {
			return l.Amount
		}
	}
	return ""
}

func eventDenoms(events []helpers.Event) []string {
	out := make([]string, 0, len(events))
	for _, e := range events {
		out = append(out, e.Attributes["denom"])
	}
	return out
}
