package keeper_test

import (
	"cosmossdk.io/math"
	"github.com/bze-alphateam/bze/x/tradebin/keeper"
	"github.com/bze-alphateam/bze/x/tradebin/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/gogoproto/proto"
	"go.uber.org/mock/gomock"
)

// expectedNextOrderId is the id the next order executed by the engine gets
func (suite *IntegrationTestSuite) expectedNextOrderId() string {
	return orderId(suite.k.GetOrderCounter(suite.ctx))
}

// placeWithEvents executes PlaceOrder on the suite context with a fresh event manager and returns its events
func (suite *IntegrationTestSuite) placeWithEvents(order types.Order) []proto.Message {
	suite.resetEvents()
	suite.Require().NoError(suite.newEngine().PlaceOrder(suite.ctx, order))

	return suite.tradebinTypedEvents()
}

// fillWithEvents executes FillOrder on the suite context with a fresh event manager and returns its events
func (suite *IntegrationTestSuite) fillWithEvents(order types.Order) []proto.Message {
	suite.resetEvents()
	suite.Require().NoError(suite.newEngine().FillOrder(suite.ctx, order))

	return suite.tradebinTypedEvents()
}

func refundedEvents(events []proto.Message) []*types.OrderMessageRefundedEvent {
	var result []*types.OrderMessageRefundedEvent
	for _, ev := range events {
		if e, ok := ev.(*types.OrderMessageRefundedEvent); ok {
			result = append(result, e)
		}
	}

	return result
}

// engineEventOrderId returns the order id carried by an order book engine event
func engineEventOrderId(ev proto.Message) (string, bool) {
	switch e := ev.(type) {
	case *types.OrderExecutedEvent:
		return e.OrderId, true
	case *types.OrderSavedEvent:
		return e.OrderId, true
	case *types.OrderCanceledEvent:
		return e.OrderId, true
	case *types.OrderMessageRefundedEvent:
		return e.OrderId, true
	}

	return "", false
}

func (suite *IntegrationTestSuite) restingSell(owner sdk.AccAddress, amount math.Int, price string) types.Order {
	id := suite.expectedNextOrderId()
	suite.Require().NoError(suite.newEngine().PlaceOrder(suite.ctx, limitOrder(types.OrderTypeSell, amount.String(), price, owner)))
	order, found := suite.k.GetOrder(suite.ctx, getMarketId(), types.OrderTypeSell, id)
	suite.Require().True(found)

	return order
}

func (suite *IntegrationTestSuite) TestOutcomeEvents_FillWithNoMatch() {
	suite.k.SetMarket(suite.ctx, market)
	taker := sdk.AccAddress("addr2_______________")

	// a buy order escrowed quote: amount * price is returned
	expectedRefund := sdk.NewInt64Coin(market.Quote, 10000)
	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, taker, sdk.NewCoins(expectedRefund)).Return(nil).Times(1)

	takerId := suite.expectedNextOrderId()
	refunded := refundedEvents(suite.fillWithEvents(limitOrder(types.OrderTypeBuy, "5000", "2", taker)))
	suite.Require().Len(refunded, 1)
	suite.Require().Equal(takerId, refunded[0].OrderId)
	suite.Require().Equal(getMarketId(), refunded[0].MarketId)
	suite.Require().Equal(types.OrderTypeBuy, refunded[0].OrderType)
	suite.Require().Equal(taker.String(), refunded[0].Owner)
	suite.Require().Equal("5000", refunded[0].Amount)
	suite.Require().Equal(expectedRefund.String(), refunded[0].Refund)
	suite.Require().Equal(types.RefundReasonNoMatch, refunded[0].Reason)
	// a fill never rests on the book
	suite.Require().Empty(suite.k.GetAllOrder(suite.ctx))
	suite.Require().Empty(suite.k.GetAllAggregatedOrder(suite.ctx))
}

func (suite *IntegrationTestSuite) TestOutcomeEvents_PartlyMatchedFill() {
	suite.k.SetMarket(suite.ctx, market)
	maker := sdk.AccAddress("addr1_______________")
	taker := sdk.AccAddress("addr2_______________")
	price := "1"
	minAmount, err := keeper.CalculateMinAmount(price)
	suite.Require().NoError(err)

	suite.restingSell(maker, minAmount, price)

	fillAmount := minAmount.MulRaw(5)
	remainder := fillAmount.Sub(minAmount)
	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, maker, sdk.NewCoins(sdk.NewCoin(market.Quote, minAmount))).Return(nil).Times(1)
	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, taker, sdk.NewCoins(sdk.NewCoin(market.Base, minAmount))).Return(nil).Times(1)
	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, taker, sdk.NewCoins(sdk.NewCoin(market.Quote, remainder))).Return(nil).Times(1)

	takerId := suite.expectedNextOrderId()
	events := suite.fillWithEvents(limitOrder(types.OrderTypeBuy, fillAmount.String(), price, taker))

	executed := 0
	for _, ev := range events {
		if e, ok := ev.(*types.OrderExecutedEvent); ok {
			executed++
			suite.Require().Equal(takerId, e.OrderId)
		}
	}
	suite.Require().Equal(1, executed)

	refunded := refundedEvents(events)
	suite.Require().Len(refunded, 1)
	suite.Require().Equal(takerId, refunded[0].OrderId)
	suite.Require().Equal(remainder.String(), refunded[0].Amount)
	suite.Require().Equal(sdk.NewCoin(market.Quote, remainder).String(), refunded[0].Refund)
	suite.Require().Equal(types.RefundReasonUnfilledRemainder, refunded[0].Reason)
	suite.Require().Empty(suite.k.GetAllOrder(suite.ctx))
}

func (suite *IntegrationTestSuite) TestOutcomeEvents_CreateRemainderBelowMinAmount() {
	suite.k.SetMarket(suite.ctx, market)
	maker := sdk.AccAddress("addr1_______________")
	taker := sdk.AccAddress("addr2_______________")
	price := "1"
	minAmount, err := keeper.CalculateMinAmount(price)
	suite.Require().NoError(err)

	suite.restingSell(maker, minAmount, price)

	// one unit more than the resting order: the remainder (1) is below the min amount
	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, maker, sdk.NewCoins(sdk.NewCoin(market.Quote, minAmount))).Return(nil).Times(1)
	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, taker, sdk.NewCoins(sdk.NewCoin(market.Base, minAmount))).Return(nil).Times(1)
	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, taker, sdk.NewCoins(sdk.NewInt64Coin(market.Quote, 1))).Return(nil).Times(1)

	takerId := suite.expectedNextOrderId()
	events := suite.placeWithEvents(limitOrder(types.OrderTypeBuy, minAmount.AddRaw(1).String(), price, taker))
	for _, ev := range events {
		_, saved := ev.(*types.OrderSavedEvent)
		suite.Require().False(saved, "the remainder must not be placed")
	}

	refunded := refundedEvents(events)
	suite.Require().Len(refunded, 1)
	suite.Require().Equal(takerId, refunded[0].OrderId)
	suite.Require().Equal(types.OrderTypeBuy, refunded[0].OrderType)
	suite.Require().Equal(taker.String(), refunded[0].Owner)
	suite.Require().Equal("1", refunded[0].Amount)
	suite.Require().Equal(sdk.NewInt64Coin(market.Quote, 1).String(), refunded[0].Refund)
	suite.Require().Equal(types.RefundReasonBelowMinAmount, refunded[0].Reason)
	suite.Require().Empty(suite.k.GetAllOrder(suite.ctx))
	suite.Require().Empty(suite.k.GetAllAggregatedOrder(suite.ctx))
}

func (suite *IntegrationTestSuite) TestOutcomeEvents_RefundRoundingToZero() {
	suite.k.SetMarket(suite.ctx, market)
	taker := sdk.AccAddress("addr2_______________")

	// 1 base unit at 0.1 is worth 0.1 quote units: nothing is sent, only dust is stored
	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

	takerId := suite.expectedNextOrderId()
	refunded := refundedEvents(suite.fillWithEvents(limitOrder(types.OrderTypeBuy, "1", "0.1", taker)))
	suite.Require().Len(refunded, 1)
	suite.Require().Equal(takerId, refunded[0].OrderId)
	suite.Require().Equal("1", refunded[0].Amount)
	suite.Require().Empty(refunded[0].Refund)
	suite.Require().Equal(types.RefundReasonNoMatch, refunded[0].Reason)

	dust, found := suite.k.GetUserDust(suite.ctx, taker.String(), market.Quote)
	suite.Require().True(found)
	suite.Require().Equal("0.100000000000000000", dust.Amount)
}

// TestOutcomeEvents_EveryExecutedOrderHasAnEvent executes a mix of orders and cancels and checks that
// every one of them appears, by its order id, in at least one engine event of its own call.
func (suite *IntegrationTestSuite) TestOutcomeEvents_EveryExecutedOrderHasAnEvent() {
	suite.k.SetMarket(suite.ctx, market)
	maker := sdk.AccAddress("addr1_______________")
	taker := sdk.AccAddress("addr2_______________")
	price := "1"
	minAmount, err := keeper.CalculateMinAmount(price)
	suite.Require().NoError(err)
	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	// resting orders
	suite.restingSell(maker, minAmount.MulRaw(3), price)
	toCancel := suite.restingSell(maker, minAmount, "3")

	cases := []struct {
		name  string
		fill  bool
		order types.Order
	}{
		// order that rests (no match at its price)
		{name: "rests", order: limitOrder(types.OrderTypeBuy, minAmount.String(), "0.5", taker)},
		// order fully matched
		{name: "fully matched", order: limitOrder(types.OrderTypeBuy, minAmount.String(), price, taker)},
		// order with a remainder below the min amount
		{name: "remainder below min", order: limitOrder(types.OrderTypeBuy, minAmount.AddRaw(1).String(), price, taker)},
		// fill partly matched (one min amount left at the price)
		{name: "fill partly matched", fill: true, order: limitOrder(types.OrderTypeBuy, minAmount.MulRaw(4).String(), price, taker)},
		// fill with no match
		{name: "fill no match", fill: true, order: limitOrder(types.OrderTypeBuy, minAmount.String(), "7", taker)},
	}

	totalRefunded := 0
	for _, c := range cases {
		id := suite.expectedNextOrderId()
		var events []proto.Message
		if c.fill {
			events = suite.fillWithEvents(c.order)
		} else {
			events = suite.placeWithEvents(c.order)
		}
		suite.Require().NotEmpty(events, c.name)
		for _, ev := range events {
			evId, ok := engineEventOrderId(ev)
			suite.Require().True(ok, c.name)
			suite.Require().Equal(id, evId, "%s: every event of the call carries the taker order id", c.name)
		}
		totalRefunded += len(refundedEvents(events))
	}
	suite.Require().Equal(3, totalRefunded)

	// cancel of an existing order: its own id
	suite.resetEvents()
	suite.Require().NoError(suite.newEngine().CancelOrder(suite.ctx, toCancel))
	events := suite.tradebinTypedEvents()
	suite.Require().Len(events, 1)
	evId, ok := engineEventOrderId(events[0])
	suite.Require().True(ok)
	suite.Require().Equal(toCancel.Id, evId)
}
