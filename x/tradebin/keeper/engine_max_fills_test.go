package keeper_test

import (
	"fmt"

	"cosmossdk.io/math"
	"github.com/bze-alphateam/bze/x/tradebin/keeper"
	"github.com/bze-alphateam/bze/x/tradebin/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/gogoproto/proto"
	"go.uber.org/mock/gomock"
)

// makerAddr returns a distinct maker address per index
func makerAddr(i int) sdk.AccAddress {
	return sdk.AccAddress(fmt.Sprintf("maker%015d", i))
}

// restSells rests count sell orders of amount at price, each from a distinct maker, and returns them
// in book (FIFO) order
func (suite *IntegrationTestSuite) restSells(count int, amount, price string) []types.Order {
	engine := suite.newEngine()
	for i := 0; i < count; i++ {
		suite.placeOrders(engine, limitOrder(types.OrderTypeSell, amount, price, makerAddr(i)))
	}

	var resting []types.Order
	suite.k.IteratePriceOrders(suite.ctx, getMarketId(), types.OrderTypeSell, price, func(ref types.OrderReference) bool {
		o, found := suite.k.GetOrder(suite.ctx, ref.MarketId, ref.OrderType, ref.Id)
		suite.Require().True(found)
		resting = append(resting, o)

		return false
	})
	suite.Require().Len(resting, count)

	return resting
}

// expectFills expects the settlement of a full fill of each maker at price 1: the maker gets amount
// quote, the taker gets amount base
func (suite *IntegrationTestSuite) expectFills(makers []types.Order, taker sdk.AccAddress, amount int64) {
	for _, m := range makers {
		owner, err := sdk.AccAddressFromBech32(m.Owner)
		suite.Require().NoError(err)
		suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, owner, sdk.NewCoins(sdk.NewInt64Coin(market.Quote, amount))).Return(nil).Times(1)
	}
	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, taker, sdk.NewCoins(sdk.NewInt64Coin(market.Base, amount))).Return(nil).Times(len(makers))
}

func executedEvents(events []proto.Message) []*types.OrderExecutedEvent {
	var result []*types.OrderExecutedEvent
	for _, ev := range events {
		if e, ok := ev.(*types.OrderExecutedEvent); ok {
			result = append(result, e)
		}
	}

	return result
}

func savedEvents(events []proto.Message) []*types.OrderSavedEvent {
	var result []*types.OrderSavedEvent
	for _, ev := range events {
		if e, ok := ev.(*types.OrderSavedEvent); ok {
			result = append(result, e)
		}
	}

	return result
}

// requireFilledInOrder checks that exactly the given makers were filled, in that order
func (suite *IntegrationTestSuite) requireFilledInOrder(events []proto.Message, makers []types.Order, amount string) {
	executed := executedEvents(events)
	suite.Require().Len(executed, len(makers))
	for i, m := range makers {
		suite.Require().Equal(m.Id, executed[i].Id)
		suite.Require().Equal(amount, executed[i].Amount)
	}
}

func (suite *IntegrationTestSuite) requireResting(order types.Order) {
	got, found := suite.k.GetOrder(suite.ctx, order.MarketId, order.OrderType, order.Id)
	suite.Require().True(found, "order %s should still rest", order.Id)
	suite.Require().Equal(order, got)
}

func (suite *IntegrationTestSuite) requireGone(order types.Order) {
	_, found := suite.k.GetOrder(suite.ctx, order.MarketId, order.OrderType, order.Id)
	suite.Require().False(found, "order %s should be filled", order.Id)
}

func (suite *IntegrationTestSuite) TestEngine_NewProcessingEngine_ZeroMaxFillsIsRefused() {
	_, err := keeper.NewProcessingEngine(suite.k, suite.bankMock, 0, suite.k.Logger())
	suite.Require().ErrorContains(err, "max fills per order must be at least 1")
}

// A taker larger than a level of maxFills+3 makers fills exactly maxFills of them and gets the
// remainder back; the orders after the cap are untouched.
func (suite *IntegrationTestSuite) TestEngine_MaxFills_PlaceOrderStopsAtTheCapAndRefunds() {
	suite.k.SetMarket(suite.ctx, market)
	const maxFills = 3
	makers := suite.restSells(maxFills+3, "10", "1")
	taker := sdk.AccAddress("addr2_______________")

	suite.expectFills(makers[:maxFills], taker, 10)
	// 100 - 3*10 = 70 base refunded as 70 quote
	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, taker, sdk.NewCoins(sdk.NewInt64Coin(market.Quote, 70))).Return(nil).Times(1)

	suite.resetEvents()
	suite.Require().NoError(suite.newEngineWithMaxFills(maxFills).PlaceOrder(suite.ctx, limitOrder(types.OrderTypeBuy, "100", "1", taker)))

	events := suite.tradebinTypedEvents()
	suite.requireFilledInOrder(events, makers[:maxFills], "10")
	refunds := refundedEvents(events)
	suite.Require().Len(refunds, 1)
	suite.Require().Equal(types.RefundReasonMaxFillsReached, refunds[0].Reason)
	suite.Require().Equal("70", refunds[0].Amount)
	suite.Require().Equal(sdk.NewInt64Coin(market.Quote, 70).String(), refunds[0].Refund)
	// never rested: both sides would rest at the same price
	suite.Require().Empty(savedEvents(events))
	_, found := suite.k.GetAggregatedOrder(suite.ctx, getMarketId(), types.OrderTypeBuy, "1")
	suite.Require().False(found)

	for _, m := range makers[:maxFills] {
		suite.requireGone(m)
	}
	for _, m := range makers[maxFills:] {
		suite.requireResting(m)
	}
	suite.checkAggregatedOrder(getMarketId(), types.OrderTypeSell, "1", "30")
	suite.Require().Len(suite.k.GetAllHistoryOrder(suite.ctx), maxFills)
}

func (suite *IntegrationTestSuite) TestEngine_MaxFills_FillOrderStopsAtTheCapAndRefunds() {
	suite.k.SetMarket(suite.ctx, market)
	const maxFills = 3
	makers := suite.restSells(maxFills+3, "10", "1")
	taker := sdk.AccAddress("addr2_______________")

	suite.expectFills(makers[:maxFills], taker, 10)
	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, taker, sdk.NewCoins(sdk.NewInt64Coin(market.Quote, 70))).Return(nil).Times(1)

	suite.resetEvents()
	suite.Require().NoError(suite.newEngineWithMaxFills(maxFills).FillOrder(suite.ctx, limitOrder(types.OrderTypeBuy, "100", "1", taker)))

	events := suite.tradebinTypedEvents()
	suite.requireFilledInOrder(events, makers[:maxFills], "10")
	refunds := refundedEvents(events)
	suite.Require().Len(refunds, 1)
	suite.Require().Equal(types.RefundReasonMaxFillsReached, refunds[0].Reason)
	suite.Require().Equal("70", refunds[0].Amount)

	for _, m := range makers[maxFills:] {
		suite.requireResting(m)
	}
	suite.checkAggregatedOrder(getMarketId(), types.OrderTypeSell, "1", "30")
}

// A level of exactly maxFills makers is emptied by the last allowed fill: the cap did not stop
// anything, so a limit order's remainder rests.
func (suite *IntegrationTestSuite) TestEngine_MaxFills_LevelOfExactlyTheCapIsEmptiedAndTheRemainderRests() {
	suite.k.SetMarket(suite.ctx, market)
	const maxFills = 3
	makers := suite.restSells(maxFills, "10", "1")
	taker := sdk.AccAddress("addr2_______________")

	suite.expectFills(makers, taker, 10)

	suite.resetEvents()
	suite.Require().NoError(suite.newEngineWithMaxFills(maxFills).PlaceOrder(suite.ctx, limitOrder(types.OrderTypeBuy, "100", "1", taker)))

	events := suite.tradebinTypedEvents()
	suite.requireFilledInOrder(events, makers, "10")
	suite.Require().Empty(refundedEvents(events))
	saved := savedEvents(events)
	suite.Require().Len(saved, 1)
	suite.Require().Equal("70", saved[0].Amount)
	_, found := suite.k.GetAggregatedOrder(suite.ctx, getMarketId(), types.OrderTypeSell, "1")
	suite.Require().False(found)
	suite.checkAggregatedOrder(getMarketId(), types.OrderTypeBuy, "1", "70")
}

// Below the cap the level is emptied: a limit order's remainder rests, a fill order's remainder is
// refunded as unfilled.
func (suite *IntegrationTestSuite) TestEngine_MaxFills_LevelBelowTheCap() {
	const maxFills = 3
	taker := sdk.AccAddress("addr2_______________")

	suite.Run("limit order rests", func() {
		suite.SetupTest()
		suite.k.SetMarket(suite.ctx, market)
		makers := suite.restSells(maxFills-1, "10", "1")
		suite.expectFills(makers, taker, 10)

		suite.resetEvents()
		suite.Require().NoError(suite.newEngineWithMaxFills(maxFills).PlaceOrder(suite.ctx, limitOrder(types.OrderTypeBuy, "100", "1", taker)))

		events := suite.tradebinTypedEvents()
		suite.requireFilledInOrder(events, makers, "10")
		suite.Require().Empty(refundedEvents(events))
		suite.checkAggregatedOrder(getMarketId(), types.OrderTypeBuy, "1", "80")
	})

	suite.Run("fill order refunds unfilled remainder", func() {
		suite.SetupTest()
		suite.k.SetMarket(suite.ctx, market)
		makers := suite.restSells(maxFills-1, "10", "1")
		suite.expectFills(makers, taker, 10)
		suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, taker, sdk.NewCoins(sdk.NewInt64Coin(market.Quote, 80))).Return(nil).Times(1)

		suite.resetEvents()
		suite.Require().NoError(suite.newEngineWithMaxFills(maxFills).FillOrder(suite.ctx, limitOrder(types.OrderTypeBuy, "100", "1", taker)))

		events := suite.tradebinTypedEvents()
		suite.requireFilledInOrder(events, makers, "10")
		refunds := refundedEvents(events)
		suite.Require().Len(refunds, 1)
		suite.Require().Equal(types.RefundReasonUnfilledRemainder, refunds[0].Reason)
		suite.Require().Equal("80", refunds[0].Amount)
	})
}

func (suite *IntegrationTestSuite) TestEngine_MaxFills_OneFillPerOrder() {
	suite.k.SetMarket(suite.ctx, market)
	makers := suite.restSells(2, "10", "1")
	taker := sdk.AccAddress("addr2_______________")
	engine := suite.newEngineWithMaxFills(1)

	suite.expectFills(makers[:1], taker, 10)
	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, taker, sdk.NewCoins(sdk.NewInt64Coin(market.Quote, 10))).Return(nil).Times(1)

	suite.resetEvents()
	suite.Require().NoError(engine.PlaceOrder(suite.ctx, limitOrder(types.OrderTypeBuy, "20", "1", taker)))
	events := suite.tradebinTypedEvents()
	suite.requireFilledInOrder(events, makers[:1], "10")
	refunds := refundedEvents(events)
	suite.Require().Len(refunds, 1)
	suite.Require().Equal(types.RefundReasonMaxFillsReached, refunds[0].Reason)
	suite.requireResting(makers[1])

	// the next order picks up where the previous one stopped
	suite.expectFills(makers[1:], taker, 10)
	suite.resetEvents()
	suite.Require().NoError(engine.PlaceOrder(suite.ctx, limitOrder(types.OrderTypeBuy, "10", "1", taker)))
	suite.requireFilledInOrder(suite.tradebinTypedEvents(), makers[1:], "10")
	suite.Require().Empty(suite.k.GetAllOrder(suite.ctx))
}

// The stop conditions are checked in priority order: a remainder below the minimum amount is refunded
// as such even when the cap is reached at the same time.
func (suite *IntegrationTestSuite) TestEngine_MaxFills_BelowMinAmountTakesPriorityOverTheCap() {
	suite.k.SetMarket(suite.ctx, market)
	makers := suite.restSells(2, "10", "1")
	taker := sdk.AccAddress("addr2_______________")

	// min amount at price 1 is 2: after one fill of 10 the taker has 1 left
	suite.expectFills(makers[:1], taker, 10)
	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, taker, sdk.NewCoins(sdk.NewInt64Coin(market.Quote, 1))).Return(nil).Times(1)

	suite.resetEvents()
	suite.Require().NoError(suite.newEngineWithMaxFills(1).PlaceOrder(suite.ctx, limitOrder(types.OrderTypeBuy, "11", "1", taker)))

	refunds := refundedEvents(suite.tradebinTypedEvents())
	suite.Require().Len(refunds, 1)
	suite.Require().Equal(types.RefundReasonBelowMinAmount, refunds[0].Reason)
	suite.Require().Equal("1", refunds[0].Amount)
}

// A resting order the taker can not fill is skipped and does not count as a fill.
func (suite *IntegrationTestSuite) TestEngine_MaxFills_SkippedOrdersAreNotCounted() {
	suite.k.SetMarket(suite.ctx, market)
	taker := sdk.AccAddress("addr2_______________")
	// a taker of 2 can not fill an order of 3 at price 1 (it would leave 1, below the min amount of 2)
	small := suite.restSells(1, "3", "1")[0]
	suite.placeOrders(suite.newEngine(), limitOrder(types.OrderTypeSell, "100", "1", makerAddr(1)))

	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, makerAddr(1), sdk.NewCoins(sdk.NewInt64Coin(market.Quote, 2))).Return(nil).Times(1)
	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, taker, sdk.NewCoins(sdk.NewInt64Coin(market.Base, 2))).Return(nil).Times(1)

	suite.resetEvents()
	suite.Require().NoError(suite.newEngineWithMaxFills(1).PlaceOrder(suite.ctx, limitOrder(types.OrderTypeBuy, "2", "1", taker)))

	events := suite.tradebinTypedEvents()
	executed := executedEvents(events)
	suite.Require().Len(executed, 1)
	suite.Require().Equal(makerAddr(1).String(), executed[0].Maker)
	suite.Require().Empty(refundedEvents(events))
	suite.requireResting(small)
	suite.checkAggregatedOrder(getMarketId(), types.OrderTypeSell, "1", "101")
}

// Nothing is written while the price level iterator is open: the fills are settled after it closes,
// the partially filled maker is written once and the fully filled one is removed once.
func (suite *IntegrationTestSuite) TestEngine_MaxFills_NoWritesWhileIterating() {
	engine, pk, bank := suite.mockedEngine()
	taker := sdk.AccAddress("addr2_______________")
	order := limitOrder(types.OrderTypeBuy, "15", "1", taker)
	placed := withId(order, orderId(9))
	full := withId(limitOrder(types.OrderTypeSell, "10", "1", makerAddr(0)), orderId(1))
	partial := withId(limitOrder(types.OrderTypeSell, "10", "1", makerAddr(1)), orderId(2))
	untouched := withId(limitOrder(types.OrderTypeSell, "10", "1", makerAddr(2)), orderId(3))
	refOf := func(o types.Order) types.OrderReference {
		return types.OrderReference{Id: o.Id, MarketId: o.MarketId, OrderType: o.OrderType}
	}

	iterating := false
	noWrite := func(what string) {
		if iterating {
			panic(what + " while the price level iterator is open")
		}
	}

	pk.EXPECT().AssignOrderId(gomock.Any(), order).Return(placed).Times(1)
	pk.EXPECT().GetMarketById(gomock.Any(), getMarketId()).Return(market, true).AnyTimes()
	pk.EXPECT().GetAggregatedOrder(gomock.Any(), getMarketId(), types.OrderTypeSell, "1").
		Return(types.AggregatedOrder{MarketId: getMarketId(), OrderType: types.OrderTypeSell, Price: "1", Amount: "30"}, true).Times(1)
	pk.EXPECT().IteratePriceOrders(gomock.Any(), getMarketId(), types.OrderTypeSell, "1", gomock.Any()).
		DoAndReturn(func(ctx sdk.Context, marketId, orderType, price string, fn func(types.OrderReference) bool) {
			iterating = true
			defer func() { iterating = false }()
			iterateRefs(refOf(full), refOf(partial), refOf(untouched))(ctx, marketId, orderType, price, fn)
		}).Times(1)
	pk.EXPECT().GetOrder(gomock.Any(), getMarketId(), types.OrderTypeSell, full.Id).Return(full, true).Times(1)
	pk.EXPECT().GetOrder(gomock.Any(), getMarketId(), types.OrderTypeSell, partial.Id).Return(partial, true).Times(1)
	// reads are allowed while iterating, coins are computed at settlement
	pk.EXPECT().GetOrderCoinsWithDust(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ sdk.Context, args types.OrderCoinsArguments) (types.OrderCoins, error) {
			noWrite("settlement")
			return types.OrderCoins{Coin: sdk.NewCoin(market.Quote, args.OrderAmount), Dust: math.LegacyZeroDec()}, nil
		}).Times(4)
	pk.EXPECT().StoreProcessedUserDust(gomock.Any(), gomock.Any(), gomock.Any()).Do(func(sdk.Context, *types.UserDust, *math.LegacyDec) { noWrite("StoreProcessedUserDust") }).Times(4)
	bank.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ interface{}, _ string, _ sdk.AccAddress, _ sdk.Coins) error {
			noWrite("SendCoinsFromModuleToAccount")
			return nil
		}).Times(4)
	pk.EXPECT().RemoveOrder(gomock.Any(), withAmount(full, "0")).Do(func(sdk.Context, types.Order) { noWrite("RemoveOrder") }).Times(1)
	pk.EXPECT().SaveOrder(gomock.Any(), withAmount(partial, "5")).DoAndReturn(func(_ sdk.Context, o types.Order) types.Order {
		noWrite("SaveOrder")
		return o
	}).Times(1)
	pk.EXPECT().SetHistoryOrder(gomock.Any(), gomock.Any(), gomock.Any()).Do(func(sdk.Context, types.HistoryOrder, string) { noWrite("SetHistoryOrder") }).Times(2)
	pk.EXPECT().GetOnOrderFillHooks().Return(nil).Times(2)
	pk.EXPECT().SetAggregatedOrder(gomock.Any(), types.AggregatedOrder{MarketId: getMarketId(), OrderType: types.OrderTypeSell, Price: "1", Amount: "15"}).
		Do(func(sdk.Context, types.AggregatedOrder) { noWrite("SetAggregatedOrder") }).Times(1)

	suite.Require().NoError(engine.FillOrder(suite.ctx, order))
}

func withAmount(order types.Order, amount string) types.Order {
	order.Amount = amount
	return order
}
