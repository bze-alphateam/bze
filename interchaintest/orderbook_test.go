package interchaintest_test

import (
	"context"
	"encoding/json"
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

// The order book matches a message only against resting orders at exactly its
// price, in the EndBlock of the block that included the tx. Amounts are in ob
// (the base), prices in ubze per ob. Every price below has a minimum order
// amount of 2 (2 × ⌈1/price⌉) except 2.35's dust step and the 0.333 refusal.
const (
	obAdminFunds = 100_000_000_000 // 100,000 BZE: the denom and market create fees, proposals
	obUserFunds  = 10_000_000_000  // 10,000 BZE each
	obMint       = 100_000_000
	obUserBase   = 10_000_000 // ob sent to each maker and the taker

	obTxGas    = 500_000
	obFillGas  = 1_000_000
	obBatchGas = 2_000_000 // a tx carrying five orders

	// the step 16 per-block cap and the step 17 queue window
	obPerBlockCap = 3
	obQueueWindow = 2
	obBatchOrders = 5

	obCodespace = "tradebin"
)

// TestOrderBook runs the tradebin order-book engine end to end on one market,
// ob/ubze: resting orders, partial fills, FIFO at one price, remainders saved
// or refunded, the minimum kept on a resting order, dust at a fractional
// price, MsgFillOrders on buy orders with its refunds, cancels and their
// refusals, a fill and a cancel of the same order in one block, crossed-book
// and placement refusals, a self-trade, the per-block message cap and the
// queue spam surcharge (both set through governance). Every step asserts the
// exact balances of both makers and the taker, the trading fees on their way
// to the burner, the engine events and the order book; the suite ends with
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
	t.Run("12 fill before a queued cancel", s.fillBeforeCancel)
	t.Run("13 crossed book refusals", s.crossedBook)
	t.Run("14 placement refusals", s.placementRefusals)
	t.Run("15 self-trade", s.selfTrade)
	t.Run("16 per-block message cap", s.perBlockCap)
	t.Run("17 queue spam surcharge", s.queueSurcharge)
	t.Run("18 module holds the book and the dust", s.conservation)
}

type orderBookSuite struct {
	// the AMM suite's query and tx helpers (balances, events, params)
	*ammSuite

	admin, maker1, maker2, taker ibc.Wallet
	ob                           string
	gov                          string
	baseline                     sdk.Coins // tradebin module balances once the market exists

	maker1Buy  string                     // step 1: maker1's buy at 2, partially filled until step 10
	filledSell string                     // step 3: maker1's sell at 3, filled whole
	maker2Sell string                     // step 6: maker2's sell at 4 that keeps the minimum
	defaults   map[string]json.RawMessage // tradebin params before the suite changed any
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
		"tradebin":              &s.tradebin,
		"burner":                &s.burner,
		"txfeecollector_burner": &s.burnerFees,
		"txfeecollector_cp":     &s.cpFees,
	} {
		*addr, err = helpers.ModuleAddress(ctx, bze, name)
		require.NoError(t, err)
	}
	s.params = s.queryParams(t)
	s.defaults = s.rawParams(t)

	s.ob = "factory/" + s.admin.FormattedAddress() + "/ob"
	s.market = s.ob + "/" + chain.Denom
	s.mustExec(t, s.admin, "tokenfactory", "create-denom", "ob")
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
	res, msgID := s.place(t, s.maker1, "buy", 1000, "2")
	s.requireFees(t, res, s.maker1, s.makerFee(), sdk.NewInt64Coin(chain.Denom, 2000))
	want.paid(t, s, s.maker1, res, s.makerFee())
	want.move(s.maker1, chain.Denom, -2000)

	s.maker1Buy = s.requireSaved(t, res.Height, msgID, s.maker1, "buy", "1000", "2")
	s.requireBalances(t, want)
	s.requireBook(t, obBook{Buy: map[string]string{"2": "1000"}})
	s.requireOrders(t, s.maker1, obOrder{ID: s.maker1Buy, OrderType: "buy", Amount: "1000", Price: "2", Owner: s.maker1.FormattedAddress()})
}

// 2. The taker sells less than the resting buy: the order stays with the
// rest, the maker gets the ob, the taker the ubze, and a history record is
// written. Selling at a price with a resting buy is taking: the taker fee.
func (s *orderBookSuite) partialFill(t *testing.T) {
	want, history := s.snapshot(t), s.history(t)
	res, msgID := s.place(t, s.taker, "sell", 400, "2")
	s.requireFees(t, res, s.taker, s.takerFee(), sdk.NewInt64Coin(s.ob, 400))
	want.paid(t, s, s.taker, res, s.takerFee())

	s.requireExecuted(t, res.Height, msgID, s.taker, obExec{s.maker1Buy, s.maker1, "buy", "400", "2"})
	s.requireNotSaved(t, res.Height, msgID)
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
	res1, msg1 := s.place(t, s.maker1, "sell", 100, "3")
	s.requireFees(t, res1, s.maker1, s.makerFee(), sdk.NewInt64Coin(s.ob, 100))
	s.filledSell = s.requireSaved(t, res1.Height, msg1, s.maker1, "sell", "100", "3")
	res2, msg2 := s.place(t, s.maker2, "sell", 100, "3")
	s.requireFees(t, res2, s.maker2, s.makerFee(), sdk.NewInt64Coin(s.ob, 100))
	maker2Sell := s.requireSaved(t, res2.Height, msg2, s.maker2, "sell", "100", "3")
	want.paid(t, s, s.maker1, res1, s.makerFee())
	want.paid(t, s, s.maker2, res2, s.makerFee())
	want.move(s.maker1, s.ob, -100)
	want.move(s.maker2, s.ob, -100)

	res, msgID := s.place(t, s.taker, "buy", 150, "3")
	s.requireFees(t, res, s.taker, s.takerFee(), sdk.NewInt64Coin(chain.Denom, 450))
	want.paid(t, s, s.taker, res, s.takerFee())
	s.requireExecuted(t, res.Height, msgID, s.taker,
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
	res, msgID := s.place(t, s.taker, "buy", 200, "3")
	s.requireFees(t, res, s.taker, s.takerFee(), sdk.NewInt64Coin(chain.Denom, 600))
	want.paid(t, s, s.taker, res, s.takerFee())

	s.requireExecuted(t, res.Height, msgID, s.taker, obExec{maker2Sell, s.maker2, "sell", "50", "3"})
	saved := s.requireSaved(t, res.Height, msgID, s.taker, "buy", "150", "3")
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
	res, msgID := s.place(t, s.maker1, "sell", 151, "3")
	s.requireFees(t, res, s.maker1, s.takerFee(), sdk.NewInt64Coin(s.ob, 151))
	want.paid(t, s, s.maker1, res, s.takerFee())

	s.requireExecuted(t, res.Height, msgID, s.maker1, obExec{takerBuy, s.taker, "buy", "150", "3"})
	s.requireNotSaved(t, res.Height, msgID)
	require.Equal(t, sdkmath.OneInt(), s.transferred(t, s.blockEvents(t, res.Height), s.tradebin, s.maker1.FormattedAddress(), s.ob), "1 ob refunded")
	s.requireRefunded(t, res.Height, msgID, s.maker1, "sell", "1", "1"+s.ob, "below_min_amount")
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
	res1, msg1 := s.place(t, s.maker2, "sell", 600, "4")
	s.requireFees(t, res1, s.maker2, s.makerFee(), sdk.NewInt64Coin(s.ob, 600))
	s.maker2Sell = s.requireSaved(t, res1.Height, msg1, s.maker2, "sell", "600", "4")
	want.paid(t, s, s.maker2, res1, s.makerFee())
	want.move(s.maker2, s.ob, -600)

	res, msgID := s.place(t, s.taker, "buy", 599, "4")
	s.requireFees(t, res, s.taker, s.takerFee(), sdk.NewInt64Coin(chain.Denom, 599*4))
	want.paid(t, s, s.taker, res, s.takerFee())
	s.requireExecuted(t, res.Height, msgID, s.taker, obExec{s.maker2Sell, s.maker2, "sell", "598", "4"})
	s.requireNotSaved(t, res.Height, msgID)
	require.Equal(t, sdkmath.NewInt(4), s.transferred(t, s.blockEvents(t, res.Height), s.tradebin, s.taker.FormattedAddress(), chain.Denom), "1 ob × 4 refunded")
	s.requireRefunded(t, res.Height, msgID, s.taker, "buy", "1", "4"+chain.Denom, "below_min_amount")
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
func (s *orderBookSuite) dust(t *testing.T) {
	want := s.snapshot(t)
	require.Empty(t, s.userDust(t, s.maker2))
	require.Empty(t, s.userDust(t, s.taker))

	buys := make([]string, 0, 3)
	for _, c := range []struct {
		paid int64
		dust map[string]string
	}{
		{24, obDust("0.5")},
		{23, obDust("")},
		{24, obDust("0.5")},
	} {
		res, msgID := s.place(t, s.maker2, "buy", 10, "2.35")
		s.requireFees(t, res, s.maker2, s.makerFee(), sdk.NewInt64Coin(chain.Denom, c.paid))
		want.paid(t, s, s.maker2, res, s.makerFee())
		want.move(s.maker2, chain.Denom, -c.paid)
		buys = append(buys, s.requireSaved(t, res.Height, msgID, s.maker2, "buy", "10", "2.35"))
		require.Equal(t, c.dust, s.userDust(t, s.maker2), "maker2 dust after paying %d", c.paid)
	}
	s.requireBook(t, obBook{Buy: map[string]string{"2.35": "30", "2": "600"}, Sell: map[string]string{"4": "2"}})

	res, msgID := s.place(t, s.taker, "sell", 30, "2.35")
	s.requireFees(t, res, s.taker, s.takerFee(), sdk.NewInt64Coin(s.ob, 30))
	want.paid(t, s, s.taker, res, s.takerFee())
	s.requireExecuted(t, res.Height, msgID, s.taker,
		obExec{buys[0], s.maker2, "buy", "10", "2.35"},
		obExec{buys[1], s.maker2, "buy", "10", "2.35"},
		obExec{buys[2], s.maker2, "buy", "10", "2.35"},
	)
	// 23 (dust 0.5), 23 + 1 (dust 1 paid out), 23 (dust 0.5)
	block := filterType(s.blockEvents(t, res.Height), "transfer")
	got := helpers.EventsWith(block, map[string]string{"sender": s.tradebin, "recipient": s.taker.FormattedAddress()})
	amounts := make([]string, 0, len(got))
	for _, e := range got {
		amounts = append(amounts, e.Attributes["amount"])
	}
	require.Equal(t, []string{"23" + chain.Denom, "24" + chain.Denom, "23" + chain.Denom}, amounts, "taker receipts")
	want.move(s.taker, s.ob, -30)
	want.move(s.taker, chain.Denom, 70)
	want.move(s.maker2, s.ob, 30)

	s.requireBalances(t, want)
	require.Equal(t, obDust("0.5"), s.userDust(t, s.taker), "taker dust")
	require.Equal(t, obDust("0.5"), s.userDust(t, s.maker2), "maker2 dust")
	s.requireBook(t, obBook{Buy: map[string]string{"2": "600"}, Sell: map[string]string{"4": "2"}})
}

// 8. MsgFillOrders on buy orders sells ob at two prices in one message: one
// queue message per price, one taker fee for the whole message.
func (s *orderBookSuite) fillBuyOrders(t *testing.T) {
	want, history := s.snapshot(t), s.history(t)
	var buy18 string
	for _, price := range []string{"1.8", "1.5"} {
		res, msgID := s.place(t, s.maker1, "buy", 100, price)
		s.requireFees(t, res, s.maker1, s.makerFee(), sdk.NewCoin(chain.Denom, mulPrice(100, price)))
		want.paid(t, s, s.maker1, res, s.makerFee())
		want.move(s.maker1, chain.Denom, -mulPrice(100, price).Int64())
		id := s.requireSaved(t, res.Height, msgID, s.maker1, "buy", "100", price)
		if price == "1.8" {
			buy18 = id
		}
	}

	res := s.mustExecMsgs(t, s.taker, obFillGas, s.fillMsg(s.taker, "buy", obItem{"2", "100"}, obItem{"1.8", "100"}))
	s.requireFees(t, res, s.taker, s.takerFee(), sdk.NewInt64Coin(s.ob, 200))
	want.paid(t, s, s.taker, res, s.takerFee())
	ids := s.queuedMessages(t, res, s.taker, "sell", "2", "1.8")
	s.requireExecuted(t, res.Height, ids[0], s.taker, obExec{s.maker1Buy, s.maker1, "buy", "100", "2"})
	s.requireExecuted(t, res.Height, ids[1], s.taker, obExec{buy18, s.maker1, "buy", "100", "1.8"})
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
// below the minimum is skipped (never queued nor charged). A message whose
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
	ids := s.queuedMessages(t, res, s.taker, "sell", "1.7", "1.5")
	s.requireExecuted(t, res.Height, ids[0], s.taker)
	s.requireExecuted(t, res.Height, ids[1], s.taker, obExec{buy15ID, s.maker1, "buy", "100", "1.5"})
	require.Equal(t, sdkmath.NewInt(50+50), s.transferred(t, s.blockEvents(t, res.Height), s.tradebin, s.taker.FormattedAddress(), s.ob), "50 ob refunded at 1.7, 50 at 1.5")
	s.requireRefunded(t, res.Height, ids[0], s.taker, "sell", "50", "50"+s.ob, "no_match")
	s.requireRefunded(t, res.Height, ids[1], s.taker, "sell", "50", "50"+s.ob, "unfilled_remainder")
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
	res, msgID := s.place(t, s.maker1, "buy", 10, "2.35")
	s.requireFees(t, res, s.maker1, s.makerFee(), sdk.NewInt64Coin(chain.Denom, 24))
	id := s.requireSaved(t, res.Height, msgID, s.maker1, "buy", "10", "2.35")
	require.Equal(t, obDust("0.5"), s.userDust(t, s.maker1))
	want.paid(t, s, s.maker1, res, s.makerFee())
	want.move(s.maker1, chain.Denom, -24)

	res = s.mustExec(t, s.maker1, "tradebin", "cancel-order", s.market, id, "buy")
	s.requireCanceled(t, res, s.maker1, id, "buy", "10", "2.35")
	require.Equal(t, sdkmath.NewInt(24), s.transferred(t, s.blockEvents(t, res.Height), s.tradebin, s.maker1.FormattedAddress(), chain.Denom), "23 refunded + 1 of dust")
	want.paid(t, s, s.maker1, res, sdkmath.ZeroInt())
	want.move(s.maker1, chain.Denom, 24)
	require.Empty(t, s.userDust(t, s.maker1), "dust settled")
	s.requireBalances(t, want)
	s.requireBook(t, obBook{Sell: map[string]string{"4": "2"}})
}

// 11. Only the owner cancels an order, only a resting order can be
// cancelled, and a second cancel of an order with one already queued is
// refused (here both in one tx: nothing is queued, the order stays).
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
		s.requireRefused(t, res, err, 0, "cancel already pending for order "+s.maker2Sell)
		want.paid(t, s, s.maker2, res, sdkmath.ZeroInt())
		s.requireBalances(t, want)
		require.Equal(t, book, s.book(t), "order book changed")
		s.requireOrders(t, s.maker2, obOrder{ID: s.maker2Sell, OrderType: "sell", Amount: "2", Price: "4", Owner: s.maker2.FormattedAddress()})
	})
}

// 12. maker2 fills its own resting sell and cancels it in one tx. The fill is
// queued first, so the engine executes it and the cancel finds nothing: no
// OrderCanceledEvent but an OrderCancelSkippedEvent, no refund, the ob locked
// by the order comes back through the fill.
func (s *orderBookSuite) fillBeforeCancel(t *testing.T) {
	want := s.snapshot(t)
	res := s.mustExecMsgs(t, s.maker2, obFillGas,
		s.fillMsg(s.maker2, "sell", obItem{"4", "2"}),
		s.cancelMsg(s.maker2, "sell", s.maker2Sell),
	)
	s.requireFees(t, res, s.maker2, s.takerFee(), sdk.NewInt64Coin(chain.Denom, 8))
	want.paid(t, s, s.maker2, res, s.takerFee())
	fill := s.queuedMessages(t, res, s.maker2, "buy", "4")[0]
	cancel := s.requireTxEvent(t, res, "OrderCancelMessageEvent", map[string]string{"orderId": s.maker2Sell}).Attributes["message_id"]
	require.Greater(t, cancel, fill, "the cancel is queued after the fill")

	s.requireExecuted(t, res.Height, fill, s.maker2, obExec{s.maker2Sell, s.maker2, "sell", "2", "4"})
	canceled, err := helpers.BlockEvents(s.ctx, s.bze, res.Height, tradebinEventPrefix+"OrderCanceledEvent")
	require.NoError(t, err)
	require.Empty(t, canceled, "the cancel found no order")
	skipped := s.engineEvents(t, res.Height, "OrderCancelSkippedEvent", map[string]string{"message_id": cancel})
	require.Len(t, skipped, 1, "the skipped cancel is reported")
	e := skipped[0].Attributes
	require.Equal(t, []string{s.market, "sell", s.maker2Sell, s.maker2.FormattedAddress(), "order_not_found"},
		[]string{e["market_id"], e["order_type"], e["order_id"], e["owner"], e["reason"]})
	want.move(s.maker2, s.ob, 2)

	s.requireBalances(t, want)
	s.requireBook(t, obBook{})
	s.requireOrders(t, s.maker2)
}

// 13. A buy above the best resting sell, a sell below the best resting buy,
// and a sell below a buy still queued in the same block are refused.
func (s *orderBookSuite) crossedBook(t *testing.T) {
	want := s.snapshot(t)
	res, msgID := s.place(t, s.maker1, "sell", 100, "6")
	s.requireFees(t, res, s.maker1, s.makerFee(), sdk.NewInt64Coin(s.ob, 100))
	s.requireSaved(t, res.Height, msgID, s.maker1, "sell", "100", "6")
	want.paid(t, s, s.maker1, res, s.makerFee())
	want.move(s.maker1, s.ob, -100)
	res, msgID = s.place(t, s.maker2, "buy", 100, "1")
	s.requireFees(t, res, s.maker2, s.makerFee(), sdk.NewInt64Coin(chain.Denom, 100))
	s.requireSaved(t, res.Height, msgID, s.maker2, "buy", "100", "1")
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
		{"sell below a queued buy", []any{s.orderMsg(s.taker, "buy", 10, "5"), s.orderMsg(s.taker, "sell", 10, "4.5")}, "price is outdated. Better price is available: 5.000000000000000000"},
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

// 14. Orders the msg server or ValidateBasic refuse: nothing is locked or
// queued, a refusal in the msg server costs the tx fee only, one in
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
	res1, msg1 := s.place(t, s.maker1, "sell", 50, "5")
	s.requireFees(t, res1, s.maker1, s.makerFee(), sdk.NewInt64Coin(s.ob, 50))
	sell := s.requireSaved(t, res1.Height, msg1, s.maker1, "sell", "50", "5")
	res2, msg2 := s.place(t, s.maker1, "buy", 50, "5")
	s.requireFees(t, res2, s.maker1, s.takerFee(), sdk.NewInt64Coin(chain.Denom, 250))
	s.requireExecuted(t, res2.Height, msg2, s.maker1, obExec{sell, s.maker1, "sell", "50", "5"})
	s.requireNotSaved(t, res2.Height, msg2)

	want.paid(t, s, s.maker1, res1, s.makerFee())
	want.paid(t, s, s.maker1, res2, s.takerFee())
	s.requireBalances(t, want)
	s.requireBook(t, obBook{Buy: map[string]string{"1": "100"}, Sell: map[string]string{"6": "100"}})
	s.requireNewHistory(t, history, obHistory{"sell", "50", "5", s.maker1.FormattedAddress(), s.maker1.FormattedAddress()})
}

// 16. With orderBookPerBlockMessages lowered to 3 by governance, five orders
// queued in one block are saved three in that block's EndBlock and the other
// two, in order, in the next one.
func (s *orderBookSuite) perBlockCap(t *testing.T) {
	s.updateParams(t, "Order book cap", map[string]uint64{"orderBookPerBlockMessages": obPerBlockCap})

	want := s.snapshot(t)
	prices := []string{"10", "11", "12", "13", "14"}
	res := s.mustExecMsgs(t, s.maker1, obBatchGas, s.batch(s.maker1, prices)...)
	s.requireFees(t, res, s.maker1, s.makerFee().MulRaw(obBatchOrders), sdk.NewInt64Coin(s.ob, 10*obBatchOrders))
	want.paid(t, s, s.maker1, res, s.makerFee().MulRaw(obBatchOrders))
	want.move(s.maker1, s.ob, -10*obBatchOrders)
	ids := s.queuedMessages(t, res, s.maker1, "sell", prices...)

	require.NoError(t, testutil.WaitForBlocks(s.ctx, 2, s.bze))
	require.Equal(t, ids[:obPerBlockCap], s.savedIDs(t, res.Height), "the first three in the tx's block")
	require.Equal(t, ids[obPerBlockCap:], s.savedIDs(t, res.Height+1), "the other two in the next block")
	s.requireBalances(t, want)
	s.requireBook(t, obBook{
		Buy:  map[string]string{"1": "100"},
		Sell: map[string]string{"6": "100", "10": "10", "11": "10", "12": "10", "13": "10", "14": "10"},
	})
}

// 17. Every order placed while more than orderBookExtraGasWindow messages are
// queued costs (queued − window) × orderBookQueueExtraGas extra gas. The same
// five-order tx runs with the default window (no surcharge) and again with
// the window lowered to 2 by governance: the 4th and 5th orders see 3 and 4
// queued messages, so the second tx uses exactly 75,000 gas more. The params
// are restored afterwards.
func (s *orderBookSuite) queueSurcharge(t *testing.T) {
	window := s.paramUint(t, "orderBookExtraGasWindow")
	extra := s.paramUint(t, "orderBookQueueExtraGas")
	require.Greater(t, window, uint64(obBatchOrders), "the default window must not surcharge five orders")

	prices := []string{"20", "21", "22", "23", "24"}
	run := func() helpers.TxResult {
		want := s.snapshot(t)
		res := s.mustExecMsgs(t, s.maker2, obBatchGas, s.batch(s.maker2, prices)...)
		s.requireFees(t, res, s.maker2, s.makerFee().MulRaw(obBatchOrders), sdk.NewInt64Coin(s.ob, 10*obBatchOrders))
		want.paid(t, s, s.maker2, res, s.makerFee().MulRaw(obBatchOrders))
		want.move(s.maker2, s.ob, -10*obBatchOrders)
		require.NoError(t, testutil.WaitForBlocks(s.ctx, 2, s.bze))
		s.requireBalances(t, want)
		return res
	}

	base := run()
	s.updateParams(t, "Order book queue window", map[string]uint64{"orderBookExtraGasWindow": obQueueWindow})
	surcharged := run()

	var want uint64
	for queued := uint64(0); queued < obBatchOrders; queued++ {
		if queued > obQueueWindow {
			want += (queued - obQueueWindow) * extra
		}
	}
	require.Equal(t, uint64(75_000), want, "the scenario's surcharge")
	require.Equal(t, int64(want), surcharged.GasUsed-base.GasUsed, "gas used %d without and %d with the surcharge", base.GasUsed, surcharged.GasUsed)

	s.updateParams(t, "Order book defaults", map[string]uint64{
		"orderBookPerBlockMessages": s.originalUint(t, "orderBookPerBlockMessages"),
		"orderBookExtraGasWindow":   window,
	})
	require.Equal(t, s.defaults, s.rawParams(t), "params restored")
	s.requireBook(t, obBook{
		Buy: map[string]string{"1": "100"},
		Sell: map[string]string{"6": "100", "10": "10", "11": "10", "12": "10", "13": "10", "14": "10",
			"20": "20", "21": "20", "22": "20", "23": "20", "24": "20"},
	})
}

// 18. The tradebin module holds, on top of what it held before the market
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
// the user to the tradebin module, the fee on to the burner fee collector,
// and the EndBlock the fee to the burner.
func (s *orderBookSuite) requireFees(t *testing.T, res helpers.TxResult, u ibc.Wallet, fee sdkmath.Int, locked sdk.Coin) {
	t.Helper()
	user := u.FormattedAddress()
	toModule := sdk.NewCoins(sdk.NewCoin(chain.Denom, fee)).Add(locked)
	require.Equal(t, toModule.AmountOf(chain.Denom), s.transferred(t, res.Events, user, s.tradebin, chain.Denom), "ubze to the tradebin module")
	require.Equal(t, toModule.AmountOf(s.ob), s.transferred(t, res.Events, user, s.tradebin, s.ob), "ob to the tradebin module")
	require.Equal(t, fee, s.transferred(t, res.Events, s.tradebin, s.burnerFees, chain.Denom), "trading fee to the burner fee collector")
	require.Equal(t, fee, s.transferred(t, s.blockEvents(t, res.Height), s.burnerFees, s.burner, chain.Denom), "trading fee to the burner in EndBlock")
}

// place sends a create-order and returns the tx and its queue message id.
func (s *orderBookSuite) place(t *testing.T, u ibc.Wallet, typ string, amount int64, price string) (helpers.TxResult, string) {
	t.Helper()
	amt := strconv.FormatInt(amount, 10)
	res := s.mustExec(t, u, "tradebin", "create-order", typ, amt, price, s.market)
	ev := s.requireTxEvent(t, res, "OrderCreateMessageEvent", map[string]string{
		"creator": u.FormattedAddress(), "market_id": s.market, "order_type": typ, "amount": amt, "price": price,
	})
	return res, ev.Attributes["message_id"]
}

// queuedMessages returns the message ids of the OrderCreateMessageEvents of a
// tx, one per price, in the order of prices.
func (s *orderBookSuite) queuedMessages(t *testing.T, res helpers.TxResult, u ibc.Wallet, typ string, prices ...string) []string {
	t.Helper()
	events := helpers.TxEvents(res, tradebinEventPrefix+"OrderCreateMessageEvent")
	require.Len(t, events, len(prices), "one queued message per price: %+v", events)
	ids := make([]string, 0, len(prices))
	for i, e := range events {
		require.Equal(t, prices[i], e.Attributes["price"])
		require.Equal(t, typ, e.Attributes["order_type"])
		require.Equal(t, u.FormattedAddress(), e.Attributes["creator"])
		ids = append(ids, e.Attributes["message_id"])
	}
	return ids
}

func (s *orderBookSuite) engineEvents(t *testing.T, height int64, name string, attrs map[string]string) []helpers.Event {
	t.Helper()
	events, err := helpers.BlockEvents(s.ctx, s.bze, height, tradebinEventPrefix+name)
	require.NoError(t, err)
	return helpers.EventsWith(events, attrs)
}

// requireSaved asserts the EndBlock saved the message as one resting order
// and returns its id.
func (s *orderBookSuite) requireSaved(t *testing.T, height int64, msgID string, owner ibc.Wallet, typ, amount, price string) string {
	t.Helper()
	saved := s.engineEvents(t, height, "OrderSavedEvent", map[string]string{"message_id": msgID})
	require.Len(t, saved, 1, "message %s saved once at height %d", msgID, height)
	e := saved[0]
	require.Equal(t, map[string]string{"market_id": s.market, "owner": owner.FormattedAddress(), "order_type": typ, "amount": amount, "price": price},
		map[string]string{"market_id": e.Attributes["market_id"], "owner": e.Attributes["owner"], "order_type": e.Attributes["order_type"], "amount": e.Attributes["amount"], "price": e.Attributes["price"]})
	order, found := s.marketOrder(t, typ, e.Attributes["id"])
	require.True(t, found, "saved order %s is on the book", e.Attributes["id"])
	require.Equal(t, amount, order.Amount)
	return e.Attributes["id"]
}

// requireRefunded asserts the one OrderMessageRefundedEvent of a queue
// message: the unexecuted base amount, the coin sent back and the reason.
func (s *orderBookSuite) requireRefunded(t *testing.T, height int64, msgID string, owner ibc.Wallet, typ, amount, refund, reason string) {
	t.Helper()
	got := s.engineEvents(t, height, "OrderMessageRefundedEvent", map[string]string{"message_id": msgID})
	require.Len(t, got, 1, "refunds of message %s at height %d", msgID, height)
	e := got[0].Attributes
	require.Equal(t,
		[]string{s.market, typ, owner.FormattedAddress(), amount, refund, reason},
		[]string{e["market_id"], e["order_type"], e["owner"], e["amount"], e["refund"], e["reason"]},
		"refund of message %s", msgID)
}

func (s *orderBookSuite) requireNotSaved(t *testing.T, height int64, msgID string) {
	t.Helper()
	require.Empty(t, s.engineEvents(t, height, "OrderSavedEvent", map[string]string{"message_id": msgID}), "message %s saved", msgID)
}

// savedIDs returns the message ids of the orders saved at a height, in order.
func (s *orderBookSuite) savedIDs(t *testing.T, height int64) []string {
	t.Helper()
	var ids []string
	for _, e := range s.engineEvents(t, height, "OrderSavedEvent", map[string]string{"market_id": s.market}) {
		ids = append(ids, e.Attributes["message_id"])
	}
	return ids
}

// requireExecuted asserts the OrderExecutedEvents of one queue message, in
// the order the engine filled them.
func (s *orderBookSuite) requireExecuted(t *testing.T, height int64, msgID string, taker ibc.Wallet, want ...obExec) {
	t.Helper()
	got := s.engineEvents(t, height, "OrderExecutedEvent", map[string]string{"message_id": msgID})
	require.Len(t, got, len(want), "executions of message %s: %+v", msgID, got)
	for i, w := range want {
		e := got[i].Attributes
		require.Equal(t,
			[]string{w.ID, s.market, w.OrderType, w.Amount, w.Price, w.Maker.FormattedAddress(), taker.FormattedAddress()},
			[]string{e["id"], e["market_id"], e["order_type"], e["amount"], e["price"], e["maker"], e["taker"]},
			"execution %d of message %s", i, msgID)
	}
}

// requireCanceled asserts the cancel's message id reaches the EndBlock
// OrderCanceledEvent and the order left the book.
func (s *orderBookSuite) requireCanceled(t *testing.T, res helpers.TxResult, owner ibc.Wallet, id, typ, amount, price string) {
	t.Helper()
	msgID := s.requireTxEvent(t, res, "OrderCancelMessageEvent", map[string]string{"orderId": id}).Attributes["message_id"]
	canceled := s.engineEvents(t, res.Height, "OrderCanceledEvent", map[string]string{"id": id})
	require.Len(t, canceled, 1)
	e := canceled[0].Attributes
	require.Equal(t, []string{msgID, owner.FormattedAddress(), typ, amount, price}, []string{e["message_id"], e["owner"], e["order_type"], e["amount"], e["price"]})
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
func (s *orderBookSuite) batch(u ibc.Wallet, prices []string) []any {
	msgs := make([]any, 0, len(prices))
	for _, p := range prices {
		msgs = append(msgs, s.orderMsg(u, "sell", 10, p))
	}
	return msgs
}

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

// rawParams returns the tradebin params as the query prints them; a
// MsgUpdateParams carries them back with some fields replaced.
func (s *orderBookSuite) rawParams(t *testing.T) map[string]json.RawMessage {
	t.Helper()
	res, err := helpers.Query[struct {
		Params map[string]json.RawMessage `json:"params"`
	}](s.ctx, s.bze, "tradebin", "params")
	require.NoError(t, err)
	require.NotEmpty(t, res.Params)
	return res.Params
}

func uintParam(t *testing.T, params map[string]json.RawMessage, name string) uint64 {
	t.Helper()
	raw, ok := params[name]
	require.True(t, ok, "param %s missing: %v", name, params)
	v, err := strconv.ParseUint(strings.Trim(string(raw), `"`), 10, 64)
	require.NoError(t, err, "param %s = %s", name, raw)
	return v
}

func (s *orderBookSuite) paramUint(t *testing.T, name string) uint64 {
	t.Helper()
	return uintParam(t, s.rawParams(t), name)
}

func (s *orderBookSuite) originalUint(t *testing.T, name string) uint64 {
	t.Helper()
	return uintParam(t, s.defaults, name)
}

// updateParams passes a tradebin MsgUpdateParams carrying the current params
// with the given uint64 fields replaced, and checks only those changed.
func (s *orderBookSuite) updateParams(t *testing.T, title string, set map[string]uint64) {
	t.Helper()
	params := s.rawParams(t)
	want := make(map[string]json.RawMessage, len(params))
	for k, v := range params {
		want[k] = v
	}
	for k, v := range set {
		_, ok := params[k]
		require.True(t, ok, "unknown tradebin param %s", k)
		want[k] = json.RawMessage(strconv.Quote(strconv.FormatUint(v, 10)))
	}
	_, err := helpers.SubmitAndPassProposal(s.ctx, s.bze, s.admin, helpers.Proposal{
		Messages: []any{map[string]any{
			"@type":     "/bze.tradebin.MsgUpdateParams",
			"authority": s.gov,
			"params":    want,
		}},
		Title:   title,
		Summary: "e2e: " + title,
	})
	require.NoError(t, err)
	require.Equal(t, want, s.rawParams(t), "params after %q", title)
}

// mulPrice is amount × price for a price that makes it whole.
func mulPrice(amount int64, price string) sdkmath.Int {
	v := sdkmath.LegacyMustNewDecFromStr(price).MulInt64(amount)
	if !v.IsInteger() {
		panic("mulPrice: " + v.String() + " is not whole")
	}
	return v.TruncateInt()
}
