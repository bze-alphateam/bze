package interchaintest_test

import (
	"context"
	"encoding/json"
	"slices"
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

// The order book matches an order only against resting orders at exactly its
// price, inside the tx that carries it: every outcome is read from the tx
// result, with no block wait. Amounts are in ob
// (the base), prices in ubze per ob. Every price below has a minimum order
// amount of 2 (2 × ⌈1/price⌉) except 2.35's dust step and the 0.333 refusal.
const (
	obAdminFunds = 100_000_000_000 // 100,000 BZE: the denom and market create fees, proposals
	obUserFunds  = 10_000_000_000  // 10,000 BZE each
	obMint       = 100_000_000
	obUserBase   = 10_000_000 // ob sent to each maker and the taker

	obTxGas   = 500_000
	obFillGas = 1_000_000

	obCodespace = "tradebin"

	// step 16: deep price level
	obDeepMakers   = 10
	obDeepMaxFills = 5
	obDeepAmount   = 2   // ob per maker order, the minimum at obDeepPrice
	obDeepPrice    = "3" // between the resting buy at 1 and sell at 6
	obDeepFunds    = 1_000_000_000
	// one taker filling obDeepMaxFills orders: about 102k message gas for the
	// first fill and 46k for each next one (real bank, measured by the app
	// package's TestTradebin_TakerGasPerFill) plus the ante handler; filling
	// all obDeepMakers would take about 620k
	obDeepTakerGas = 550_000

	// step 18: crowded price level
	obCrowdWindow   = 2
	obCrowdExtraGas = 1_000_000
	obCrowdMakers   = 5
	obCrowdAmount   = 10
	obCrowdPrice    = "4.5" // between the resting buy at 1 and sell at 5.5
	obCrowdTxGas    = 2_500_000
	obCrowdShortGas = 1_500_000 // below the 2,000,000 the fifth owner pays
	obCrowdGasSlack = 100_000   // the gas of two placements differs by a few store reads
	// timeout_commit is 1s: a block takes about that long
	obMaxBlockTime = 5 * time.Second
)

// TestOrderBook runs the tradebin order-book engine end to end on one market,
// ob/ubze: resting orders, partial fills, FIFO at one price, remainders saved
// or refunded, the minimum kept on a resting order, dust at a fractional
// price, MsgFillOrders on buy orders with its refunds, cancels and their
// refusals, a cancel followed by a fill at its price in one tx, crossed-book
// and placement refusals, a self-trade, a top-up of a resting order (to the
// back of its price level, under a new id) and the
// extra gas of new owners at a crowded price level. Every step asserts the exact
// balances of both makers and the taker, the trading fees on their way to the
// burner, the engine events of the tx and the order book; the suite ends with
// the tradebin module holding exactly the resting orders plus the stored dust.
// Subtests run in order and share state.
func TestOrderBook(t *testing.T) {
	if testing.Short() {
		t.Skip("e2e: skipped in short mode")
	}
	ctx := context.Background()
	net := chain.Start(t, ctx)
	s := setupOrderBook(t, ctx, net.Chain)

	t.Run("0 create market", s.createMarket)
	t.Run("1 resting buy order", s.restingBuy)
	t.Run("2 partial fill of a resting order", s.partialFill)
	t.Run("3 FIFO at one price", s.fifo)
	t.Run("4 remainder saved as a new order", s.remainderSaved)
	t.Run("5 remainder below the minimum refunded", s.remainderRefunded)
	t.Run("6 resting order keeps the minimum", s.keepsMinimum)
	t.Run("7 dust at a fractional price", s.dust)
	t.Run("8 fill buy orders at several prices", s.fillBuyOrders)
	t.Run("9 fill orders refunds", s.fillRefunds)
	t.Run("10 cancel orders", s.cancelOrders)
	t.Run("11 cancel refusals", s.cancelRefusals)
	t.Run("12 cancel takes effect in its tx", s.cancelInTx)
	t.Run("13 crossed book refusals", s.crossedBook)
	t.Run("14 placement refusals", s.placementRefusals)
	t.Run("15 self-trade", s.selfTrade)
	t.Run("16 deep price level, bounded taker gas", s.deepLevel)
	t.Run("17 top-up moves to the back under a new id", s.topUp)
	t.Run("18 crowded price level costs new owners more gas", s.crowdedLevel)
	t.Run("19 module holds the book and the dust", s.conservation)
}

type orderBookSuite struct {
	// the AMM suite's query and tx helpers (balances, events, params)
	*ammSuite

	admin, maker1, maker2, taker ibc.Wallet
	ob                           string
	gov                          string
	baseline                     sdk.Coins // tradebin module balances once the market exists

	maker1Buy  string // step 1: maker1's buy at 2, partially filled until step 10
	filledSell string // step 3: maker1's sell at 3, filled whole
	maker2Sell string // step 6: maker2's sell at 4 that keeps the minimum
}

func setupOrderBook(t *testing.T, ctx context.Context, bze *cosmos.CosmosChain) *orderBookSuite {
	s := &orderBookSuite{ammSuite: &ammSuite{ctx: ctx, bze: bze}}

	s.admin = interchaintest.GetAndFundTestUsers(t, ctx, "ob-admin", sdkmath.NewInt(obAdminFunds), bze)[0]
	users := interchaintest.GetAndFundTestUsers(t, ctx, "ob", sdkmath.NewInt(obUserFunds), bze, bze, bze)
	s.maker1, s.maker2, s.taker = users[0], users[1], users[2]
	require.NoError(t, testutil.WaitForBlocks(ctx, 2, bze))

	var err error
	s.gov, err = helpers.GovAuthority(ctx, bze)
	require.NoError(t, err)
	for name, addr := range map[string]*string{
		"tradebin":          &s.tradebin,
		"burner":            &s.burner,
		"txfeecollector_cp": &s.cpFees,
	} {
		*addr, err = helpers.ModuleAddress(ctx, bze, name)
		require.NoError(t, err)
	}
	s.params = s.queryParams(t)

	s.ob = "factory/" + s.admin.FormattedAddress() + "/obk"
	s.market = s.ob + "/" + chain.Denom
	s.mustExec(t, s.admin, "tokenfactory", "create-denom", "obk")
	s.mustExec(t, s.admin, "tokenfactory", "mint", coinStr(obMint, s.ob))
	for _, u := range s.users() {
		s.mustExec(t, s.admin, "bank", "send", s.admin.FormattedAddress(), u.FormattedAddress(), coinStr(obUserBase, s.ob))
	}
	return s
}

func (s *orderBookSuite) users() []ibc.Wallet {
	return []ibc.Wallet{s.maker1, s.maker2, s.taker}
}

// 0. The creator pays the market create fee from the params; it goes through
// the tradebin module to the community-pool fee collector.
func (s *orderBookSuite) createMarket(t *testing.T) {
	fee := s.params.CreateMarketFee
	before := s.balances(t, s.admin, chain.Denom, s.ob)

	res := s.mustExecWith(t, s.admin, []helpers.TxOption{helpers.WithGas(obTxGas)}, "tradebin", "create-market", s.ob, chain.Denom)

	s.requireTxEvent(t, res, "MarketCreatedEvent", map[string]string{"creator": s.admin.FormattedAddress(), "base": s.ob, "quote": chain.Denom})
	require.Equal(t, fee.Amount, s.transferred(t, res.Events, s.admin.FormattedAddress(), s.tradebin, chain.Denom), "create fee captured")
	require.Equal(t, fee.Amount, s.transferred(t, res.Events, s.tradebin, s.cpFees, chain.Denom), "create fee to the community pool collector")
	require.Equal(t, before.Sub(fee.AddAmount(s.txFee(t, res))), s.balances(t, s.admin, chain.Denom, s.ob), "admin balances")

	s.baseline = s.moduleBalances(t)
	s.requireBook(t, obBook{})
}

// 1. A buy order with no sell at its price rests: amount × price of ubze is
// locked and the maker fee goes to the burner.
func (s *orderBookSuite) restingBuy(t *testing.T) {
	want := s.snapshot(t)
	res, orderID := s.place(t, s.maker1, "buy", 1000, "2")
	s.requireFees(t, res, s.maker1, s.makerFee(), sdk.NewInt64Coin(chain.Denom, 2000))
	want.paid(t, s, s.maker1, res, s.makerFee())
	want.move(s.maker1, chain.Denom, -2000)

	s.maker1Buy = s.requireSaved(t, res, orderID, s.maker1, "buy", "1000", "2")
	s.requireBalances(t, want)
	s.requireBook(t, obBook{Buy: map[string]string{"2": "1000"}})
	s.requireOrders(t, s.maker1, obOrder{ID: s.maker1Buy, OrderType: "buy", Amount: "1000", Price: "2", Owner: s.maker1.FormattedAddress()})
}

// 2. The taker sells less than the resting buy: the order stays with the
// rest, the maker gets the ob, the taker the ubze, and a history record is
// written. Selling at a price with a resting buy is taking: the taker fee.
func (s *orderBookSuite) partialFill(t *testing.T) {
	want, history := s.snapshot(t), s.history(t)
	res, orderID := s.place(t, s.taker, "sell", 400, "2")
	s.requireFees(t, res, s.taker, s.takerFee(), sdk.NewInt64Coin(s.ob, 400))
	want.paid(t, s, s.taker, res, s.takerFee())

	s.requireExecuted(t, res, orderID, s.taker, obExec{s.maker1Buy, s.maker1, "buy", "400", "2"})
	s.requireNotSaved(t, res, orderID)
	want.move(s.taker, s.ob, -400)
	want.move(s.taker, chain.Denom, 800)
	want.move(s.maker1, s.ob, 400)

	s.requireBalances(t, want)
	s.requireBook(t, obBook{Buy: map[string]string{"2": "600"}})
	s.requireOrders(t, s.maker1, obOrder{ID: s.maker1Buy, OrderType: "buy", Amount: "600", Price: "2", Owner: s.maker1.FormattedAddress()})
	s.requireNewHistory(t, history, obHistory{"buy", "400", "2", s.maker1.FormattedAddress(), s.taker.FormattedAddress()})
}

// 3. Two sells at one price fill in the order they were placed: a buy for
// one and a half orders fills maker1's whole and half of maker2's.
func (s *orderBookSuite) fifo(t *testing.T) {
	want, history := s.snapshot(t), s.history(t)
	res1, order1 := s.place(t, s.maker1, "sell", 100, "3")
	s.requireFees(t, res1, s.maker1, s.makerFee(), sdk.NewInt64Coin(s.ob, 100))
	s.filledSell = s.requireSaved(t, res1, order1, s.maker1, "sell", "100", "3")
	res2, order2 := s.place(t, s.maker2, "sell", 100, "3")
	s.requireFees(t, res2, s.maker2, s.makerFee(), sdk.NewInt64Coin(s.ob, 100))
	maker2Sell := s.requireSaved(t, res2, order2, s.maker2, "sell", "100", "3")
	want.paid(t, s, s.maker1, res1, s.makerFee())
	want.paid(t, s, s.maker2, res2, s.makerFee())
	want.move(s.maker1, s.ob, -100)
	want.move(s.maker2, s.ob, -100)

	res, orderID := s.place(t, s.taker, "buy", 150, "3")
	s.requireFees(t, res, s.taker, s.takerFee(), sdk.NewInt64Coin(chain.Denom, 450))
	want.paid(t, s, s.taker, res, s.takerFee())
	s.requireExecuted(t, res, orderID, s.taker,
		obExec{s.filledSell, s.maker1, "sell", "100", "3"},
		obExec{maker2Sell, s.maker2, "sell", "50", "3"},
	)
	want.move(s.taker, chain.Denom, -450)
	want.move(s.taker, s.ob, 150)
	want.move(s.maker1, chain.Denom, 300)
	want.move(s.maker2, chain.Denom, 150)

	s.requireBalances(t, want)
	s.requireBook(t, obBook{Buy: map[string]string{"2": "600"}, Sell: map[string]string{"3": "50"}})
	_, found := s.marketOrder(t, "sell", s.filledSell)
	require.False(t, found, "maker1's filled sell is gone")
	s.requireOrders(t, s.maker2, obOrder{ID: maker2Sell, OrderType: "sell", Amount: "50", Price: "3", Owner: s.maker2.FormattedAddress()})
	s.requireNewHistory(t, history,
		obHistory{"sell", "100", "3", s.maker1.FormattedAddress(), s.taker.FormattedAddress()},
		obHistory{"sell", "50", "3", s.maker2.FormattedAddress(), s.taker.FormattedAddress()},
	)
}

// 4. A buy larger than the sell level takes all of it; the remainder is at
// least the minimum and is saved as a resting buy.
func (s *orderBookSuite) remainderSaved(t *testing.T) {
	want := s.snapshot(t)
	maker2Sell := s.userOrders(t, s.maker2)[0].ID
	res, orderID := s.place(t, s.taker, "buy", 200, "3")
	s.requireFees(t, res, s.taker, s.takerFee(), sdk.NewInt64Coin(chain.Denom, 600))
	want.paid(t, s, s.taker, res, s.takerFee())

	s.requireExecuted(t, res, orderID, s.taker, obExec{maker2Sell, s.maker2, "sell", "50", "3"})
	saved := s.requireSaved(t, res, orderID, s.taker, "buy", "150", "3")
	want.move(s.taker, chain.Denom, -600)
	want.move(s.taker, s.ob, 50)
	want.move(s.maker2, chain.Denom, 150)

	s.requireBalances(t, want)
	s.requireBook(t, obBook{Buy: map[string]string{"3": "150", "2": "600"}})
	s.requireOrders(t, s.maker2)
	s.requireOrders(t, s.taker, obOrder{ID: saved, OrderType: "buy", Amount: "150", Price: "3", Owner: s.taker.FormattedAddress()})
}

// 5. A sell one unit above the buy level fills it; the 1 ob left is below the
// minimum (2), so it is refunded instead of saved.
func (s *orderBookSuite) remainderRefunded(t *testing.T) {
	want := s.snapshot(t)
	takerBuy := s.userOrders(t, s.taker)[0].ID
	res, orderID := s.place(t, s.maker1, "sell", 151, "3")
	s.requireFees(t, res, s.maker1, s.takerFee(), sdk.NewInt64Coin(s.ob, 151))
	want.paid(t, s, s.maker1, res, s.takerFee())

	s.requireExecuted(t, res, orderID, s.maker1, obExec{takerBuy, s.taker, "buy", "150", "3"})
	s.requireNotSaved(t, res, orderID)
	require.Equal(t, sdkmath.OneInt(), s.transferred(t, res.Events, s.tradebin, s.maker1.FormattedAddress(), s.ob), "1 ob refunded")
	s.requireRefunded(t, res, orderID, s.maker1, "sell", "1", "1"+s.ob, "below_min_amount")
	want.move(s.maker1, s.ob, -150)
	want.move(s.maker1, chain.Denom, 450)
	want.move(s.taker, s.ob, 150)

	s.requireBalances(t, want)
	s.requireBook(t, obBook{Buy: map[string]string{"2": "600"}})
	s.requireOrders(t, s.taker)
}

// 6. A buy that would leave a sell order below the minimum executes only up
// to the minimum: the order keeps 2, and the buyer's leftover 1 is refunded.
func (s *orderBookSuite) keepsMinimum(t *testing.T) {
	want := s.snapshot(t)
	res1, order1 := s.place(t, s.maker2, "sell", 600, "4")
	s.requireFees(t, res1, s.maker2, s.makerFee(), sdk.NewInt64Coin(s.ob, 600))
	s.maker2Sell = s.requireSaved(t, res1, order1, s.maker2, "sell", "600", "4")
	want.paid(t, s, s.maker2, res1, s.makerFee())
	want.move(s.maker2, s.ob, -600)

	res, orderID := s.place(t, s.taker, "buy", 599, "4")
	s.requireFees(t, res, s.taker, s.takerFee(), sdk.NewInt64Coin(chain.Denom, 599*4))
	want.paid(t, s, s.taker, res, s.takerFee())
	s.requireExecuted(t, res, orderID, s.taker, obExec{s.maker2Sell, s.maker2, "sell", "598", "4"})
	s.requireNotSaved(t, res, orderID)
	require.Equal(t, sdkmath.NewInt(4), s.transferred(t, res.Events, s.tradebin, s.taker.FormattedAddress(), chain.Denom), "1 ob × 4 refunded")
	s.requireRefunded(t, res, orderID, s.taker, "buy", "1", "4"+chain.Denom, "below_min_amount")
	want.move(s.taker, chain.Denom, -598*4)
	want.move(s.taker, s.ob, 598)
	want.move(s.maker2, chain.Denom, 598*4)

	s.requireBalances(t, want)
	s.requireBook(t, obBook{Buy: map[string]string{"2": "600"}, Sell: map[string]string{"4": "2"}})
	s.requireOrders(t, s.maker2, obOrder{ID: s.maker2Sell, OrderType: "sell", Amount: "2", Price: "4", Owner: s.maker2.FormattedAddress()})
}

// 7. At 2.35, 10 ob cost 23.5 ubze. A payer without dust pays 24 and keeps
// 0.5 as stored dust; the next payment takes the 0.5 from it and pays 23. A
// receiver gets 23 and stores 0.5; once its dust reaches 1 it gets 24.
// maker2's three buys at one price are one order: the first is saved, the
// next two are added to it, each time under the new buy's id; each still pays
// its own maker fee and escrow.
func (s *orderBookSuite) dust(t *testing.T) {
	want := s.snapshot(t)
	require.Empty(t, s.userDust(t, s.maker2))
	require.Empty(t, s.userDust(t, s.taker))

	var buy string
	for i, c := range []struct {
		paid int64
		dust map[string]string
	}{
		{24, obDust("0.5")},
		{23, obDust("")},
		{24, obDust("0.5")},
	} {
		res, orderID := s.place(t, s.maker2, "buy", 10, "2.35")
		s.requireFees(t, res, s.maker2, s.makerFee(), sdk.NewInt64Coin(chain.Denom, c.paid))
		want.paid(t, s, s.maker2, res, s.makerFee())
		want.move(s.maker2, chain.Denom, -c.paid)
		if i == 0 {
			buy = s.requireSaved(t, res, orderID, s.maker2, "buy", "10", "2.35")
		} else {
			require.NotEqual(t, buy, orderID, "a buy at the same price gets a new id")
			s.requireAmended(t, res, orderID, buy, s.maker2, "buy", "2.35", "10", strconv.Itoa(10*(i+1)))
			buy = orderID
		}
		require.Equal(t, c.dust, s.userDust(t, s.maker2), "maker2 dust after paying %d", c.paid)
	}
	s.requireBook(t, obBook{Buy: map[string]string{"2.35": "30", "2": "600"}, Sell: map[string]string{"4": "2"}})
	s.requireOrders(t, s.maker2,
		obOrder{ID: buy, OrderType: "buy", Amount: "30", Price: "2.35", Owner: s.maker2.FormattedAddress()},
		obOrder{ID: s.maker2Sell, OrderType: "sell", Amount: "2", Price: "4", Owner: s.maker2.FormattedAddress()},
	)

	// 23 (dust 0.5), 23 + 1 (dust 1 paid out), 23 (dust 0.5)
	for _, received := range []int64{23, 24, 23} {
		res, orderID := s.place(t, s.taker, "sell", 10, "2.35")
		s.requireFees(t, res, s.taker, s.takerFee(), sdk.NewInt64Coin(s.ob, 10))
		want.paid(t, s, s.taker, res, s.takerFee())
		s.requireExecuted(t, res, orderID, s.taker, obExec{buy, s.maker2, "buy", "10", "2.35"})
		s.requireNotSaved(t, res, orderID)
		require.Equal(t, sdkmath.NewInt(received), s.transferred(t, res.Events, s.tradebin, s.taker.FormattedAddress(), chain.Denom), "taker receipt")
		want.move(s.taker, s.ob, -10)
		want.move(s.taker, chain.Denom, received)
		want.move(s.maker2, s.ob, 10)
	}

	s.requireBalances(t, want)
	require.Equal(t, obDust("0.5"), s.userDust(t, s.taker), "taker dust")
	require.Equal(t, obDust("0.5"), s.userDust(t, s.maker2), "maker2 dust")
	s.requireBook(t, obBook{Buy: map[string]string{"2": "600"}, Sell: map[string]string{"4": "2"}})
	s.requireOrders(t, s.maker2, obOrder{ID: s.maker2Sell, OrderType: "sell", Amount: "2", Price: "4", Owner: s.maker2.FormattedAddress()})
}

// 8. MsgFillOrders on buy orders sells ob at two prices in one message: one
// order (and order id) per price, one taker fee for the whole message.
func (s *orderBookSuite) fillBuyOrders(t *testing.T) {
	want, history := s.snapshot(t), s.history(t)
	var buy18 string
	for _, price := range []string{"1.8", "1.5"} {
		res, orderID := s.place(t, s.maker1, "buy", 100, price)
		s.requireFees(t, res, s.maker1, s.makerFee(), sdk.NewCoin(chain.Denom, mulPrice(100, price)))
		want.paid(t, s, s.maker1, res, s.makerFee())
		want.move(s.maker1, chain.Denom, -mulPrice(100, price).Int64())
		id := s.requireSaved(t, res, orderID, s.maker1, "buy", "100", price)
		if price == "1.8" {
			buy18 = id
		}
	}

	res := s.mustExecMsgs(t, s.taker, obFillGas, s.fillMsg(s.taker, "buy", obItem{"2", "100"}, obItem{"1.8", "100"}))
	s.requireFees(t, res, s.taker, s.takerFee(), sdk.NewInt64Coin(s.ob, 200))
	want.paid(t, s, s.taker, res, s.takerFee())
	ids := s.fillIDs(t, res, 2)
	s.requireExecuted(t, res, ids[0], s.taker, obExec{s.maker1Buy, s.maker1, "buy", "100", "2"})
	s.requireExecuted(t, res, ids[1], s.taker, obExec{buy18, s.maker1, "buy", "100", "1.8"})
	want.move(s.taker, s.ob, -200)
	want.move(s.taker, chain.Denom, 200+180)
	want.move(s.maker1, s.ob, 200)

	s.requireBalances(t, want)
	s.requireBook(t, obBook{Buy: map[string]string{"2": "500", "1.5": "100"}, Sell: map[string]string{"4": "2"}})
	s.requireNewHistory(t, history,
		obHistory{"buy", "100", "2", s.maker1.FormattedAddress(), s.taker.FormattedAddress()},
		obHistory{"buy", "100", "1.8", s.maker1.FormattedAddress(), s.taker.FormattedAddress()},
	)
}

// 9. MsgFillOrders refunds what it can not fill: a price with nothing resting
// is refunded whole, a level smaller than the item refunds the rest, an item
// below the minimum is skipped (no order id, nothing charged). A message whose
// items are all skipped is refused.
func (s *orderBookSuite) fillRefunds(t *testing.T) {
	want := s.snapshot(t)
	buy15 := s.userOrders(t, s.maker1)
	require.Len(t, buy15, 2)
	var buy15ID string
	for _, o := range buy15 {
		if o.Price == "1.5" {
			buy15ID = o.ID
		}
	}

	res := s.mustExecMsgs(t, s.taker, obFillGas, s.fillMsg(s.taker, "buy", obItem{"1.7", "50"}, obItem{"1.5", "150"}, obItem{"1.2", "1"}))
	s.requireFees(t, res, s.taker, s.takerFee(), sdk.NewInt64Coin(s.ob, 200))
	want.paid(t, s, s.taker, res, s.takerFee())
	ids := s.fillIDs(t, res, 2)
	s.requireExecuted(t, res, ids[0], s.taker)
	s.requireExecuted(t, res, ids[1], s.taker, obExec{buy15ID, s.maker1, "buy", "100", "1.5"})
	require.Equal(t, sdkmath.NewInt(50+50), s.transferred(t, res.Events, s.tradebin, s.taker.FormattedAddress(), s.ob), "50 ob refunded at 1.7, 50 at 1.5")
	s.requireRefunded(t, res, ids[0], s.taker, "sell", "50", "50"+s.ob, "no_match")
	s.requireRefunded(t, res, ids[1], s.taker, "sell", "50", "50"+s.ob, "unfilled_remainder")
	want.move(s.taker, s.ob, -100)
	want.move(s.taker, chain.Denom, 150)
	want.move(s.maker1, s.ob, 100)
	s.requireBalances(t, want)
	s.requireBook(t, obBook{Buy: map[string]string{"2": "500"}, Sell: map[string]string{"4": "2"}})

	before := s.snapshot(t)
	res, err := helpers.ExecMsgs(s.ctx, s.bze, s.taker, []helpers.TxOption{helpers.WithGas(obFillGas)}, s.fillMsg(s.taker, "buy", obItem{"1.2", "1"}))
	s.requireRefused(t, res, err, 4011, "invalid orders to fill")
	before.paid(t, s, s.taker, res, sdkmath.ZeroInt())
	s.requireBalances(t, before)
}

// 10. Cancelling maker1's partly filled buy refunds only what is left. A buy
// at 2.35 locked 24 (23.5 + 0.5 stored dust); its cancel refunds 23.5, which
// with the stored 0.5 pays back exactly 24 and clears the dust.
func (s *orderBookSuite) cancelOrders(t *testing.T) {
	want := s.snapshot(t)
	res := s.mustExec(t, s.maker1, "tradebin", "cancel-order", s.market, s.maker1Buy, "buy")
	s.requireCanceled(t, res, s.maker1, s.maker1Buy, "buy", "500", "2")
	want.paid(t, s, s.maker1, res, sdkmath.ZeroInt())
	want.move(s.maker1, chain.Denom, 1000)
	s.requireBalances(t, want)
	s.requireBook(t, obBook{Sell: map[string]string{"4": "2"}})
	s.requireOrders(t, s.maker1)

	require.Empty(t, s.userDust(t, s.maker1))
	res, orderID := s.place(t, s.maker1, "buy", 10, "2.35")
	s.requireFees(t, res, s.maker1, s.makerFee(), sdk.NewInt64Coin(chain.Denom, 24))
	id := s.requireSaved(t, res, orderID, s.maker1, "buy", "10", "2.35")
	require.Equal(t, obDust("0.5"), s.userDust(t, s.maker1))
	want.paid(t, s, s.maker1, res, s.makerFee())
	want.move(s.maker1, chain.Denom, -24)

	res = s.mustExec(t, s.maker1, "tradebin", "cancel-order", s.market, id, "buy")
	s.requireCanceled(t, res, s.maker1, id, "buy", "10", "2.35")
	require.Equal(t, sdkmath.NewInt(24), s.transferred(t, res.Events, s.tradebin, s.maker1.FormattedAddress(), chain.Denom), "23 refunded + 1 of dust")
	want.paid(t, s, s.maker1, res, sdkmath.ZeroInt())
	want.move(s.maker1, chain.Denom, 24)
	require.Empty(t, s.userDust(t, s.maker1), "dust settled")
	s.requireBalances(t, want)
	s.requireBook(t, obBook{Sell: map[string]string{"4": "2"}})
}

// 11. Only the owner cancels an order, only a resting order can be
// cancelled, and a second cancel of the same order in one tx finds it gone:
// the tx is refused and reverted, so the first cancel is undone and the order
// stays.
func (s *orderBookSuite) cancelRefusals(t *testing.T) {
	for _, c := range []struct {
		name   string
		user   ibc.Wallet
		id     string
		code   uint32
		reason string
	}{
		{"not the owner", s.taker, s.maker2Sell, 4009, "not authorized"},
		{"unknown order", s.maker2, "999999999999999999999999", 4008, "order not found"},
		{"filled order", s.maker1, s.filledSell, 4008, "order not found"},
	} {
		t.Run(c.name, func(t *testing.T) {
			want, book := s.snapshot(t), s.book(t)
			res, err := helpers.ExecTxWith(s.ctx, s.bze, c.user, []helpers.TxOption{helpers.WithGas(obTxGas)}, "tradebin", "cancel-order", s.market, c.id, "sell")
			s.requireRefused(t, res, err, c.code, c.reason)
			want.paid(t, s, c.user, res, sdkmath.ZeroInt())
			s.requireBalances(t, want)
			require.Equal(t, book, s.book(t), "order book changed")
		})
	}

	t.Run("duplicate cancel", func(t *testing.T) {
		want, book := s.snapshot(t), s.book(t)
		cancel := s.cancelMsg(s.maker2, "sell", s.maker2Sell)
		res, err := helpers.ExecMsgs(s.ctx, s.bze, s.maker2, []helpers.TxOption{helpers.WithGas(obTxGas)}, cancel, cancel)
		s.requireRefused(t, res, err, 4008, "order not found")
		want.paid(t, s, s.maker2, res, sdkmath.ZeroInt())
		s.requireBalances(t, want)
		require.Equal(t, book, s.book(t), "order book changed")
		s.requireOrders(t, s.maker2, obOrder{ID: s.maker2Sell, OrderType: "sell", Amount: "2", Price: "4", Owner: s.maker2.FormattedAddress()})
	})
}

// 12. maker2 cancels its resting sell and, in the same tx, fills sell orders
// at its price. The cancel executes first and takes effect at once: the order
// leaves the book and its ob comes back, so the fill finds nothing and its
// ubze is refunded whole (no_match).
func (s *orderBookSuite) cancelInTx(t *testing.T) {
	want := s.snapshot(t)
	res := s.mustExecMsgs(t, s.maker2, obFillGas,
		s.cancelMsg(s.maker2, "sell", s.maker2Sell),
		s.fillMsg(s.maker2, "sell", obItem{"4", "2"}),
	)
	s.requireFees(t, res, s.maker2, s.takerFee(), sdk.NewInt64Coin(chain.Denom, 8))
	want.paid(t, s, s.maker2, res, s.takerFee())
	s.requireCanceled(t, res, s.maker2, s.maker2Sell, "sell", "2", "4")
	fill := s.fillIDs(t, res, 1)[0]
	require.Greater(t, fill, s.maker2Sell, "the fill gets a new order id")

	s.requireExecuted(t, res, fill, s.maker2)
	s.requireNotSaved(t, res, fill)
	s.requireRefunded(t, res, fill, s.maker2, "buy", "2", "8"+chain.Denom, "no_match")
	require.Equal(t, sdkmath.NewInt(2), s.transferred(t, res.Events, s.tradebin, s.maker2.FormattedAddress(), s.ob), "the cancelled order's ob")
	require.Equal(t, sdkmath.NewInt(8), s.transferred(t, res.Events, s.tradebin, s.maker2.FormattedAddress(), chain.Denom), "the fill's ubze")
	want.move(s.maker2, s.ob, 2)

	s.requireBalances(t, want)
	s.requireBook(t, obBook{})
	s.requireOrders(t, s.maker2)
}

// 13. A buy above the best resting sell, a sell below the best resting buy,
// and a sell below a buy placed earlier in the same tx are refused (the
// refusal reverts the whole tx, the buy included).
func (s *orderBookSuite) crossedBook(t *testing.T) {
	want := s.snapshot(t)
	res, orderID := s.place(t, s.maker1, "sell", 100, "6")
	s.requireFees(t, res, s.maker1, s.makerFee(), sdk.NewInt64Coin(s.ob, 100))
	s.requireSaved(t, res, orderID, s.maker1, "sell", "100", "6")
	want.paid(t, s, s.maker1, res, s.makerFee())
	want.move(s.maker1, s.ob, -100)
	res, orderID = s.place(t, s.maker2, "buy", 100, "1")
	s.requireFees(t, res, s.maker2, s.makerFee(), sdk.NewInt64Coin(chain.Denom, 100))
	s.requireSaved(t, res, orderID, s.maker2, "buy", "100", "1")
	want.paid(t, s, s.maker2, res, s.makerFee())
	want.move(s.maker2, chain.Denom, -100)
	s.requireBalances(t, want)
	s.requireBook(t, obBook{Buy: map[string]string{"1": "100"}, Sell: map[string]string{"6": "100"}})

	for _, c := range []struct {
		name   string
		msgs   []any
		reason string
	}{
		{"buy above the best sell", []any{s.orderMsg(s.taker, "buy", 10, "7")}, "buying price is invalid. A better price is available: 6.000000000000000000"},
		{"sell below the best buy", []any{s.orderMsg(s.taker, "sell", 10, "0.9")}, "selling price is invalid. A better price is available: 1.000000000000000000"},
		{"sell below a buy placed earlier in the tx", []any{s.orderMsg(s.taker, "buy", 10, "5"), s.orderMsg(s.taker, "sell", 10, "4.5")}, "selling price is invalid. A better price is available: 5.000000000000000000"},
	} {
		t.Run(c.name, func(t *testing.T) {
			want, book := s.snapshot(t), s.book(t)
			res, err := helpers.ExecMsgs(s.ctx, s.bze, s.taker, []helpers.TxOption{helpers.WithGas(obFillGas)}, c.msgs...)
			s.requireRefused(t, res, err, 4004, c.reason)
			want.paid(t, s, s.taker, res, sdkmath.ZeroInt())
			s.requireBalances(t, want)
			require.Equal(t, book, s.book(t), "order book changed")
		})
	}
}

// 14. Orders the msg server or ValidateBasic refuse: nothing is locked, a
// refusal in the msg server costs the tx fee only, one in
// ValidateBasic (CheckTx) costs nothing.
func (s *orderBookSuite) placementRefusals(t *testing.T) {
	order := func(typ string, amount int64, price, market string) map[string]any {
		m := s.orderMsg(s.taker, typ, amount, price)
		m["market_id"] = market
		return m
	}
	for _, c := range []struct {
		name    string
		msg     map[string]any
		code    uint32
		reason  string
		checkTx bool
	}{
		// 2 × ⌈1 / 0.333⌉ = 8
		{"below the minimum amount", order("buy", 7, "0.333", s.market), 4003, "amount should be bigger than: 8", false},
		{"price at the minimum price", order("buy", 10, "0.0000000001", s.market), 4004, "price should be higher than 0.000000000100000000", true},
		{"unknown market", order("buy", 10, "1", "nope/"+chain.Denom), 4006, "market id: nope/" + chain.Denom, false},
		{"not enough funds", order("buy", 1_000_000_000_000_000, "1", s.market), 5, "insufficient funds", false},
		{"bad order type", order("hold", 10, "1", s.market), 4002, "invalid order type", true},
		{"empty market id", order("buy", 10, "1", ""), 4005, "no market id provided", true},
	} {
		t.Run(c.name, func(t *testing.T) {
			want, book := s.snapshot(t), s.book(t)
			res, err := helpers.ExecMsgs(s.ctx, s.bze, s.taker, []helpers.TxOption{helpers.WithGas(obTxGas)}, c.msg)
			s.requireRefused(t, res, err, c.code, c.reason)
			if c.code == 5 {
				require.Equal(t, "sdk", res.Codespace, res.RawLog)
			}
			require.Equal(t, c.checkTx, res.Height == 0, "refused by CheckTx: want %v, got height %d", c.checkTx, res.Height)
			want.paid(t, s, s.taker, res, sdkmath.ZeroInt())
			s.requireBalances(t, want)
			require.Equal(t, book, s.book(t), "order book changed")
		})
	}
}

// 15. maker1 sells and then buys the same 50 ob at one price: the order is
// filled by its own owner, and maker1 is left with only the maker fee, the
// taker fee and the tx fees spent.
func (s *orderBookSuite) selfTrade(t *testing.T) {
	want, history := s.snapshot(t), s.history(t)
	res1, order1 := s.place(t, s.maker1, "sell", 50, "5")
	s.requireFees(t, res1, s.maker1, s.makerFee(), sdk.NewInt64Coin(s.ob, 50))
	sell := s.requireSaved(t, res1, order1, s.maker1, "sell", "50", "5")
	res2, order2 := s.place(t, s.maker1, "buy", 50, "5")
	s.requireFees(t, res2, s.maker1, s.takerFee(), sdk.NewInt64Coin(chain.Denom, 250))
	s.requireExecuted(t, res2, order2, s.maker1, obExec{sell, s.maker1, "sell", "50", "5"})
	s.requireNotSaved(t, res2, order2)

	want.paid(t, s, s.maker1, res1, s.makerFee())
	want.paid(t, s, s.maker1, res2, s.takerFee())
	s.requireBalances(t, want)
	s.requireBook(t, obBook{Buy: map[string]string{"1": "100"}, Sell: map[string]string{"6": "100"}})
	s.requireNewHistory(t, history, obHistory{"sell", "50", "5", s.maker1.FormattedAddress(), s.maker1.FormattedAddress()})
}

// 16. A price level of small orders from many distinct owners. With
// maxFillsPerOrder at obDeepMaxFills, one taker crossing the whole level fills
// the first obDeepMaxFills orders in FIFO order in its tx, for a bounded gas,
// and gets the rest of its amount back with reason max_fills_reached; the
// next taker fills the rest. Blocks keep their pace.
func (s *orderBookSuite) deepLevel(t *testing.T) {
	maxFills := s.tradebinParam(t, "maxFillsPerOrder")
	s.setTradebinParam(t, "maxFillsPerOrder", `"`+strconv.Itoa(obDeepMaxFills)+`"`)
	defer s.setTradebinParam(t, "maxFillsPerOrder", string(maxFills))

	makers := interchaintest.GetAndFundTestUsers(t, s.ctx, "ob-deep", sdkmath.NewInt(obDeepFunds), slices.Repeat([]ibc.Chain{s.bze}, obDeepMakers)...)
	require.NoError(t, testutil.WaitForBlocks(s.ctx, 2, s.bze))
	ids := make([]string, len(makers))
	for i, m := range makers {
		s.mustExec(t, s.admin, "bank", "send", s.admin.FormattedAddress(), m.FormattedAddress(), coinStr(obDeepAmount, s.ob))
		res, id := s.place(t, m, "sell", obDeepAmount, obDeepPrice)
		ids[i] = s.requireSaved(t, res, id, m, "sell", strconv.Itoa(obDeepAmount), obDeepPrice)
	}
	s.requireBook(t, obBook{Buy: map[string]string{"1": "100"}, Sell: map[string]string{obDeepPrice: strconv.Itoa(obDeepMakers * obDeepAmount), "6": "100"}})
	execs := func(from, to int) []obExec {
		var want []obExec
		for i := from; i < to; i++ {
			want = append(want, obExec{ids[i], makers[i], "sell", strconv.Itoa(obDeepAmount), obDeepPrice})
		}
		return want
	}
	makersPaid := func(res helpers.TxResult, from, to int) {
		for i, m := range makers {
			want := sdkmath.ZeroInt()
			if i >= from && i < to {
				want = mulPrice(obDeepAmount, obDeepPrice)
			}
			require.Equal(t, want, s.transferred(t, res.Events, s.tradebin, m.FormattedAddress(), chain.Denom), "maker %d paid", i)
		}
	}
	filled := int64(obDeepMaxFills * obDeepAmount)

	// the whole level in one order: obDeepMaxFills fills, the rest refunded
	want := s.snapshot(t)
	whole := int64(obDeepMakers * obDeepAmount)
	res1 := s.mustExecMsgs(t, s.taker, obFillGas, s.orderMsg(s.taker, "buy", whole, obDeepPrice))
	t.Logf("taker gas for %d fills: %d", obDeepMaxFills, res1.GasUsed)
	require.Less(t, res1.GasUsed, int64(obDeepTakerGas), "taker gas is bounded by maxFillsPerOrder")
	id1 := s.fillIDs(t, res1, 1)[0]
	s.requireExecuted(t, res1, id1, s.taker, execs(0, obDeepMaxFills)...)
	rest := whole - filled
	s.requireRefunded(t, res1, id1, s.taker, "buy", strconv.FormatInt(rest, 10), mulPrice(rest, obDeepPrice).String()+chain.Denom, "max_fills_reached")
	s.requireNotSaved(t, res1, id1)
	makersPaid(res1, 0, obDeepMaxFills)
	want.paid(t, s, s.taker, res1, s.takerFee())
	want.move(s.taker, chain.Denom, -mulPrice(filled, obDeepPrice).Int64())
	want.move(s.taker, s.ob, filled)
	s.requireBalances(t, want)
	s.requireBook(t, obBook{Buy: map[string]string{"1": "100"}, Sell: map[string]string{obDeepPrice: strconv.FormatInt(rest, 10), "6": "100"}})

	// the next order fills the remaining orders and empties the level
	res2 := s.mustExecMsgs(t, s.taker, obFillGas, s.orderMsg(s.taker, "buy", rest, obDeepPrice))
	t.Logf("taker gas for the next %d fills: %d", obDeepMakers-obDeepMaxFills, res2.GasUsed)
	require.Less(t, res2.GasUsed, int64(obDeepTakerGas), "taker gas is bounded by maxFillsPerOrder")
	id2 := s.fillIDs(t, res2, 1)[0]
	s.requireExecuted(t, res2, id2, s.taker, execs(obDeepMaxFills, obDeepMakers)...)
	require.Empty(t, s.engineEvents(t, res2, "OrderMessageRefundedEvent", map[string]string{"order_id": id2}))
	s.requireNotSaved(t, res2, id2)
	makersPaid(res2, obDeepMaxFills, obDeepMakers)
	want.paid(t, s, s.taker, res2, s.takerFee())
	want.move(s.taker, chain.Denom, -mulPrice(rest, obDeepPrice).Int64())
	want.move(s.taker, s.ob, rest)
	s.requireBalances(t, want)
	s.requireBook(t, obBook{Buy: map[string]string{"1": "100"}, Sell: map[string]string{"6": "100"}})

	// the blocks carrying the takers and the blocks after them come at the usual pace
	require.NoError(t, testutil.WaitForBlocks(s.ctx, 1, s.bze))
	for _, h := range []int64{res1.Height, res1.Height + 1, res2.Height, res2.Height + 1} {
		gap := s.blockTime(t, h).Sub(s.blockTime(t, h-1))
		require.Less(t, gap, obMaxBlockTime, "block %d came %s after the previous one", h, gap)
	}
}

// 17. maker1 and maker2 rest a sell each at one price, then maker1 adds to
// its sell at that price (BZE-267): maker1's order grows under the new id of
// the top-up and goes behind maker2's, the old id is gone (a cancel by it is
// refused as not found) and the tx emits an OrderAmendedEvent with the old id
// as previous_id instead of saving a new order. A taker for 18 then fills
// maker2's whole sell first and 8 of maker1's grown one.
func (s *orderBookSuite) topUp(t *testing.T) {
	const price = "5.5"
	want, history := s.snapshot(t), s.history(t)
	res1, first := s.place(t, s.maker1, "sell", 10, price)
	s.requireSaved(t, res1, first, s.maker1, "sell", "10", price)
	res2, second := s.place(t, s.maker2, "sell", 10, price)
	s.requireSaved(t, res2, second, s.maker2, "sell", "10", price)
	res3, topUp := s.place(t, s.maker1, "sell", 6, price)
	s.requireFees(t, res3, s.maker1, s.makerFee(), sdk.NewInt64Coin(s.ob, 6))
	require.NotEqual(t, first, topUp, "the top-up gets a new id")
	s.requireAmended(t, res3, topUp, first, s.maker1, "sell", price, "6", "16")
	require.Empty(t, s.engineEvents(t, res3, "OrderSavedEvent", nil), "the top-up saves no order")
	for _, r := range []struct {
		u   ibc.Wallet
		res helpers.TxResult
		ob  int64
	}{{s.maker1, res1, 10}, {s.maker2, res2, 10}, {s.maker1, res3, 6}} {
		want.paid(t, s, r.u, r.res, s.makerFee())
		want.move(r.u, s.ob, -r.ob)
	}
	s.requireBalances(t, want)
	s.requireBook(t, obBook{Buy: map[string]string{"1": "100"}, Sell: map[string]string{price: "26", "6": "100"}})
	maker1Orders := s.userOrders(t, s.maker1)
	require.Contains(t, maker1Orders, obOrder{ID: topUp, OrderType: "sell", Amount: "16", Price: price, Owner: s.maker1.FormattedAddress()})
	for _, o := range maker1Orders {
		require.NotEqual(t, first, o.ID, "maker1's old id is gone")
	}

	// a cancel by the old id is refused and changes nothing
	book := s.book(t)
	res, err := helpers.ExecTxWith(s.ctx, s.bze, s.maker1, []helpers.TxOption{helpers.WithGas(obTxGas)}, "tradebin", "cancel-order", s.market, first, "sell")
	s.requireRefused(t, res, err, 4008, "order not found")
	want.paid(t, s, s.maker1, res, sdkmath.ZeroInt())
	s.requireBalances(t, want)
	require.Equal(t, book, s.book(t), "order book changed")

	res, orderID := s.place(t, s.taker, "buy", 18, price)
	s.requireFees(t, res, s.taker, s.takerFee(), sdk.NewCoin(chain.Denom, mulPrice(18, price)))
	want.paid(t, s, s.taker, res, s.takerFee())
	s.requireExecuted(t, res, orderID, s.taker,
		obExec{second, s.maker2, "sell", "10", price},
		obExec{topUp, s.maker1, "sell", "8", price},
	)
	s.requireNotSaved(t, res, orderID)
	want.move(s.taker, chain.Denom, -mulPrice(18, price).Int64())
	want.move(s.taker, s.ob, 18)
	want.move(s.maker2, chain.Denom, mulPrice(10, price).Int64())
	want.move(s.maker1, chain.Denom, mulPrice(8, price).Int64())

	s.requireBalances(t, want)
	s.requireBook(t, obBook{Buy: map[string]string{"1": "100"}, Sell: map[string]string{price: "8", "6": "100"}})
	_, found := s.marketOrder(t, "sell", second)
	require.False(t, found, "maker2's sell, ahead of the top-up, is filled whole")
	grown, found := s.marketOrder(t, "sell", topUp)
	require.True(t, found, "maker1's grown sell still rests")
	require.Equal(t, "8", grown.Amount)
	s.requireNewHistory(t, history,
		obHistory{"sell", "10", price, s.maker2.FormattedAddress(), s.taker.FormattedAddress()},
		obHistory{"sell", "8", price, s.maker1.FormattedAddress(), s.taker.FormattedAddress()},
	)
}

// 18. With priceLevelOwnersWindow 2 and priceLevelExtraGas 1,000,000, five
// distinct makers join one price level. The first three pay no extra gas;
// the fourth joins a level of three owners and pays (3 - 2) × 1,000,000 more
// gas than the third. The first maker tops up its order without the extra
// gas. The fifth maker would pay 2,000,000 extra and sends its tx with less
// gas: the tx runs out of gas and leaves no order and no escrow. The
// aggregated level counts its owners. The orders are cancelled at the end,
// the level goes away and the params are restored.
func (s *orderBookSuite) crowdedLevel(t *testing.T) {
	window, extraGas := s.tradebinParam(t, "priceLevelOwnersWindow"), s.tradebinParam(t, "priceLevelExtraGas")
	s.setTradebinParam(t, "priceLevelOwnersWindow", `"`+strconv.Itoa(obCrowdWindow)+`"`)
	defer s.setTradebinParam(t, "priceLevelOwnersWindow", string(window))
	s.setTradebinParam(t, "priceLevelExtraGas", `"`+strconv.Itoa(obCrowdExtraGas)+`"`)
	defer s.setTradebinParam(t, "priceLevelExtraGas", string(extraGas))

	bookBefore := s.book(t)
	makers := interchaintest.GetAndFundTestUsers(t, s.ctx, "ob-crowd", sdkmath.NewInt(obDeepFunds), slices.Repeat([]ibc.Chain{s.bze}, obCrowdMakers)...)
	require.NoError(t, testutil.WaitForBlocks(s.ctx, 2, s.bze))
	for _, m := range makers {
		s.mustExec(t, s.admin, "bank", "send", s.admin.FormattedAddress(), m.FormattedAddress(), coinStr(2*obCrowdAmount, s.ob))
	}

	ids := make([]string, obCrowdMakers-1)
	gas := make([]int64, obCrowdMakers-1)
	for i, m := range makers[:obCrowdMakers-1] {
		res := s.mustExecMsgs(t, m, obCrowdTxGas, s.orderMsg(m, "sell", obCrowdAmount, obCrowdPrice))
		ids[i] = s.requireSaved(t, res, s.fillIDs(t, res, 1)[0], m, "sell", strconv.Itoa(obCrowdAmount), obCrowdPrice)
		gas[i] = res.GasUsed
		t.Logf("maker %d joins a level of %d owners: %d gas", i+1, i, res.GasUsed)
		require.Equal(t, uint64(i+1), s.levelOwners(t, "sell", obCrowdPrice), "owners after maker %d", i+1)
	}
	// up to the window owners at the level, a new owner pays no extra gas
	require.InDelta(t, gas[0], gas[1], obCrowdGasSlack)
	require.InDelta(t, gas[1], gas[2], obCrowdGasSlack)
	// the fourth joins three owners: one owner past the window
	require.InDelta(t, gas[2]+obCrowdExtraGas, gas[3], obCrowdGasSlack, "the fourth owner pays the extra gas")

	// the first maker adds to its order: no new owner, no extra gas; the
	// merged order rests under the top-up's new id
	res := s.mustExecMsgs(t, makers[0], obCrowdTxGas, s.orderMsg(makers[0], "sell", obCrowdAmount, obCrowdPrice))
	topUp := s.fillIDs(t, res, 1)[0]
	s.requireAmended(t, res, topUp, ids[0], makers[0], "sell", obCrowdPrice, strconv.Itoa(obCrowdAmount), strconv.Itoa(2*obCrowdAmount))
	ids[0] = topUp
	t.Logf("maker 1 tops up at a level of 4 owners: %d gas", res.GasUsed)
	require.Less(t, res.GasUsed, gas[2]+obCrowdGasSlack, "a top-up pays no extra gas")
	require.Equal(t, uint64(obCrowdMakers-1), s.levelOwners(t, "sell", obCrowdPrice))

	// the fifth would pay 2,000,000 extra gas: with less gas its tx fails and
	// leaves nothing behind
	last := makers[obCrowdMakers-1]
	book := s.book(t)
	res, err := helpers.ExecMsgs(s.ctx, s.bze, last, []helpers.TxOption{helpers.WithGas(obCrowdShortGas)}, s.orderMsg(last, "sell", obCrowdAmount, obCrowdPrice))
	require.NoError(t, err)
	require.Equal(t, "sdk", res.Codespace, res.RawLog)
	require.EqualValues(t, 11, res.Code, res.RawLog)
	require.Contains(t, res.RawLog, "out of gas")
	s.requireOrders(t, last)
	require.Equal(t, sdkmath.NewInt(2*obCrowdAmount), s.balance(t, last.FormattedAddress(), s.ob), "nothing escrowed")
	require.Equal(t, book, s.book(t), "order book changed")
	require.Equal(t, uint64(obCrowdMakers-1), s.levelOwners(t, "sell", obCrowdPrice))

	// every owner leaving takes one off the count, the last one the level
	for i, m := range makers[:obCrowdMakers-1] {
		s.mustExecMsgs(t, m, obTxGas, s.cancelMsg(m, "sell", ids[i]))
		if i < obCrowdMakers-2 {
			require.Equal(t, uint64(obCrowdMakers-2-i), s.levelOwners(t, "sell", obCrowdPrice))
		}
	}
	require.Equal(t, bookBefore, s.book(t))
}

// 19. The tradebin module holds, on top of what it held before the market
// existed, exactly the ob of the resting sells and the ubze of the resting
// buys (amount × price) plus the dust it stores for users. The aggregated
// levels match the users' orders.
func (s *orderBookSuite) conservation(t *testing.T) {
	book := s.book(t)
	levels := obBook{Buy: map[string]string{}, Sell: map[string]string{}}
	for _, u := range s.users() {
		for _, o := range s.userOrders(t, u) {
			side := levels.Sell
			if o.OrderType == "buy" {
				side = levels.Buy
			}
			side[o.Price] = parseInt(side[o.Price]).Add(parseInt(o.Amount)).String()
		}
	}
	for _, side := range []*map[string]string{&levels.Buy, &levels.Sell} {
		if len(*side) == 0 {
			*side = nil
		}
	}
	require.Equal(t, book, levels, "aggregated levels = sum of the users' orders")

	quote := sdkmath.LegacyZeroDec()
	for price, amount := range book.Buy {
		quote = quote.Add(sdkmath.LegacyMustNewDecFromStr(price).MulInt(parseInt(amount)))
	}
	base := sdkmath.ZeroInt()
	for _, amount := range book.Sell {
		base = base.Add(parseInt(amount))
	}
	for _, u := range s.users() {
		for denom, amount := range s.userDust(t, u) {
			require.Equal(t, chain.Denom, denom, "dust only arises in the quote")
			quote = quote.Add(sdkmath.LegacyMustNewDecFromStr(amount))
		}
	}
	require.True(t, quote.IsInteger(), "resting buys plus dust are whole ubze: %s", quote)

	held := s.moduleBalances(t)
	require.Equal(t, quote.TruncateInt(), held.AmountOf(chain.Denom).Sub(s.baseline.AmountOf(chain.Denom)), "ubze held for buys and dust")
	require.Equal(t, base, held.AmountOf(s.ob).Sub(s.baseline.AmountOf(s.ob)), "ob held for sells")
}

// --- order book helpers ---

// tradebinParam returns one tradebin param as the query prints it.
func (s *orderBookSuite) tradebinParam(t *testing.T, name string) json.RawMessage {
	t.Helper()
	res, err := helpers.Query[struct {
		Params map[string]json.RawMessage `json:"params"`
	}](s.ctx, s.bze, "tradebin", "params")
	require.NoError(t, err)
	require.Contains(t, res.Params, name, "tradebin param %s", name)
	return res.Params[name]
}

// setTradebinParam passes a tradebin MsgUpdateParams that changes one param
// (value is its JSON).
func (s *orderBookSuite) setTradebinParam(t *testing.T, name, value string) {
	t.Helper()
	res, err := helpers.Query[struct {
		Params map[string]json.RawMessage `json:"params"`
	}](s.ctx, s.bze, "tradebin", "params")
	require.NoError(t, err)
	params := res.Params
	requireDecimalDecParams(t, params)
	require.NoError(t, helpers.DecParamsForTx(params, helpers.TradebinDecParams...))
	params[name] = json.RawMessage(value)
	_, err = helpers.SubmitAndPassProposal(s.ctx, s.bze, s.admin, helpers.Proposal{
		Messages: []any{map[string]any{"@type": "/bze.tradebin.MsgUpdateParams", "authority": s.gov, "params": params}},
		Title:    "Set tradebin " + name,
		Summary:  "e2e: tradebin " + name,
	})
	require.NoError(t, err)
	require.JSONEq(t, value, string(s.tradebinParam(t, name)), "tradebin %s", name)
}

// requireDecimalDecParams requires the queried tradebin Dec params (those the
// release has) to be printed as decimals: the params query output must be
// valid input for a MsgUpdateParams as it is.
func requireDecimalDecParams(t *testing.T, params map[string]json.RawMessage) {
	t.Helper()
	for _, name := range helpers.TradebinDecParams {
		raw, ok := params[name]
		if !ok {
			continue
		}
		var v string
		require.NoError(t, json.Unmarshal(raw, &v), "tradebin param %s", name)
		require.Contains(t, v, ".", "the tradebin params query must print %s as a decimal, got %s", name, raw)
	}
}

// blockTime reads the header time of the block at height.
func (s *orderBookSuite) blockTime(t *testing.T, height int64) time.Time {
	t.Helper()
	res, err := helpers.Query[struct {
		Header struct {
			Time time.Time `json:"time"`
		} `json:"header"`
		Block struct {
			Header struct {
				Time time.Time `json:"time"`
			} `json:"header"`
		} `json:"block"`
	}](s.ctx, s.bze, "block", "--type=height", strconv.FormatInt(height, 10))
	require.NoError(t, err)
	if !res.Header.Time.IsZero() {
		return res.Header.Time
	}
	require.False(t, res.Block.Header.Time.IsZero(), "no header time for block %d", height)
	return res.Block.Header.Time
}

type obBook struct {
	Buy, Sell map[string]string // price → amount, nil when the side is empty
}

type obOrder struct {
	ID        string `json:"id"`
	OrderType string `json:"order_type"`
	Amount    string `json:"amount"`
	Price     string `json:"price"`
	Owner     string `json:"owner"`
}

type obHistory struct {
	OrderType, Amount, Price, Maker, Taker string
}

// obExec is one OrderExecutedEvent: the resting order, its owner (maker),
// its type, the executed amount and its price.
type obExec struct {
	ID        string
	Maker     ibc.Wallet
	OrderType string
	Amount    string
	Price     string
}

type obItem struct {
	Price, Amount string
}

// obLedger holds the expected ubze and ob balances of the users.
type obLedger map[string]sdk.Coins

func (l obLedger) move(u ibc.Wallet, denom string, delta int64) {
	a := u.FormattedAddress()
	switch {
	case delta > 0:
		l[a] = l[a].Add(sdk.NewInt64Coin(denom, delta))
	case delta < 0:
		l[a] = l[a].Sub(sdk.NewInt64Coin(denom, -delta))
	}
}

// paid takes the tx fee (none when CheckTx refused the tx) and the trading
// fee from the user.
func (l obLedger) paid(t *testing.T, s *orderBookSuite, u ibc.Wallet, res helpers.TxResult, tradingFee sdkmath.Int) {
	t.Helper()
	fee := tradingFee
	if res.Height > 0 {
		fee = fee.Add(s.txFee(t, res))
	}
	l.move(u, chain.Denom, -fee.Int64())
}

func (s *orderBookSuite) snapshot(t *testing.T) obLedger {
	t.Helper()
	l := obLedger{}
	for _, u := range s.users() {
		l[u.FormattedAddress()] = s.balances(t, u, chain.Denom, s.ob)
	}
	return l
}

func (s *orderBookSuite) requireBalances(t *testing.T, want obLedger) {
	t.Helper()
	got := s.snapshot(t)
	for _, u := range s.users() {
		require.Equal(t, want[u.FormattedAddress()].String(), got[u.FormattedAddress()].String(), "balances of %s", u.KeyName())
	}
}

func (s *orderBookSuite) makerFee() sdkmath.Int { return s.params.MarketMakerFee.Amount }
func (s *orderBookSuite) takerFee() sdkmath.Int { return s.params.MarketTakerFee.Amount }

// requireFees asserts the tx moved the trading fee and the order funds from
// the user to the tradebin module and the fee straight on to the burner.
func (s *orderBookSuite) requireFees(t *testing.T, res helpers.TxResult, u ibc.Wallet, fee sdkmath.Int, locked sdk.Coin) {
	t.Helper()
	user := u.FormattedAddress()
	toModule := sdk.NewCoins(sdk.NewCoin(chain.Denom, fee)).Add(locked)
	require.Equal(t, toModule.AmountOf(chain.Denom), s.transferred(t, res.Events, user, s.tradebin, chain.Denom), "ubze to the tradebin module")
	require.Equal(t, toModule.AmountOf(s.ob), s.transferred(t, res.Events, user, s.tradebin, s.ob), "ob to the tradebin module")
	require.Equal(t, fee, s.transferred(t, res.Events, s.tradebin, s.burner, chain.Denom), "trading fee to the burner")
}

// place sends a create-order and returns the tx and the order id the engine
// gave the order (every order gets one, resting or not).
func (s *orderBookSuite) place(t *testing.T, u ibc.Wallet, typ string, amount int64, price string) (helpers.TxResult, string) {
	t.Helper()
	amt := strconv.FormatInt(amount, 10)
	res := s.mustExec(t, u, "tradebin", "create-order", typ, amt, price, s.market)
	return res, s.fillIDs(t, res, 1)[0]
}

// obOutcomeEvents are the engine events that carry the id of the order being
// executed in their order_id attribute.
var obOutcomeEvents = []string{
	tradebinEventPrefix + "OrderExecutedEvent",
	tradebinEventPrefix + "OrderSavedEvent",
	tradebinEventPrefix + "OrderMessageRefundedEvent",
}

// obAmendedEvent carries the id of the order being executed (the new id the
// merged order rests under) in its id attribute.
const obAmendedEvent = tradebinEventPrefix + "OrderAmendedEvent"

// fillIDs returns the order ids of the orders a tx executed (one per order,
// one per MsgFillOrders entry), in the order the engine processed them.
func (s *orderBookSuite) fillIDs(t *testing.T, res helpers.TxResult, want int) []string {
	t.Helper()
	var ids []string
	seen := map[string]bool{}
	for _, e := range res.Events {
		key := "order_id"
		if e.Type == obAmendedEvent {
			key = "id"
		} else if !slices.Contains(obOutcomeEvents, e.Type) {
			continue
		}
		id := e.Attributes[key]
		require.NotEmpty(t, id, "%s without order_id", e.Type)
		if !seen[id] {
			seen[id] = true
			ids = append(ids, id)
		}
	}
	require.Len(t, ids, want, "orders executed by tx %s", res.TxHash)
	return ids
}

func (s *orderBookSuite) engineEvents(t *testing.T, res helpers.TxResult, name string, attrs map[string]string) []helpers.Event {
	t.Helper()
	return helpers.EventsWith(helpers.TxEvents(res, tradebinEventPrefix+name), attrs)
}

// requireSaved asserts the tx saved the order as one resting order under its
// own id and returns it.
func (s *orderBookSuite) requireSaved(t *testing.T, res helpers.TxResult, orderID string, owner ibc.Wallet, typ, amount, price string) string {
	t.Helper()
	saved := s.engineEvents(t, res, "OrderSavedEvent", map[string]string{"order_id": orderID})
	require.Len(t, saved, 1, "order %s saved once in tx %s", orderID, res.TxHash)
	e := saved[0]
	require.Equal(t, map[string]string{"id": orderID, "market_id": s.market, "owner": owner.FormattedAddress(), "order_type": typ, "amount": amount, "price": price},
		map[string]string{"id": e.Attributes["id"], "market_id": e.Attributes["market_id"], "owner": e.Attributes["owner"], "order_type": e.Attributes["order_type"], "amount": e.Attributes["amount"], "price": e.Attributes["price"]})
	order, found := s.marketOrder(t, typ, orderID)
	require.True(t, found, "saved order %s is on the book", orderID)
	require.Equal(t, amount, order.Amount)
	return orderID
}

// requireAmended asserts the tx merged amount into the owner's resting order
// previousID under the new id (one OrderAmendedEvent, total the order's amount
// after it), that the order rests under id with that amount and that
// previousID is gone.
func (s *orderBookSuite) requireAmended(t *testing.T, res helpers.TxResult, id, previousID string, owner ibc.Wallet, typ, price, added, total string) {
	t.Helper()
	got := s.engineEvents(t, res, "OrderAmendedEvent", map[string]string{"id": id})
	require.Len(t, got, 1, "order %s amended once in tx %s", id, res.TxHash)
	e := got[0].Attributes
	require.Equal(t,
		[]string{s.market, typ, price, owner.FormattedAddress(), added, total, previousID},
		[]string{e["market_id"], e["order_type"], e["price"], e["owner"], e["added_amount"], e["amount"], e["previous_id"]},
		"amendment of order %s", id)
	order, found := s.marketOrder(t, typ, id)
	require.True(t, found, "amended order %s is on the book", id)
	require.Equal(t, total, order.Amount)
	_, found = s.marketOrder(t, typ, previousID)
	require.False(t, found, "order %s replaced by %s is gone", previousID, id)
}

// requireRefunded asserts the one OrderMessageRefundedEvent of an order: the
// unexecuted base amount, the coin sent back and the reason.
func (s *orderBookSuite) requireRefunded(t *testing.T, res helpers.TxResult, orderID string, owner ibc.Wallet, typ, amount, refund, reason string) {
	t.Helper()
	got := s.engineEvents(t, res, "OrderMessageRefundedEvent", map[string]string{"order_id": orderID})
	require.Len(t, got, 1, "refunds of order %s in tx %s", orderID, res.TxHash)
	e := got[0].Attributes
	require.Equal(t,
		[]string{s.market, typ, owner.FormattedAddress(), amount, refund, reason},
		[]string{e["market_id"], e["order_type"], e["owner"], e["amount"], e["refund"], e["reason"]},
		"refund of order %s", orderID)
}

func (s *orderBookSuite) requireNotSaved(t *testing.T, res helpers.TxResult, orderID string) {
	t.Helper()
	require.Empty(t, s.engineEvents(t, res, "OrderSavedEvent", map[string]string{"order_id": orderID}), "order %s saved", orderID)
	_, found := s.marketOrder(t, "buy", orderID)
	require.False(t, found, "order %s rests as a buy", orderID)
	_, found = s.marketOrder(t, "sell", orderID)
	require.False(t, found, "order %s rests as a sell", orderID)
}

// requireExecuted asserts the OrderExecutedEvents of one taker order, in the
// order the engine filled them.
func (s *orderBookSuite) requireExecuted(t *testing.T, res helpers.TxResult, orderID string, taker ibc.Wallet, want ...obExec) {
	t.Helper()
	got := s.engineEvents(t, res, "OrderExecutedEvent", map[string]string{"order_id": orderID})
	require.Len(t, got, len(want), "executions of order %s: %+v", orderID, got)
	for i, w := range want {
		e := got[i].Attributes
		require.Equal(t,
			[]string{w.ID, s.market, w.OrderType, w.Amount, w.Price, w.Maker.FormattedAddress(), taker.FormattedAddress()},
			[]string{e["id"], e["market_id"], e["order_type"], e["amount"], e["price"], e["maker"], e["taker"]},
			"execution %d of order %s", i, orderID)
	}
}

// requireCanceled asserts the tx's OrderCanceledEvent of the order (its own
// id in order_id) and that the order left the book.
func (s *orderBookSuite) requireCanceled(t *testing.T, res helpers.TxResult, owner ibc.Wallet, id, typ, amount, price string) {
	t.Helper()
	canceled := s.engineEvents(t, res, "OrderCanceledEvent", map[string]string{"id": id})
	require.Len(t, canceled, 1)
	e := canceled[0].Attributes
	require.Equal(t, []string{id, owner.FormattedAddress(), typ, amount, price}, []string{e["order_id"], e["owner"], e["order_type"], e["amount"], e["price"]})
	_, found := s.marketOrder(t, typ, id)
	require.False(t, found, "cancelled order %s left the book", id)
}

// requireRefused asserts a refusal with the tradebin code (or, for code 5,
// the sdk's insufficient funds; 0 skips the code) and the chain's message.
func (s *orderBookSuite) requireRefused(t *testing.T, res helpers.TxResult, err error, code uint32, reason string) {
	t.Helper()
	got, err := helpers.RejectionReason(res, err)
	require.NoError(t, err)
	require.Contains(t, got, reason)
	if code == 0 {
		return
	}
	require.EqualValues(t, code, res.Code, got)
	if code != 5 {
		require.Equal(t, obCodespace, res.Codespace, got)
	}
}

func (s *orderBookSuite) mustExecMsgs(t *testing.T, u ibc.Wallet, gas uint64, msgs ...any) helpers.TxResult {
	t.Helper()
	res, err := helpers.ExecMsgs(s.ctx, s.bze, u, []helpers.TxOption{helpers.WithGas(gas)}, msgs...)
	require.NoError(t, err)
	require.Zero(t, res.Code, res.RawLog)
	return res
}

func (s *orderBookSuite) orderMsg(u ibc.Wallet, typ string, amount int64, price string) map[string]any {
	return map[string]any{
		"@type":      "/bze.tradebin.MsgCreateOrder",
		"creator":    u.FormattedAddress(),
		"order_type": typ,
		"amount":     strconv.FormatInt(amount, 10),
		"price":      price,
		"market_id":  s.market,
	}
}

func (s *orderBookSuite) cancelMsg(u ibc.Wallet, typ, id string) map[string]any {
	return map[string]any{
		"@type":      "/bze.tradebin.MsgCancelOrder",
		"creator":    u.FormattedAddress(),
		"market_id":  s.market,
		"order_id":   id,
		"order_type": typ,
	}
}

func (s *orderBookSuite) fillMsg(u ibc.Wallet, typ string, items ...obItem) map[string]any {
	orders := make([]map[string]string, 0, len(items))
	for _, it := range items {
		orders = append(orders, map[string]string{"price": it.Price, "amount": it.Amount})
	}
	return map[string]any{
		"@type":      "/bze.tradebin.MsgFillOrders",
		"creator":    u.FormattedAddress(),
		"market_id":  s.market,
		"order_type": typ,
		"orders":     orders,
	}
}

// batch is one 10 ob sell order per price.
func (s *orderBookSuite) aggregates(t *testing.T, typ string) map[string]string {
	t.Helper()
	res, err := helpers.Query[struct {
		List []struct {
			Amount string `json:"amount"`
			Price  string `json:"price"`
		} `json:"list"`
	}](s.ctx, s.bze, "tradebin", "market-aggregated-orders", s.market, typ)
	require.NoError(t, err)
	if len(res.List) == 0 {
		return nil
	}
	out := make(map[string]string, len(res.List))
	for _, l := range res.List {
		out[l.Price] = l.Amount
	}
	return out
}

// levelOwners returns the distinct owners MarketAggregatedOrders reports at
// one price of a side.
func (s *orderBookSuite) levelOwners(t *testing.T, typ, price string) uint64 {
	t.Helper()
	res, err := helpers.Query[struct {
		List []struct {
			Price  string        `json:"price"`
			Owners helpers.Int64 `json:"owners"`
		} `json:"list"`
	}](s.ctx, s.bze, "tradebin", "market-aggregated-orders", s.market, typ)
	require.NoError(t, err)
	for _, l := range res.List {
		if l.Price == price {
			return uint64(l.Owners)
		}
	}
	require.Failf(t, "price level not found", "%s %s", typ, price)
	return 0
}

func (s *orderBookSuite) book(t *testing.T) obBook {
	t.Helper()
	return obBook{Buy: s.aggregates(t, "buy"), Sell: s.aggregates(t, "sell")}
}

func (s *orderBookSuite) requireBook(t *testing.T, want obBook) {
	t.Helper()
	require.Equal(t, want, s.book(t), "aggregated order book")
}

func (s *orderBookSuite) marketOrder(t *testing.T, typ, id string) (obOrder, bool) {
	t.Helper()
	res, err := helpers.Query[struct {
		Order obOrder `json:"order"`
	}](s.ctx, s.bze, "tradebin", "market-order", s.market, typ, id)
	if err != nil {
		require.Contains(t, err.Error(), "not found")
		return obOrder{}, false
	}
	return res.Order, true
}

func (s *orderBookSuite) userOrders(t *testing.T, u ibc.Wallet) []obOrder {
	t.Helper()
	res, err := helpers.Query[struct {
		List []obOrder `json:"list"`
	}](s.ctx, s.bze, "tradebin", "user-market-orders", u.FormattedAddress(), s.market)
	require.NoError(t, err)
	out := make([]obOrder, 0, len(res.List))
	for _, ref := range res.List {
		order, found := s.marketOrder(t, ref.OrderType, ref.ID)
		require.True(t, found, "user order %s is on the book", ref.ID)
		out = append(out, order)
	}
	return out
}

// requireOrders asserts the user's resting orders (ids are zero-filled, so
// the query lists them in placement order).
func (s *orderBookSuite) requireOrders(t *testing.T, u ibc.Wallet, want ...obOrder) {
	t.Helper()
	got := s.userOrders(t, u)
	if len(want) == 0 {
		require.Empty(t, got, "orders of %s", u.KeyName())
		return
	}
	require.Equal(t, want, got, "orders of %s", u.KeyName())
}

// userDust returns denom → stored dust, normalised to 18 decimals.
func (s *orderBookSuite) userDust(t *testing.T, u ibc.Wallet) map[string]string {
	t.Helper()
	res, err := helpers.Query[struct {
		List []struct {
			Amount string `json:"amount"`
			Denom  string `json:"denom"`
		} `json:"list"`
	}](s.ctx, s.bze, "tradebin", "all-user-dust", u.FormattedAddress())
	require.NoError(t, err)
	out := map[string]string{}
	for _, d := range res.List {
		out[d.Denom] = sdkmath.LegacyMustNewDecFromStr(d.Amount).String()
	}
	return out
}

// obDust is the expected ubze dust map; "" means no dust stored.
func obDust(amount string) map[string]string {
	if amount == "" {
		return map[string]string{}
	}
	return map[string]string{chain.Denom: sdkmath.LegacyMustNewDecFromStr(amount).String()}
}

func (s *orderBookSuite) history(t *testing.T) []obHistory {
	t.Helper()
	res, err := helpers.Query[struct {
		List []struct {
			OrderType string `json:"order_type"`
			Amount    string `json:"amount"`
			Price     string `json:"price"`
			Maker     string `json:"maker"`
			Taker     string `json:"taker"`
		} `json:"list"`
	}](s.ctx, s.bze, "tradebin", "market-history", s.market)
	require.NoError(t, err)
	out := make([]obHistory, 0, len(res.List))
	for _, h := range res.List {
		out = append(out, obHistory{h.OrderType, h.Amount, h.Price, h.Maker, h.Taker})
	}
	return out
}

// requireNewHistory asserts the records added since before, as a multiset.
func (s *orderBookSuite) requireNewHistory(t *testing.T, before []obHistory, want ...obHistory) {
	t.Helper()
	added := s.history(t)
	for _, b := range before {
		for i, a := range added {
			if a == b {
				added = append(added[:i], added[i+1:]...)
				break
			}
		}
	}
	require.ElementsMatch(t, want, added, "new market history records")
}

func (s *orderBookSuite) moduleBalances(t *testing.T) sdk.Coins {
	t.Helper()
	out := sdk.NewCoins()
	for _, d := range []string{chain.Denom, s.ob} {
		out = out.Add(sdk.NewCoin(d, s.balance(t, s.tradebin, d)))
	}
	return out
}

// mulPrice is amount × price for a price that makes it whole.
func mulPrice(amount int64, price string) sdkmath.Int {
	v := sdkmath.LegacyMustNewDecFromStr(price).MulInt64(amount)
	if !v.IsInteger() {
		panic("mulPrice: " + v.String() + " is not whole")
	}
	return v.TruncateInt()
}
