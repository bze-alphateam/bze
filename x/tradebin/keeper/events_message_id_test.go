package keeper_test

import (
	"fmt"
	"strings"

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

func (suite *IntegrationTestSuite) TestEventsMessageId_CreateOrder() {
	suite.k.SetMarket(suite.ctx, market)
	// start from a non-zero counter so the id is not the default value
	suite.k.SetQueueMessageCounter(suite.ctx, 7)

	addr1 := sdk.AccAddress("addr1_______________")
	params := suite.k.GetParams(suite.ctx)
	makerFee := sdk.NewCoins(params.MarketMakerFee)
	suite.bankMock.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), addr1, types.ModuleName, makerFee).Return(nil).Times(1)
	suite.bankMock.EXPECT().SendCoinsFromModuleToModule(gomock.Any(), types.ModuleName, gomock.Any(), makerFee).Return(nil).Times(1)
	suite.bankMock.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), addr1, types.ModuleName, gomock.Any()).Return(nil).Times(1)

	suite.resetEvents()
	_, err := suite.msgServer.CreateOrder(suite.ctx, &types.MsgCreateOrder{
		Amount:    "1000000",
		Price:     "2",
		MarketId:  getMarketId(),
		OrderType: types.OrderTypeBuy,
		Creator:   addr1.String(),
	})
	suite.Require().NoError(err)

	qms := suite.k.GetAllQueueMessage(suite.ctx)
	suite.Require().Len(qms, 1)
	suite.Require().Equal(fmt.Sprintf("%024d", 7), qms[0].MessageId)

	var found []*types.OrderCreateMessageEvent
	for _, ev := range suite.tradebinTypedEvents() {
		if e, ok := ev.(*types.OrderCreateMessageEvent); ok {
			found = append(found, e)
		}
	}
	suite.Require().Len(found, 1)
	suite.Require().Equal(qms[0].MessageId, found[0].MessageId)
	suite.Require().Equal(addr1.String(), found[0].Creator)
	suite.Require().Equal(types.OrderTypeBuy, found[0].OrderType)
}

func (suite *IntegrationTestSuite) TestEventsMessageId_CancelOrder() {
	suite.k.SetMarket(suite.ctx, market)
	suite.k.SetQueueMessageCounter(suite.ctx, 3)
	order := suite.k.NewOrder(suite.ctx, types.Order{
		MarketId:  getMarketId(),
		OrderType: types.OrderTypeBuy,
		Amount:    "102000",
		Price:     "1",
		Owner:     "me",
	})

	suite.resetEvents()
	_, err := suite.msgServer.CancelOrder(suite.ctx, &types.MsgCancelOrder{
		Creator:   "me",
		MarketId:  getMarketId(),
		OrderId:   order.Id,
		OrderType: types.OrderTypeBuy,
	})
	suite.Require().NoError(err)

	qms := suite.k.GetAllQueueMessage(suite.ctx)
	suite.Require().Len(qms, 1)
	suite.Require().Equal(fmt.Sprintf("%024d", 3), qms[0].MessageId)

	var found []*types.OrderCancelMessageEvent
	for _, ev := range suite.tradebinTypedEvents() {
		if e, ok := ev.(*types.OrderCancelMessageEvent); ok {
			found = append(found, e)
		}
	}
	suite.Require().Len(found, 1)
	suite.Require().Equal(qms[0].MessageId, found[0].MessageId)
	suite.Require().Equal(order.Id, found[0].OrderId)
}

func (suite *IntegrationTestSuite) TestEventsMessageId_FillOrdersEmitsOneEventPerQueuedFill() {
	allPrices, _, addr2 := suite.msgOrderFillSetup(types.OrderTypeSell)
	engine, err := keeper.NewProcessingEngine(suite.k, suite.bankMock, suite.k.Logger())
	suite.Require().NoError(err)
	engine.ProcessQueueMessages(suite.ctx)

	// taker fee + funds capture from the user, fee forwarded to the fee collector
	suite.bankMock.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), addr2, types.ModuleName, gomock.Any()).Return(nil).Times(2)
	suite.bankMock.EXPECT().SendCoinsFromModuleToModule(gomock.Any(), types.ModuleName, gomock.Any(), gomock.Any()).Return(nil).Times(1)

	suite.resetEvents()
	_, err = suite.msgServer.FillOrders(suite.ctx, &types.MsgFillOrders{
		Creator:   addr2.String(),
		MarketId:  getMarketId(),
		OrderType: types.OrderTypeSell,
		Orders: []*types.FillOrderItem{
			{Amount: "500000", Price: allPrices[0]},
			{Amount: "500000", Price: allPrices[1]},
		},
	})
	suite.Require().NoError(err)

	qms := suite.k.GetAllQueueMessage(suite.ctx)
	suite.Require().Len(qms, 2)
	idByPrice := map[string]string{}
	for _, qm := range qms {
		idByPrice[qm.Price] = qm.MessageId
	}
	suite.Require().NotEqual(idByPrice[allPrices[0]], idByPrice[allPrices[1]])

	var found []*types.OrderCreateMessageEvent
	for _, ev := range suite.tradebinTypedEvents() {
		if e, ok := ev.(*types.OrderCreateMessageEvent); ok {
			found = append(found, e)
		}
	}
	suite.Require().Len(found, 2)
	for _, e := range found {
		suite.Require().Equal(idByPrice[e.Price], e.MessageId)
		suite.Require().Equal(addr2.String(), e.Creator)
		// the queue message stores the inverted order type: filling sells means buying
		suite.Require().Equal(types.OrderTypeBuy, e.OrderType)
		suite.Require().Equal("500000", e.Amount)
	}
}

func (suite *IntegrationTestSuite) TestEventsMessageId_EngineFillAndSaveRemainder() {
	suite.k.SetMarket(suite.ctx, market)
	engine, err := keeper.NewProcessingEngine(suite.k, suite.bankMock, suite.k.Logger())
	suite.Require().NoError(err)

	makerAddr := sdk.AccAddress("addr1_______________")
	takerAddr := sdk.AccAddress("addr2_______________")
	price := "1"
	sellAmt, err := keeper.CalculateMinAmount(price)
	suite.Require().NoError(err)
	sellAmt = sellAmt.MulRaw(2)

	// both messages land in the same block: the sell rests first, then the buy fills it and rests its remainder
	sellQm := suite.k.SetQueueMessage(suite.ctx, types.QueueMessage{
		MarketId:    getMarketId(),
		MessageType: types.OrderTypeSell,
		Amount:      sellAmt.String(),
		Price:       price,
		OrderType:   types.OrderTypeSell,
		Owner:       makerAddr.String(),
	})
	buyQm := suite.k.SetQueueMessage(suite.ctx, types.QueueMessage{
		MarketId:    getMarketId(),
		MessageType: types.OrderTypeBuy,
		Amount:      sellAmt.MulRaw(2).String(),
		Price:       price,
		OrderType:   types.OrderTypeBuy,
		Owner:       takerAddr.String(),
	})
	suite.Require().NotEqual(sellQm.MessageId, buyQm.MessageId)

	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	suite.resetEvents()
	engine.ProcessQueueMessages(suite.ctx)

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
	suite.Require().Equal(sellQm.MessageId, savedByOwner[makerAddr.String()].MessageId)
	suite.Require().Equal(buyQm.MessageId, savedByOwner[takerAddr.String()].MessageId)
	suite.Require().Equal(sellAmt.String(), savedByOwner[takerAddr.String()].Amount)

	suite.Require().Len(executed, 1)
	suite.Require().Equal(buyQm.MessageId, executed[0].MessageId)
	// id stays the resting (maker) order's id, taker stays the message owner
	suite.Require().Equal(savedByOwner[makerAddr.String()].Id, executed[0].Id)
	suite.Require().Equal(takerAddr.String(), executed[0].Taker)
	suite.Require().Equal(makerAddr.String(), executed[0].Maker)
}

func (suite *IntegrationTestSuite) TestEventsMessageId_EngineCancel() {
	suite.k.SetMarket(suite.ctx, market)
	engine, err := keeper.NewProcessingEngine(suite.k, suite.bankMock, suite.k.Logger())
	suite.Require().NoError(err)

	addr1 := sdk.AccAddress("addr1_______________")
	order := suite.k.NewOrder(suite.ctx, types.Order{
		MarketId:  getMarketId(),
		OrderType: types.OrderTypeBuy,
		Amount:    "102000",
		Price:     "1",
		Owner:     addr1.String(),
	})

	suite.k.SetQueueMessageCounter(suite.ctx, 5)
	cancelQm := suite.k.SetQueueMessage(suite.ctx, types.QueueMessage{
		MarketId:    getMarketId(),
		MessageType: types.MessageTypeCancel,
		OrderId:     order.Id,
		OrderType:   order.OrderType,
		Owner:       addr1.String(),
	})
	suite.Require().Equal(fmt.Sprintf("%024d", 5), cancelQm.MessageId)

	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, addr1, gomock.Any()).Return(nil).Times(1)

	suite.resetEvents()
	engine.ProcessQueueMessages(suite.ctx)

	var found []*types.OrderCanceledEvent
	for _, ev := range suite.tradebinTypedEvents() {
		if e, ok := ev.(*types.OrderCanceledEvent); ok {
			found = append(found, e)
		}
	}
	suite.Require().Len(found, 1)
	suite.Require().Equal(cancelQm.MessageId, found[0].MessageId)
	suite.Require().Equal(order.Id, found[0].Id)
}
