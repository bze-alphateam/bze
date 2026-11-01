package keeper_test

import (
	"fmt"
	"strings"

	"cosmossdk.io/math"
	"github.com/bze-alphateam/bze/x/tradebin/keeper"
	"github.com/bze-alphateam/bze/x/tradebin/types"
	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/gogoproto/proto"
	"go.uber.org/mock/gomock"
)

// resetEvents gives the suite context a fresh event manager so a test only sees the events it caused
func (suite *IntegrationTestSuite) resetEvents() {
	suite.ctx = suite.ctx.WithEventManager(sdk.NewEventManager())
}

// tradebinTypedEvents decodes every tradebin typed event emitted on the suite context
func (suite *IntegrationTestSuite) tradebinTypedEvents() []proto.Message {
	var result []proto.Message
	for _, ev := range suite.ctx.EventManager().Events() {
		if !strings.HasPrefix(ev.Type, "bze.tradebin.") {
			continue
		}
		msg, err := sdk.ParseTypedEvent(abci.Event(ev))
		suite.Require().NoError(err)
		result = append(result, msg)
	}

	return result
}

// historyRecordingKeeper is the real keeper with every history index the engine writes recorded
type historyRecordingKeeper struct {
	*keeper.Keeper
	indexes []string
}

func (h *historyRecordingKeeper) SetHistoryOrder(ctx sdk.Context, order types.HistoryOrder, index string) {
	h.indexes = append(h.indexes, index)
	h.Keeper.SetHistoryOrder(ctx, order, index)
}

func orderId(counter uint64) string {
	return fmt.Sprintf("%024d", counter)
}

func (suite *IntegrationTestSuite) TestEventsOrderId_CreateOrderRests() {
	suite.k.SetMarket(suite.ctx, market)
	// start from a non-zero counter so the id is not the default value
	suite.k.SetOrderCounter(suite.ctx, 7)

	addr1 := sdk.AccAddress("addr1_______________")
	params := suite.k.GetParams(suite.ctx)
	makerFee := sdk.NewCoins(params.MarketMakerFee)
	suite.bankMock.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), addr1, types.ModuleName, makerFee).Return(nil).Times(1)
	suite.bankMock.EXPECT().SendCoinsFromModuleToModule(gomock.Any(), types.ModuleName, gomock.Any(), makerFee).Return(nil).Times(1)
	suite.bankMock.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), addr1, types.ModuleName, sdk.NewCoins(sdk.NewInt64Coin(market.Quote, 2000000))).Return(nil).Times(1)

	suite.resetEvents()
	_, err := suite.msgServer.CreateOrder(suite.ctx, &types.MsgCreateOrder{
		Amount:    "1000000",
		Price:     "2",
		MarketId:  getMarketId(),
		OrderType: types.OrderTypeBuy,
		Creator:   addr1.String(),
	})
	suite.Require().NoError(err)
	suite.Require().Equal(uint64(8), suite.k.GetOrderCounter(suite.ctx))

	order, found := suite.k.GetOrder(suite.ctx, getMarketId(), types.OrderTypeBuy, orderId(7))
	suite.Require().True(found)
	suite.Require().Equal("1000000", order.Amount)

	var found2 []*types.OrderSavedEvent
	for _, ev := range suite.tradebinTypedEvents() {
		if e, ok := ev.(*types.OrderSavedEvent); ok {
			found2 = append(found2, e)
		}
	}
	suite.Require().Len(found2, 1)
	suite.Require().Equal(orderId(7), found2[0].Id)
	suite.Require().Equal(orderId(7), found2[0].OrderId)
	suite.Require().Equal(addr1.String(), found2[0].Owner)
	suite.Require().Equal(types.OrderTypeBuy, found2[0].OrderType)
	suite.Require().Equal("1000000", found2[0].Amount)
}

func (suite *IntegrationTestSuite) TestEventsOrderId_CancelOrder() {
	suite.k.SetMarket(suite.ctx, market)
	suite.k.SetOrderCounter(suite.ctx, 3)
	addr1 := sdk.AccAddress("addr1_______________")
	suite.placeOrders(suite.newEngine(), limitOrder(types.OrderTypeBuy, "102000", "1", addr1))
	order, found := suite.k.GetOrder(suite.ctx, getMarketId(), types.OrderTypeBuy, orderId(3))
	suite.Require().True(found)

	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, addr1, sdk.NewCoins(sdk.NewInt64Coin(market.Quote, 102000))).Return(nil).Times(1)

	suite.resetEvents()
	_, err := suite.msgServer.CancelOrder(suite.ctx, &types.MsgCancelOrder{
		Creator:   addr1.String(),
		MarketId:  getMarketId(),
		OrderId:   order.Id,
		OrderType: types.OrderTypeBuy,
	})
	suite.Require().NoError(err)
	// a cancel does not take an order id
	suite.Require().Equal(uint64(4), suite.k.GetOrderCounter(suite.ctx))

	var found2 []*types.OrderCanceledEvent
	for _, ev := range suite.tradebinTypedEvents() {
		if e, ok := ev.(*types.OrderCanceledEvent); ok {
			found2 = append(found2, e)
		}
	}
	suite.Require().Len(found2, 1)
	suite.Require().Equal(order.Id, found2[0].OrderId)
	suite.Require().Equal(order.Id, found2[0].Id)
	suite.Require().Equal("102000", found2[0].Amount)
	suite.Require().Equal(addr1.String(), found2[0].Owner)
}

// TestEventsOrderId_FillOrdersEntriesGetOwnIds checks that every FillOrders entry is executed as its own
// taker order with its own id, in entry order, and that the history index is {takerId}{makerId}.
func (suite *IntegrationTestSuite) TestEventsOrderId_FillOrdersEntriesGetOwnIds() {
	allPrices, addr1, addr2 := suite.msgOrderFillSetup(types.OrderTypeSell)
	// the 10 resting makers took ids 0..9
	suite.Require().Equal(uint64(10), suite.k.GetOrderCounter(suite.ctx))
	maker0, found := suite.k.GetOrder(suite.ctx, getMarketId(), types.OrderTypeSell, orderId(0))
	suite.Require().True(found)
	suite.Require().Equal(allPrices[0], maker0.Price)
	maker1, found := suite.k.GetOrder(suite.ctx, getMarketId(), types.OrderTypeSell, orderId(1))
	suite.Require().True(found)
	suite.Require().Equal(allPrices[1], maker1.Price)

	price0, _ := math.NewIntFromString(allPrices[0])
	price1, _ := math.NewIntFromString(allPrices[1])
	params := suite.k.GetParams(suite.ctx)
	takerFee := sdk.NewCoins(params.MarketTakerFee)
	// one escrow per entry, then the taker fee
	escrow0 := sdk.NewCoins(sdk.NewCoin(market.Quote, price0.MulRaw(500000)))
	escrow1 := sdk.NewCoins(sdk.NewCoin(market.Quote, price1.MulRaw(500000)))
	suite.bankMock.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), addr2, types.ModuleName, escrow0).Return(nil).Times(1)
	suite.bankMock.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), addr2, types.ModuleName, escrow1).Return(nil).Times(1)
	suite.bankMock.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), addr2, types.ModuleName, takerFee).Return(nil).Times(1)
	suite.bankMock.EXPECT().SendCoinsFromModuleToModule(gomock.Any(), types.ModuleName, gomock.Any(), takerFee).Return(nil).Times(1)
	// settlement of each entry
	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, addr1, escrow0).Return(nil).Times(1)
	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, addr1, escrow1).Return(nil).Times(1)
	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, addr2, sdk.NewCoins(sdk.NewInt64Coin(market.Base, 500000))).Return(nil).Times(2)

	suite.resetEvents()
	_, err := suite.msgServer.FillOrders(suite.ctx, &types.MsgFillOrders{
		Creator:   addr2.String(),
		MarketId:  getMarketId(),
		OrderType: types.OrderTypeSell,
		Orders: []*types.FillOrderItem{
			{Amount: "500000", Price: allPrices[0]},
			{Amount: "500000", Price: allPrices[1]},
		},
	})
	suite.Require().NoError(err)
	// one id per entry
	suite.Require().Equal(uint64(12), suite.k.GetOrderCounter(suite.ctx))

	var executed []*types.OrderExecutedEvent
	for _, ev := range suite.tradebinTypedEvents() {
		switch e := ev.(type) {
		case *types.OrderExecutedEvent:
			executed = append(executed, e)
		case *types.OrderSavedEvent:
			suite.Fail("a fill never rests on the book")
		}
	}
	suite.Require().Len(executed, 2)
	suite.Require().Equal(orderId(10), executed[0].OrderId)
	suite.Require().Equal(maker0.Id, executed[0].Id)
	suite.Require().Equal(allPrices[0], executed[0].Price)
	suite.Require().Equal(orderId(11), executed[1].OrderId)
	suite.Require().Equal(maker1.Id, executed[1].Id)
	suite.Require().Equal(allPrices[1], executed[1].Price)
	for _, e := range executed {
		suite.Require().Equal(addr2.String(), e.Taker)
		suite.Require().Equal(addr1.String(), e.Maker)
		suite.Require().Equal("500000", e.Amount)
	}

	history := suite.k.GetAllHistoryOrder(suite.ctx)
	suite.Require().Len(history, 2)
}

// TestEventsOrderId_EngineFillAndSaveRemainder places a sell that rests, then a buy that fills it and rests
// its remainder: the remainder keeps the taker id, the execution carries the taker id and the maker's id,
// and the history entry is indexed {takerId}{makerId}.
func (suite *IntegrationTestSuite) TestEventsOrderId_EngineFillAndSaveRemainder() {
	suite.k.SetMarket(suite.ctx, market)
	recorder := &historyRecordingKeeper{Keeper: suite.k}
	engine, err := keeper.NewProcessingEngine(recorder, suite.bankMock, suite.k.Logger())
	suite.Require().NoError(err)

	makerAddr := sdk.AccAddress("addr1_______________")
	takerAddr := sdk.AccAddress("addr2_______________")
	price := "1"
	sellAmt, err := keeper.CalculateMinAmount(price)
	suite.Require().NoError(err)
	sellAmt = sellAmt.MulRaw(2)
	suite.k.SetOrderCounter(suite.ctx, 41)

	suite.resetEvents()
	suite.Require().NoError(engine.PlaceOrder(suite.ctx, limitOrder(types.OrderTypeSell, sellAmt.String(), price, makerAddr)))
	makerId := orderId(41)
	suite.Require().Equal(uint64(42), suite.k.GetOrderCounter(suite.ctx))

	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, makerAddr, sdk.NewCoins(sdk.NewCoin(market.Quote, sellAmt))).Return(nil).Times(1)
	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, takerAddr, sdk.NewCoins(sdk.NewCoin(market.Base, sellAmt))).Return(nil).Times(1)

	suite.Require().NoError(engine.PlaceOrder(suite.ctx, limitOrder(types.OrderTypeBuy, sellAmt.MulRaw(2).String(), price, takerAddr)))
	takerId := orderId(42)
	suite.Require().Equal(uint64(43), suite.k.GetOrderCounter(suite.ctx))

	var executed []*types.OrderExecutedEvent
	savedByOwner := map[string]*types.OrderSavedEvent{}
	for _, ev := range suite.tradebinTypedEvents() {
		switch e := ev.(type) {
		case *types.OrderExecutedEvent:
			executed = append(executed, e)
		case *types.OrderSavedEvent:
			savedByOwner[e.Owner] = e
		}
	}

	suite.Require().Len(savedByOwner, 2)
	suite.Require().Equal(makerId, savedByOwner[makerAddr.String()].OrderId)
	suite.Require().Equal(makerId, savedByOwner[makerAddr.String()].Id)
	suite.Require().Equal(takerId, savedByOwner[takerAddr.String()].OrderId)
	suite.Require().Equal(takerId, savedByOwner[takerAddr.String()].Id)
	suite.Require().Equal(sellAmt.String(), savedByOwner[takerAddr.String()].Amount)

	remainder, found := suite.k.GetOrder(suite.ctx, getMarketId(), types.OrderTypeBuy, takerId)
	suite.Require().True(found)
	suite.Require().Equal(sellAmt.String(), remainder.Amount)
	_, found = suite.k.GetOrder(suite.ctx, getMarketId(), types.OrderTypeSell, makerId)
	suite.Require().False(found)

	suite.Require().Len(executed, 1)
	suite.Require().Equal(takerId, executed[0].OrderId)
	// id stays the resting (maker) order's id, taker stays the taker order's owner
	suite.Require().Equal(makerId, executed[0].Id)
	suite.Require().Equal(takerAddr.String(), executed[0].Taker)
	suite.Require().Equal(makerAddr.String(), executed[0].Maker)

	suite.Require().Equal([]string{takerId + makerId}, recorder.indexes)
	history := suite.k.GetAllHistoryOrder(suite.ctx)
	suite.Require().Len(history, 1)
	suite.Require().Equal(makerAddr.String(), history[0].Maker)
	suite.Require().Equal(takerAddr.String(), history[0].Taker)
	suite.Require().Equal(sellAmt.String(), history[0].Amount)
}

// TestEventsOrderId_FullyFilledTakerGetsId checks that a taker that is completely filled still takes an
// order id, which its execution carries, although it never rests on the book.
func (suite *IntegrationTestSuite) TestEventsOrderId_FullyFilledTakerGetsId() {
	suite.k.SetMarket(suite.ctx, market)
	recorder := &historyRecordingKeeper{Keeper: suite.k}
	engine, err := keeper.NewProcessingEngine(recorder, suite.bankMock, suite.k.Logger())
	suite.Require().NoError(err)

	makerAddr := sdk.AccAddress("addr1_______________")
	takerAddr := sdk.AccAddress("addr2_______________")
	price := "1"
	amt, err := keeper.CalculateMinAmount(price)
	suite.Require().NoError(err)

	suite.Require().NoError(engine.PlaceOrder(suite.ctx, limitOrder(types.OrderTypeSell, amt.String(), price, makerAddr)))
	suite.Require().NoError(engine.PlaceOrder(suite.ctx, limitOrder(types.OrderTypeSell, amt.String(), price, makerAddr)))
	suite.Require().Equal(uint64(2), suite.k.GetOrderCounter(suite.ctx))

	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, makerAddr, sdk.NewCoins(sdk.NewCoin(market.Quote, amt))).Return(nil).Times(2)
	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, takerAddr, sdk.NewCoins(sdk.NewCoin(market.Base, amt))).Return(nil).Times(2)

	// first taker with PlaceOrder, second with FillOrder: each takes the next id
	suite.resetEvents()
	suite.Require().NoError(engine.PlaceOrder(suite.ctx, limitOrder(types.OrderTypeBuy, amt.String(), price, takerAddr)))
	suite.Require().Equal(uint64(3), suite.k.GetOrderCounter(suite.ctx))
	suite.Require().NoError(engine.FillOrder(suite.ctx, limitOrder(types.OrderTypeBuy, amt.String(), price, takerAddr)))
	suite.Require().Equal(uint64(4), suite.k.GetOrderCounter(suite.ctx))

	var executed []*types.OrderExecutedEvent
	for _, ev := range suite.tradebinTypedEvents() {
		switch e := ev.(type) {
		case *types.OrderExecutedEvent:
			executed = append(executed, e)
		case *types.OrderSavedEvent, *types.OrderMessageRefundedEvent:
			suite.Failf("unexpected event", "%T", e)
		}
	}
	suite.Require().Len(executed, 2)
	suite.Require().Equal(orderId(2), executed[0].OrderId)
	suite.Require().Equal(orderId(0), executed[0].Id)
	suite.Require().Equal(orderId(3), executed[1].OrderId)
	suite.Require().Equal(orderId(1), executed[1].Id)
	suite.Require().Equal([]string{orderId(2) + orderId(0), orderId(3) + orderId(1)}, recorder.indexes)

	suite.Require().Empty(suite.k.GetAllOrder(suite.ctx))
	suite.Require().Empty(suite.k.GetAllAggregatedOrder(suite.ctx))
}

func (suite *IntegrationTestSuite) TestEventsOrderId_EngineCancel() {
	suite.k.SetMarket(suite.ctx, market)
	engine := suite.newEngine()

	addr1 := sdk.AccAddress("addr1_______________")
	suite.k.SetOrderCounter(suite.ctx, 5)
	suite.placeOrders(engine, limitOrder(types.OrderTypeBuy, "102000", "1", addr1))
	order, found := suite.k.GetOrder(suite.ctx, getMarketId(), types.OrderTypeBuy, orderId(5))
	suite.Require().True(found)

	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, addr1, sdk.NewCoins(sdk.NewInt64Coin(market.Quote, 102000))).Return(nil).Times(1)

	suite.resetEvents()
	suite.Require().NoError(engine.CancelOrder(suite.ctx, order))
	suite.Require().Equal(uint64(6), suite.k.GetOrderCounter(suite.ctx))

	var found2 []*types.OrderCanceledEvent
	for _, ev := range suite.tradebinTypedEvents() {
		if e, ok := ev.(*types.OrderCanceledEvent); ok {
			found2 = append(found2, e)
		}
	}
	suite.Require().Len(found2, 1)
	suite.Require().Equal(order.Id, found2[0].OrderId)
	suite.Require().Equal(order.Id, found2[0].Id)
	_, found = suite.k.GetOrder(suite.ctx, getMarketId(), types.OrderTypeBuy, order.Id)
	suite.Require().False(found)
	suite.Require().Empty(suite.k.GetAllAggregatedOrder(suite.ctx))
}
