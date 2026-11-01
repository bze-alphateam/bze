package keeper_test

import (
	"time"

	"github.com/bze-alphateam/bze/x/tradebin/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/gogoproto/proto"
	"go.uber.org/mock/gomock"
)

func amendedEvents(events []proto.Message) []*types.OrderAmendedEvent {
	var result []*types.OrderAmendedEvent
	for _, ev := range events {
		if e, ok := ev.(*types.OrderAmendedEvent); ok {
			result = append(result, e)
		}
	}

	return result
}

// requireOwnerPriceOrder checks the owner-price index entry of the owner at the market, side and price
func (suite *IntegrationTestSuite) requireOwnerPriceOrder(owner sdk.AccAddress, orderType, price, id string) {
	ref, found := suite.k.GetOwnerPriceOrder(suite.ctx, owner.String(), getMarketId(), orderType, price)
	suite.Require().True(found)
	suite.Require().Equal(types.OrderReference{Id: id, MarketId: getMarketId(), OrderType: orderType}, ref)
}

func (suite *IntegrationTestSuite) requireNoOwnerPriceOrder(owner sdk.AccAddress, orderType, price string) {
	_, found := suite.k.GetOwnerPriceOrder(suite.ctx, owner.String(), getMarketId(), orderType, price)
	suite.Require().False(found)
}

// seedRestingOrder writes a resting order and its aggregate straight to the store. It is used for a
// book where both sides rest at one price, which the engine never produces on its own: an order that
// stops while the opposite side still rests at its price is refunded, not rested.
func (suite *IntegrationTestSuite) seedRestingOrder(order types.Order) {
	suite.k.SaveOrder(suite.ctx, order)
	suite.k.SetAggregatedOrder(suite.ctx, types.AggregatedOrder{
		MarketId:  order.MarketId,
		OrderType: order.OrderType,
		Amount:    order.Amount,
		Price:     order.Price,
	})
}

func (suite *IntegrationTestSuite) TestEngineMerge_SameOwnerSamePriceAddsToRestingOrder() {
	suite.k.SetMarket(suite.ctx, market)
	engine := suite.newEngine()
	owner := sdk.AccAddress("addr1_______________")

	placedAt := time.Unix(1_700_000_000, 0)
	suite.ctx = suite.ctx.WithBlockTime(placedAt)
	suite.placeOrders(engine, limitOrder(types.OrderTypeSell, "10", "1", owner))
	suite.Require().Equal(uint64(1), suite.k.GetOrderCounter(suite.ctx))

	// a later block, the same price written with all its decimals
	suite.ctx = suite.ctx.WithBlockTime(placedAt.Add(time.Hour))
	suite.resetEvents()
	suite.placeOrders(engine, limitOrder(types.OrderTypeSell, "6", "1.000000000000000000", owner))

	// no new id, the resting order grew and kept its creation time
	suite.Require().Equal(uint64(1), suite.k.GetOrderCounter(suite.ctx))
	orders := suite.k.GetAllOrder(suite.ctx)
	suite.Require().Len(orders, 1)
	suite.Require().Equal(orderId(0), orders[0].Id)
	suite.Require().Equal("16", orders[0].Amount)
	suite.Require().Equal("1", orders[0].Price)
	suite.Require().Equal(placedAt.Unix(), orders[0].CreatedAt)
	suite.checkAggregatedOrder(getMarketId(), types.OrderTypeSell, "1", "16")
	suite.requireOwnerPriceOrder(owner, types.OrderTypeSell, "1", orderId(0))

	events := suite.tradebinTypedEvents()
	suite.Require().Empty(savedEvents(events))
	suite.Require().Equal([]*types.OrderAmendedEvent{{
		Id:          orderId(0),
		MarketId:    getMarketId(),
		OrderType:   types.OrderTypeSell,
		Price:       "1",
		Owner:       owner.String(),
		AddedAmount: "6",
		Amount:      "16",
	}}, amendedEvents(events))
}

func (suite *IntegrationTestSuite) TestEngineMerge_OtherOwnerPriceOrSideGetsNewOrder() {
	suite.k.SetMarket(suite.ctx, market)
	engine := suite.newEngine()
	owner := sdk.AccAddress("addr1_______________")
	other := sdk.AccAddress("addr2_______________")

	suite.placeOrders(engine, limitOrder(types.OrderTypeSell, "10", "1", owner))

	suite.resetEvents()
	suite.placeOrders(engine,
		// another owner at the same price
		limitOrder(types.OrderTypeSell, "10", "1", other),
		// the same owner at another price
		limitOrder(types.OrderTypeSell, "10", "2", owner),
		// the same owner on the other side
		limitOrder(types.OrderTypeBuy, "10", "0.5", owner),
	)

	suite.Require().Equal(uint64(4), suite.k.GetOrderCounter(suite.ctx))
	suite.Require().Len(suite.k.GetAllOrder(suite.ctx), 4)
	events := suite.tradebinTypedEvents()
	suite.Require().Empty(amendedEvents(events))
	saved := savedEvents(events)
	suite.Require().Len(saved, 3)
	for i, ev := range saved {
		suite.Require().Equal(orderId(uint64(i+1)), ev.Id)
		suite.Require().Equal("10", ev.Amount)
	}

	suite.checkAggregatedOrder(getMarketId(), types.OrderTypeSell, "1", "20")
	suite.requireOwnerPriceOrder(owner, types.OrderTypeSell, "1", orderId(0))
	suite.requireOwnerPriceOrder(other, types.OrderTypeSell, "1", orderId(1))
	suite.requireOwnerPriceOrder(owner, types.OrderTypeSell, "2", orderId(2))
	suite.requireOwnerPriceOrder(owner, types.OrderTypeBuy, "0.5", orderId(3))
	// the index is per side: the owner's sell at 1 is not a buy at 1
	suite.requireNoOwnerPriceOrder(owner, types.OrderTypeBuy, "1")
}

// The top-up is matched first, under the id of the resting order, and its remainder is merged.
func (suite *IntegrationTestSuite) TestEngineMerge_TopUpMatchesFirstThenMerges() {
	suite.k.SetMarket(suite.ctx, market)
	engine := suite.newEngine()
	owner := sdk.AccAddress("addr1_______________")
	seller := sdk.AccAddress("addr2_______________")

	suite.placeOrders(engine, limitOrder(types.OrderTypeBuy, "10", "1", owner))
	sell := types.Order{Id: orderId(50), MarketId: getMarketId(), OrderType: types.OrderTypeSell, Amount: "4", Price: "1", Owner: seller.String()}
	suite.seedRestingOrder(sell)

	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, seller, sdk.NewCoins(sdk.NewInt64Coin(market.Quote, 4))).Return(nil).Times(1)
	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, owner, sdk.NewCoins(sdk.NewInt64Coin(market.Base, 4))).Return(nil).Times(1)

	suite.resetEvents()
	suite.placeOrders(engine, limitOrder(types.OrderTypeBuy, "7", "1", owner))

	suite.Require().Equal(uint64(1), suite.k.GetOrderCounter(suite.ctx))
	orders := suite.k.GetAllOrder(suite.ctx)
	suite.Require().Len(orders, 1)
	suite.Require().Equal(orderId(0), orders[0].Id)
	suite.Require().Equal("13", orders[0].Amount)
	suite.checkAggregatedOrder(getMarketId(), types.OrderTypeBuy, "1", "13")
	_, found := suite.k.GetAggregatedOrder(suite.ctx, getMarketId(), types.OrderTypeSell, "1")
	suite.Require().False(found)

	events := suite.tradebinTypedEvents()
	executed := executedEvents(events)
	suite.Require().Len(executed, 1)
	suite.Require().Equal(sell.Id, executed[0].Id)
	suite.Require().Equal(orderId(0), executed[0].OrderId)
	suite.Require().Equal("4", executed[0].Amount)
	amended := amendedEvents(events)
	suite.Require().Len(amended, 1)
	suite.Require().Equal("3", amended[0].AddedAmount)
	suite.Require().Equal("13", amended[0].Amount)
	suite.Require().Empty(savedEvents(events))
}

// A top-up whose remainder is below the minimum amount is refunded like a fresh order, and the resting
// order is untouched.
func (suite *IntegrationTestSuite) TestEngineMerge_TopUpRemainderBelowMinIsRefunded() {
	suite.k.SetMarket(suite.ctx, market)
	engine := suite.newEngine()
	owner := sdk.AccAddress("addr1_______________")
	seller := sdk.AccAddress("addr2_______________")

	suite.placeOrders(engine, limitOrder(types.OrderTypeBuy, "10", "1", owner))
	suite.seedRestingOrder(types.Order{Id: orderId(50), MarketId: getMarketId(), OrderType: types.OrderTypeSell, Amount: "4", Price: "1", Owner: seller.String()})

	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, seller, sdk.NewCoins(sdk.NewInt64Coin(market.Quote, 4))).Return(nil).Times(1)
	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, owner, sdk.NewCoins(sdk.NewInt64Coin(market.Base, 4))).Return(nil).Times(1)
	// the remainder of 1 (min amount at price 1 is 2) goes back as quote
	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, owner, sdk.NewCoins(sdk.NewInt64Coin(market.Quote, 1))).Return(nil).Times(1)

	suite.resetEvents()
	suite.placeOrders(engine, limitOrder(types.OrderTypeBuy, "5", "1", owner))

	resting, found := suite.k.GetOrder(suite.ctx, getMarketId(), types.OrderTypeBuy, orderId(0))
	suite.Require().True(found)
	suite.Require().Equal("10", resting.Amount)
	suite.checkAggregatedOrder(getMarketId(), types.OrderTypeBuy, "1", "10")
	suite.requireOwnerPriceOrder(owner, types.OrderTypeBuy, "1", orderId(0))

	events := suite.tradebinTypedEvents()
	suite.Require().Empty(amendedEvents(events))
	refunds := refundedEvents(events)
	suite.Require().Len(refunds, 1)
	suite.Require().Equal(orderId(0), refunds[0].OrderId)
	suite.Require().Equal(types.RefundReasonBelowMinAmount, refunds[0].Reason)
	suite.Require().Equal("1", refunds[0].Amount)
}

func (suite *IntegrationTestSuite) TestEngineMerge_CancelRefundsTheWholeMergedOrder() {
	suite.k.SetMarket(suite.ctx, market)
	engine := suite.newEngine()
	owner := sdk.AccAddress("addr1_______________")

	suite.placeOrders(engine,
		limitOrder(types.OrderTypeSell, "10", "1", owner),
		limitOrder(types.OrderTypeSell, "6", "1", owner),
	)
	merged, found := suite.k.GetOrder(suite.ctx, getMarketId(), types.OrderTypeSell, orderId(0))
	suite.Require().True(found)

	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, owner, sdk.NewCoins(sdk.NewInt64Coin(market.Base, 16))).Return(nil).Times(1)
	suite.Require().NoError(engine.CancelOrder(suite.ctx, merged))

	suite.Require().Empty(suite.k.GetAllOrder(suite.ctx))
	_, found = suite.k.GetAggregatedOrder(suite.ctx, getMarketId(), types.OrderTypeSell, "1")
	suite.Require().False(found)
	suite.requireNoOwnerPriceOrder(owner, types.OrderTypeSell, "1")
}

// A partial fill keeps the index entry, the fill that empties the merged order removes it, and the
// owner's next order at that price is a new order.
func (suite *IntegrationTestSuite) TestEngineMerge_FillThatEmptiesTheMergedOrderRemovesTheIndex() {
	suite.k.SetMarket(suite.ctx, market)
	engine := suite.newEngine()
	owner := sdk.AccAddress("addr1_______________")
	taker := sdk.AccAddress("addr2_______________")

	suite.placeOrders(engine,
		limitOrder(types.OrderTypeSell, "10", "1", owner),
		limitOrder(types.OrderTypeSell, "6", "1", owner),
	)

	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, owner, sdk.NewCoins(sdk.NewInt64Coin(market.Quote, 5))).Return(nil).Times(1)
	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, taker, sdk.NewCoins(sdk.NewInt64Coin(market.Base, 5))).Return(nil).Times(1)
	suite.Require().NoError(engine.FillOrder(suite.ctx, limitOrder(types.OrderTypeBuy, "5", "1", taker)))
	suite.requireOwnerPriceOrder(owner, types.OrderTypeSell, "1", orderId(0))

	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, owner, sdk.NewCoins(sdk.NewInt64Coin(market.Quote, 11))).Return(nil).Times(1)
	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, taker, sdk.NewCoins(sdk.NewInt64Coin(market.Base, 11))).Return(nil).Times(1)
	suite.Require().NoError(engine.FillOrder(suite.ctx, limitOrder(types.OrderTypeBuy, "11", "1", taker)))
	suite.Require().Empty(suite.k.GetAllOrder(suite.ctx))
	suite.requireNoOwnerPriceOrder(owner, types.OrderTypeSell, "1")

	counter := suite.k.GetOrderCounter(suite.ctx)
	suite.resetEvents()
	suite.placeOrders(engine, limitOrder(types.OrderTypeSell, "3", "1", owner))
	suite.Require().Equal(counter+1, suite.k.GetOrderCounter(suite.ctx))
	saved := savedEvents(suite.tradebinTypedEvents())
	suite.Require().Len(saved, 1)
	suite.Require().Equal(orderId(counter), saved[0].Id)
	suite.requireOwnerPriceOrder(owner, types.OrderTypeSell, "1", orderId(counter))
}

// A top-up keeps the place of the resting order in the book: it is filled before later makers.
func (suite *IntegrationTestSuite) TestEngineMerge_TopUpKeepsFifoPosition() {
	suite.k.SetMarket(suite.ctx, market)
	engine := suite.newEngine()
	first := sdk.AccAddress("addr1_______________")
	second := sdk.AccAddress("addr2_______________")
	taker := sdk.AccAddress("addr3_______________")

	suite.placeOrders(engine,
		limitOrder(types.OrderTypeSell, "10", "1", first),
		limitOrder(types.OrderTypeSell, "10", "1", second),
		limitOrder(types.OrderTypeSell, "5", "1", first),
	)

	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, first, sdk.NewCoins(sdk.NewInt64Coin(market.Quote, 15))).Return(nil).Times(1)
	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, second, sdk.NewCoins(sdk.NewInt64Coin(market.Quote, 5))).Return(nil).Times(1)
	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, taker, sdk.NewCoins(sdk.NewInt64Coin(market.Base, 15))).Return(nil).Times(1)
	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, taker, sdk.NewCoins(sdk.NewInt64Coin(market.Base, 5))).Return(nil).Times(1)

	suite.resetEvents()
	suite.Require().NoError(engine.FillOrder(suite.ctx, limitOrder(types.OrderTypeBuy, "20", "1", taker)))

	executed := executedEvents(suite.tradebinTypedEvents())
	suite.Require().Len(executed, 2)
	suite.Require().Equal(orderId(0), executed[0].Id)
	suite.Require().Equal("15", executed[0].Amount)
	suite.Require().Equal(orderId(1), executed[1].Id)
	suite.Require().Equal("5", executed[1].Amount)

	rest, found := suite.k.GetOrder(suite.ctx, getMarketId(), types.OrderTypeSell, orderId(1))
	suite.Require().True(found)
	suite.Require().Equal("5", rest.Amount)
	suite.checkAggregatedOrder(getMarketId(), types.OrderTypeSell, "1", "5")
}

// An owner-price entry that points at a missing order is refused.
func (suite *IntegrationTestSuite) TestEngineMerge_IndexedOrderMissingReturnsError() {
	engine, pk, _ := suite.mockedEngine()
	owner := sdk.AccAddress("addr1_______________")
	order := limitOrder(types.OrderTypeBuy, "100", "1", owner)
	ref := types.OrderReference{Id: orderId(3), MarketId: getMarketId(), OrderType: types.OrderTypeBuy}

	gomock.InOrder(
		pk.EXPECT().GetOwnerPriceOrder(gomock.Any(), owner.String(), getMarketId(), types.OrderTypeBuy, "1").Return(ref, true).Times(1),
		pk.EXPECT().GetOrder(gomock.Any(), getMarketId(), types.OrderTypeBuy, orderId(3)).Return(types.Order{}, false).Times(1),
	)

	suite.resetEvents()
	err := engine.PlaceOrder(suite.ctx, order)
	suite.Require().ErrorContains(err, "not found")
	suite.requireNoEngineEvents()
}

// While an owner rests an order at a price, the opposite side at that price is empty, so a top-up
// never matches and is always merged whole. Checked after every way an opposite order can stop at
// that price: a partial fill of the resting order, the fill cap, and a resting order skipped because
// filling it would leave it below the minimum amount.
func (suite *IntegrationTestSuite) TestEngineMerge_TopUpNeverMatches() {
	const price = "0.2" // min amount 10
	owner := sdk.AccAddress("addr1_______________")
	other := sdk.AccAddress("addr2_______________")
	seller := sdk.AccAddress("addr3_______________")

	for _, tc := range []struct {
		name     string
		maxFills uint64
		resting  []types.Order
		sell     string
		reason   string
	}{
		{"a smaller sell partly fills the resting order", 10, []types.Order{limitOrder(types.OrderTypeBuy, "50", price, owner)}, "20", ""},
		{"the fill cap stops the sell", 1, []types.Order{limitOrder(types.OrderTypeBuy, "50", price, other), limitOrder(types.OrderTypeBuy, "50", price, owner)}, "80", types.RefundReasonMaxFillsReached},
		{"the sell skips the resting order", 10, []types.Order{limitOrder(types.OrderTypeBuy, "19", price, owner)}, "15", types.RefundReasonBelowMinAmount},
	} {
		suite.Run(tc.name, func() {
			suite.SetupTest()
			suite.k.SetMarket(suite.ctx, market)
			suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
			engine := suite.newEngineWithMaxFills(tc.maxFills)
			suite.placeOrders(engine, tc.resting...)

			suite.resetEvents()
			suite.placeOrders(engine, limitOrder(types.OrderTypeSell, tc.sell, price, seller))
			refunds := refundedEvents(suite.tradebinTypedEvents())
			if tc.reason == "" {
				suite.Require().Empty(refunds)
			} else {
				suite.Require().Len(refunds, 1)
				suite.Require().Equal(tc.reason, refunds[0].Reason)
			}
			_, found := suite.k.GetAggregatedOrder(suite.ctx, getMarketId(), types.OrderTypeSell, price)
			suite.Require().False(found, "the sell does not rest at the price of the resting buys")

			suite.resetEvents()
			suite.placeOrders(engine, limitOrder(types.OrderTypeBuy, "10", price, owner))
			events := suite.tradebinTypedEvents()
			suite.Require().Len(events, 1, "the top-up only amends: %v", events)
			amended := amendedEvents(events)
			suite.Require().Len(amended, 1)
			suite.Require().Equal("10", amended[0].AddedAmount)
		})
	}
}
