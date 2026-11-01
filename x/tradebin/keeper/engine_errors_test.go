package keeper_test

import (
	"errors"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	"github.com/bze-alphateam/bze/x/tradebin/keeper"
	"github.com/bze-alphateam/bze/x/tradebin/testutil"
	"github.com/bze-alphateam/bze/x/tradebin/types"
	v2types "github.com/bze-alphateam/bze/x/tradebin/v2types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"go.uber.org/mock/gomock"
)

// mockedEngine builds the engine on a mocked ProcessingKeeper and bank. The controller is bound to the
// current test and finished at its cleanup, so a missing expected call fails the test as well as an
// unexpected one.
func (suite *IntegrationTestSuite) mockedEngine() (*keeper.ProcessingEngine, *testutil.MockProcessingKeeper, *testutil.MockEngineBankKeeper) {
	ctrl := gomock.NewController(suite.T())
	pk := testutil.NewMockProcessingKeeper(ctrl)
	bank := testutil.NewMockEngineBankKeeper(ctrl)
	engine, err := keeper.NewProcessingEngine(pk, bank, v2types.DefaultMaxFillsPerOrder, log.NewNopLogger())
	suite.Require().NoError(err)

	return engine, pk, bank
}

// iterateRefs makes a mocked IteratePriceOrders walk the given references in order
func iterateRefs(refs ...types.OrderReference) func(sdk.Context, string, string, string, func(types.OrderReference) bool) {
	return func(_ sdk.Context, _, _, _ string, fn func(types.OrderReference) bool) {
		for _, ref := range refs {
			if fn(ref) {
				return
			}
		}
	}
}

func withId(order types.Order, id string) types.Order {
	order.Id = id
	return order
}

// requireNoEngineEvents checks that nothing was emitted on the suite context since resetEvents
func (suite *IntegrationTestSuite) requireNoEngineEvents() {
	suite.Require().Empty(suite.tradebinTypedEvents())
}

func (suite *IntegrationTestSuite) TestEngine_PlaceOrder_CorruptOppositeAggregateReturnsError() {
	engine, pk, _ := suite.mockedEngine()
	taker := sdk.AccAddress("addr2_______________")
	order := limitOrder(types.OrderTypeBuy, "100", "1", taker)
	placed := withId(order, orderId(0))

	gomock.InOrder(
		pk.EXPECT().GetOwnerPriceOrder(gomock.Any(), taker.String(), getMarketId(), types.OrderTypeBuy, "1").Return(types.OrderReference{}, false).Times(1),
		pk.EXPECT().AssignOrderId(gomock.Any(), order).Return(placed).Times(1),
		pk.EXPECT().GetAggregatedOrder(gomock.Any(), getMarketId(), types.OrderTypeSell, "1").
			Return(types.AggregatedOrder{MarketId: getMarketId(), OrderType: types.OrderTypeSell, Price: "1", Amount: "not-a-number"}, true).Times(1),
		pk.EXPECT().GetMarketById(gomock.Any(), getMarketId()).Return(market, true).Times(1),
	)

	suite.resetEvents()
	err := engine.PlaceOrder(suite.ctx, order)
	suite.Require().ErrorContains(err, "could not convert agg amount")
	suite.requireNoEngineEvents()
}

// With no opposite side the order rests, and adding it to its own side's corrupt aggregate fails.
func (suite *IntegrationTestSuite) TestEngine_PlaceOrder_CorruptOwnSideAggregateReturnsError() {
	engine, pk, _ := suite.mockedEngine()
	maker := sdk.AccAddress("addr1_______________")
	order := limitOrder(types.OrderTypeBuy, "100", "1", maker)
	placed := withId(order, orderId(0))

	gomock.InOrder(
		pk.EXPECT().GetOwnerPriceOrder(gomock.Any(), maker.String(), getMarketId(), types.OrderTypeBuy, "1").Return(types.OrderReference{}, false).Times(1),
		pk.EXPECT().AssignOrderId(gomock.Any(), order).Return(placed).Times(1),
		pk.EXPECT().GetAggregatedOrder(gomock.Any(), getMarketId(), types.OrderTypeSell, "1").Return(types.AggregatedOrder{}, false).Times(1),
		pk.EXPECT().GetOrder(gomock.Any(), getMarketId(), types.OrderTypeBuy, placed.Id).Return(types.Order{}, false).Times(1),
		pk.EXPECT().GetOwnerPriceOrder(gomock.Any(), maker.String(), getMarketId(), types.OrderTypeBuy, "1").Return(types.OrderReference{}, false).Times(1),
		pk.EXPECT().SaveOrder(gomock.Any(), placed).Return(placed).Times(1),
		pk.EXPECT().GetAggregatedOrder(gomock.Any(), getMarketId(), types.OrderTypeBuy, "1").
			Return(types.AggregatedOrder{MarketId: getMarketId(), OrderType: types.OrderTypeBuy, Price: "1", Amount: "not-a-number"}, true).Times(1),
	)

	err := engine.PlaceOrder(suite.ctx, order)
	suite.Require().ErrorContains(err, "could not convert agg amount")
}

func (suite *IntegrationTestSuite) TestEngine_PlaceOrder_CorruptOrderAmountReturnsError() {
	engine, pk, _ := suite.mockedEngine()
	taker := sdk.AccAddress("addr2_______________")
	order := limitOrder(types.OrderTypeBuy, "not-a-number", "1", taker)

	pk.EXPECT().GetOwnerPriceOrder(gomock.Any(), taker.String(), getMarketId(), types.OrderTypeBuy, "1").Return(types.OrderReference{}, false).Times(1)
	pk.EXPECT().AssignOrderId(gomock.Any(), order).Return(withId(order, orderId(0))).Times(1)

	err := engine.PlaceOrder(suite.ctx, order)
	suite.Require().ErrorContains(err, "could not convert order amount")
}

func (suite *IntegrationTestSuite) TestEngine_FillOrder_CorruptMakerOrderAmountReturnsError() {
	engine, pk, _ := suite.mockedEngine()
	maker := sdk.AccAddress("addr1_______________")
	taker := sdk.AccAddress("addr2_______________")
	order := limitOrder(types.OrderTypeBuy, "100", "1", taker)
	ref := types.OrderReference{Id: orderId(0), MarketId: getMarketId(), OrderType: types.OrderTypeSell}
	corruptMaker := withId(limitOrder(types.OrderTypeSell, "not-a-number", "1", maker), orderId(0))

	gomock.InOrder(
		pk.EXPECT().AssignOrderId(gomock.Any(), order).Return(withId(order, orderId(1))).Times(1),
		pk.EXPECT().GetMarketById(gomock.Any(), getMarketId()).Return(market, true).Times(1),
		pk.EXPECT().GetAggregatedOrder(gomock.Any(), getMarketId(), types.OrderTypeSell, "1").
			Return(types.AggregatedOrder{MarketId: getMarketId(), OrderType: types.OrderTypeSell, Price: "1", Amount: "100"}, true).Times(1),
		pk.EXPECT().IteratePriceOrders(gomock.Any(), getMarketId(), types.OrderTypeSell, "1", gomock.Any()).DoAndReturn(iterateRefs(ref)).Times(1),
		pk.EXPECT().GetOrder(gomock.Any(), getMarketId(), types.OrderTypeSell, orderId(0)).Return(corruptMaker, true).Times(1),
	)

	suite.resetEvents()
	err := engine.FillOrder(suite.ctx, order)
	suite.Require().ErrorContains(err, "could not convert order to fill amount")
	suite.requireNoEngineEvents()
}

func (suite *IntegrationTestSuite) TestEngine_CancelOrder_CorruptAggregateReturnsError() {
	engine, pk, bank := suite.mockedEngine()
	owner := sdk.AccAddress("addr1_______________")
	order := withId(limitOrder(types.OrderTypeBuy, "100", "1", owner), orderId(0))
	refund := sdk.NewInt64Coin(market.Quote, 100)

	gomock.InOrder(
		pk.EXPECT().GetOwnerPriceOrder(gomock.Any(), owner.String(), getMarketId(), types.OrderTypeBuy, "1").Return(types.OrderReference{Id: order.Id, MarketId: getMarketId(), OrderType: types.OrderTypeBuy}, true).Times(1),
		pk.EXPECT().RemoveOrder(gomock.Any(), order).Times(1),
		pk.EXPECT().GetMarketById(gomock.Any(), getMarketId()).Return(market, true).Times(1),
		pk.EXPECT().GetOrderCoinsWithDust(gomock.Any(), gomock.Any()).Return(types.OrderCoins{Coin: refund, Dust: math.LegacyZeroDec()}, nil).Times(1),
		bank.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, owner, sdk.NewCoins(refund)).Return(nil).Times(1),
		pk.EXPECT().GetAggregatedOrder(gomock.Any(), getMarketId(), types.OrderTypeBuy, "1").
			Return(types.AggregatedOrder{MarketId: getMarketId(), OrderType: types.OrderTypeBuy, Price: "1", Amount: "not-a-number"}, true).Times(1),
	)
	pk.EXPECT().StoreProcessedUserDust(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

	suite.resetEvents()
	err := engine.CancelOrder(suite.ctx, order)
	suite.Require().ErrorContains(err, "could not convert agg amount")
	suite.requireNoEngineEvents()
}

func (suite *IntegrationTestSuite) TestEngine_CancelOrder_CorruptOrderAmountReturnsError() {
	engine, pk, _ := suite.mockedEngine()
	owner := sdk.AccAddress("addr1_______________")
	order := withId(limitOrder(types.OrderTypeBuy, "not-a-number", "1", owner), orderId(0))

	gomock.InOrder(
		pk.EXPECT().GetOwnerPriceOrder(gomock.Any(), owner.String(), getMarketId(), types.OrderTypeBuy, "1").Return(types.OrderReference{Id: order.Id, MarketId: getMarketId(), OrderType: types.OrderTypeBuy}, true).Times(1),
		pk.EXPECT().RemoveOrder(gomock.Any(), order).Times(1),
		pk.EXPECT().GetMarketById(gomock.Any(), getMarketId()).Return(market, true).Times(1),
	)

	err := engine.CancelOrder(suite.ctx, order)
	suite.Require().ErrorContains(err, "could not convert order amount")
}

func (suite *IntegrationTestSuite) TestEngine_FillOrder_RefundBankErrorReturnsError() {
	engine, pk, bank := suite.mockedEngine()
	taker := sdk.AccAddress("addr2_______________")
	order := limitOrder(types.OrderTypeBuy, "5000", "2", taker)
	refund := sdk.NewInt64Coin(market.Quote, 10000)

	gomock.InOrder(
		pk.EXPECT().AssignOrderId(gomock.Any(), order).Return(withId(order, orderId(0))).Times(1),
		pk.EXPECT().GetMarketById(gomock.Any(), getMarketId()).Return(market, true).Times(1),
		pk.EXPECT().GetAggregatedOrder(gomock.Any(), getMarketId(), types.OrderTypeSell, "2").Return(types.AggregatedOrder{}, false).Times(1),
		pk.EXPECT().GetOrderCoinsWithDust(gomock.Any(), gomock.Any()).Return(types.OrderCoins{Coin: refund, Dust: math.LegacyZeroDec()}, nil).Times(1),
		bank.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, taker, sdk.NewCoins(refund)).Return(errors.New("bank refund failure")).Times(1),
	)
	pk.EXPECT().StoreProcessedUserDust(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

	suite.resetEvents()
	err := engine.FillOrder(suite.ctx, order)
	suite.Require().ErrorContains(err, "bank refund failure")
	suite.Require().Empty(refundedEvents(suite.tradebinTypedEvents()))
}

func (suite *IntegrationTestSuite) TestEngine_CancelOrder_RefundBankErrorReturnsError() {
	engine, pk, bank := suite.mockedEngine()
	owner := sdk.AccAddress("addr1_______________")
	order := withId(limitOrder(types.OrderTypeBuy, "100", "1", owner), orderId(0))
	refund := sdk.NewInt64Coin(market.Quote, 100)

	gomock.InOrder(
		pk.EXPECT().GetOwnerPriceOrder(gomock.Any(), owner.String(), getMarketId(), types.OrderTypeBuy, "1").Return(types.OrderReference{Id: order.Id, MarketId: getMarketId(), OrderType: types.OrderTypeBuy}, true).Times(1),
		pk.EXPECT().RemoveOrder(gomock.Any(), order).Times(1),
		pk.EXPECT().GetMarketById(gomock.Any(), getMarketId()).Return(market, true).Times(1),
		pk.EXPECT().GetOrderCoinsWithDust(gomock.Any(), gomock.Any()).Return(types.OrderCoins{Coin: refund, Dust: math.LegacyZeroDec()}, nil).Times(1),
		bank.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, owner, sdk.NewCoins(refund)).Return(errors.New("bank refund failure")).Times(1),
	)
	pk.EXPECT().GetAggregatedOrder(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	pk.EXPECT().StoreProcessedUserDust(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

	suite.resetEvents()
	err := engine.CancelOrder(suite.ctx, order)
	suite.Require().ErrorContains(err, "bank refund failure")
	suite.requireNoEngineEvents()
}

// TestEngine_MidFillBankErrorRevertsTx fills two resting makers in one FillOrders entry and makes the
// second maker's settlement fail. The message returns the error and, run on a cache context like a
// transaction, leaves the committed state exactly as it was: nothing in the engine swallows the error.
func (suite *IntegrationTestSuite) TestEngine_MidFillBankErrorRevertsTx() {
	suite.k.SetMarket(suite.ctx, market)
	maker1 := sdk.AccAddress("addr1_______________")
	maker2 := sdk.AccAddress("addr3_______________")
	taker := sdk.AccAddress("addr2_______________")
	price := "1"
	makerAmount := int64(1_000_000)

	suite.placeOrders(suite.newEngine(),
		limitOrder(types.OrderTypeSell, "1000000", price, maker1),
		limitOrder(types.OrderTypeSell, "1000000", price, maker2),
	)
	order1, found := suite.k.GetOrder(suite.ctx, getMarketId(), types.OrderTypeSell, orderId(0))
	suite.Require().True(found)
	order2, found := suite.k.GetOrder(suite.ctx, getMarketId(), types.OrderTypeSell, orderId(1))
	suite.Require().True(found)

	type send struct {
		to    sdk.AccAddress
		coins sdk.Coins
	}
	var sends []send
	record := func(err error) func(_ sdk.Context, _ string, to sdk.AccAddress, coins sdk.Coins) error {
		return func(_ sdk.Context, _ string, to sdk.AccAddress, coins sdk.Coins) error {
			sends = append(sends, send{to: to, coins: coins})
			return err
		}
	}
	makerCoins := sdk.NewCoins(sdk.NewInt64Coin(market.Quote, makerAmount))
	takerCoins := sdk.NewCoins(sdk.NewInt64Coin(market.Base, makerAmount))
	suite.bankMock.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), taker, types.ModuleName, sdk.NewCoins(sdk.NewInt64Coin(market.Quote, 2*makerAmount))).Return(nil).Times(1)
	gomock.InOrder(
		suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, maker1, makerCoins).DoAndReturn(record(nil)).Times(1),
		suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, taker, takerCoins).DoAndReturn(record(nil)).Times(1),
		suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, maker2, makerCoins).DoAndReturn(record(errors.New("maker settlement failure"))).Times(1),
	)

	cacheCtx, _ := suite.ctx.CacheContext()
	_, err := suite.msgServer.FillOrders(cacheCtx, &types.MsgFillOrders{
		Creator:   taker.String(),
		MarketId:  getMarketId(),
		OrderType: types.OrderTypeSell,
		Orders:    []*types.FillOrderItem{{Amount: "2000000", Price: price}},
	})
	suite.Require().ErrorContains(err, "maker settlement failure")
	suite.Require().Equal([]send{
		{to: maker1, coins: makerCoins},
		{to: taker, coins: takerCoins},
		{to: maker2, coins: makerCoins},
	}, sends)

	// inside the failed transaction the first maker was already settled and removed
	_, found = suite.k.GetOrder(cacheCtx, getMarketId(), types.OrderTypeSell, order1.Id)
	suite.Require().False(found)

	// the committed state is untouched
	stored1, found := suite.k.GetOrder(suite.ctx, getMarketId(), types.OrderTypeSell, order1.Id)
	suite.Require().True(found)
	suite.Require().Equal(order1, stored1)
	stored2, found := suite.k.GetOrder(suite.ctx, getMarketId(), types.OrderTypeSell, order2.Id)
	suite.Require().True(found)
	suite.Require().Equal(order2, stored2)
	suite.Require().Len(suite.k.GetAllOrder(suite.ctx), 2)
	suite.checkAggregatedOrder(getMarketId(), types.OrderTypeSell, price, "2000000")
	suite.Require().Empty(suite.k.GetAllHistoryOrder(suite.ctx))
	suite.Require().Equal(uint64(2), suite.k.GetOrderCounter(suite.ctx))
	_, found = suite.k.GetUserDust(suite.ctx, taker.String(), market.Quote)
	suite.Require().False(found)
}
