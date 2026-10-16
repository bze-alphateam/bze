package keeper_test

import (
	"errors"

	"cosmossdk.io/math"
	"github.com/bze-alphateam/bze/x/tradebin/keeper"
	"github.com/bze-alphateam/bze/x/tradebin/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/gogoproto/proto"
	"go.uber.org/mock/gomock"
)

// processBlock runs the order book engine once on the suite context with a fresh event manager
func (suite *IntegrationTestSuite) processBlock() []proto.Message {
	engine, err := keeper.NewProcessingEngine(suite.k, suite.bankMock, suite.k.Logger())
	suite.Require().NoError(err)
	suite.resetEvents()
	engine.ProcessQueueMessages(suite.ctx)

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

func failedEvents(events []proto.Message) []*types.OrderMessageFailedEvent {
	var result []*types.OrderMessageFailedEvent
	for _, ev := range events {
		if e, ok := ev.(*types.OrderMessageFailedEvent); ok {
			result = append(result, e)
		}
	}

	return result
}

// engineEventMessageId returns the queue message id carried by an order book engine event
func engineEventMessageId(ev proto.Message) (string, bool) {
	switch e := ev.(type) {
	case *types.OrderExecutedEvent:
		return e.MessageId, true
	case *types.OrderSavedEvent:
		return e.MessageId, true
	case *types.OrderCanceledEvent:
		return e.MessageId, true
	case *types.OrderMessageRefundedEvent:
		return e.MessageId, true
	case *types.OrderCancelSkippedEvent:
		return e.MessageId, true
	case *types.OrderMessageFailedEvent:
		return e.MessageId, true
	}

	return "", false
}

func (suite *IntegrationTestSuite) restingSell(owner sdk.AccAddress, amount math.Int, price string) types.QueueMessage {
	return suite.k.SetQueueMessage(suite.ctx, types.QueueMessage{
		MarketId:    getMarketId(),
		MessageType: types.OrderTypeSell,
		Amount:      amount.String(),
		Price:       price,
		OrderType:   types.OrderTypeSell,
		Owner:       owner.String(),
	})
}

func (suite *IntegrationTestSuite) TestOutcomeEvents_CancelOfMissingOrder() {
	suite.k.SetMarket(suite.ctx, market)
	addr1 := sdk.AccAddress("addr1_______________")

	cancelQm := suite.k.SetQueueMessage(suite.ctx, types.QueueMessage{
		MarketId:    getMarketId(),
		MessageType: types.MessageTypeCancel,
		OrderId:     "missing-order-id",
		OrderType:   types.OrderTypeBuy,
		Owner:       addr1.String(),
	})

	events := suite.processBlock()

	var skipped []*types.OrderCancelSkippedEvent
	for _, ev := range events {
		if e, ok := ev.(*types.OrderCancelSkippedEvent); ok {
			skipped = append(skipped, e)
		}
	}
	suite.Require().Len(skipped, 1)
	suite.Require().Equal(cancelQm.MessageId, skipped[0].MessageId)
	suite.Require().Equal(getMarketId(), skipped[0].MarketId)
	suite.Require().Equal(types.OrderTypeBuy, skipped[0].OrderType)
	suite.Require().Equal("missing-order-id", skipped[0].OrderId)
	suite.Require().Equal(addr1.String(), skipped[0].Owner)
	suite.Require().Equal(types.CancelSkippedReasonOrderNotFound, skipped[0].Reason)
	suite.Require().Empty(suite.k.GetAllQueueMessage(suite.ctx))
}

func (suite *IntegrationTestSuite) TestOutcomeEvents_FillWithNoMatch() {
	suite.k.SetMarket(suite.ctx, market)
	taker := sdk.AccAddress("addr2_______________")

	fillQm := suite.k.SetQueueMessage(suite.ctx, types.QueueMessage{
		MarketId:    getMarketId(),
		MessageType: types.MessageTypeFillBuy,
		Amount:      "5000",
		Price:       "2",
		OrderType:   types.OrderTypeBuy,
		Owner:       taker.String(),
	})

	// a buy message escrowed quote: amount * price is returned
	expectedRefund := sdk.NewInt64Coin(market.Quote, 10000)
	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, taker, sdk.NewCoins(expectedRefund)).Return(nil).Times(1)

	refunded := refundedEvents(suite.processBlock())
	suite.Require().Len(refunded, 1)
	suite.Require().Equal(fillQm.MessageId, refunded[0].MessageId)
	suite.Require().Equal(getMarketId(), refunded[0].MarketId)
	suite.Require().Equal(types.OrderTypeBuy, refunded[0].OrderType)
	suite.Require().Equal(taker.String(), refunded[0].Owner)
	suite.Require().Equal("5000", refunded[0].Amount)
	suite.Require().Equal(expectedRefund.String(), refunded[0].Refund)
	suite.Require().Equal(types.RefundReasonNoMatch, refunded[0].Reason)
}

func (suite *IntegrationTestSuite) TestOutcomeEvents_PartlyMatchedFill() {
	suite.k.SetMarket(suite.ctx, market)
	maker := sdk.AccAddress("addr1_______________")
	taker := sdk.AccAddress("addr2_______________")
	price := "1"
	minAmount, err := keeper.CalculateMinAmount(price)
	suite.Require().NoError(err)

	suite.restingSell(maker, minAmount, price)
	suite.processBlock()

	fillAmount := minAmount.MulRaw(5)
	fillQm := suite.k.SetQueueMessage(suite.ctx, types.QueueMessage{
		MarketId:    getMarketId(),
		MessageType: types.MessageTypeFillBuy,
		Amount:      fillAmount.String(),
		Price:       price,
		OrderType:   types.OrderTypeBuy,
		Owner:       taker.String(),
	})
	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	events := suite.processBlock()

	executed := 0
	for _, ev := range events {
		if e, ok := ev.(*types.OrderExecutedEvent); ok {
			executed++
			suite.Require().Equal(fillQm.MessageId, e.MessageId)
		}
	}
	suite.Require().Equal(1, executed)

	remainder := fillAmount.Sub(minAmount)
	refunded := refundedEvents(events)
	suite.Require().Len(refunded, 1)
	suite.Require().Equal(fillQm.MessageId, refunded[0].MessageId)
	suite.Require().Equal(remainder.String(), refunded[0].Amount)
	suite.Require().Equal(sdk.NewCoin(market.Quote, remainder).String(), refunded[0].Refund)
	suite.Require().Equal(types.RefundReasonUnfilledRemainder, refunded[0].Reason)
}

func (suite *IntegrationTestSuite) TestOutcomeEvents_CreateRemainderBelowMinAmount() {
	suite.k.SetMarket(suite.ctx, market)
	maker := sdk.AccAddress("addr1_______________")
	taker := sdk.AccAddress("addr2_______________")
	price := "1"
	minAmount, err := keeper.CalculateMinAmount(price)
	suite.Require().NoError(err)

	suite.restingSell(maker, minAmount, price)
	suite.processBlock()

	// one unit more than the resting order: the remainder (1) is below the min amount
	createQm := suite.k.SetQueueMessage(suite.ctx, types.QueueMessage{
		MarketId:    getMarketId(),
		MessageType: types.OrderTypeBuy,
		Amount:      minAmount.AddRaw(1).String(),
		Price:       price,
		OrderType:   types.OrderTypeBuy,
		Owner:       taker.String(),
	})
	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	events := suite.processBlock()
	for _, ev := range events {
		_, saved := ev.(*types.OrderSavedEvent)
		suite.Require().False(saved, "the remainder must not be placed")
	}

	refunded := refundedEvents(events)
	suite.Require().Len(refunded, 1)
	suite.Require().Equal(createQm.MessageId, refunded[0].MessageId)
	suite.Require().Equal(types.OrderTypeBuy, refunded[0].OrderType)
	suite.Require().Equal(taker.String(), refunded[0].Owner)
	suite.Require().Equal("1", refunded[0].Amount)
	suite.Require().Equal(sdk.NewInt64Coin(market.Quote, 1).String(), refunded[0].Refund)
	suite.Require().Equal(types.RefundReasonBelowMinAmount, refunded[0].Reason)
}

func (suite *IntegrationTestSuite) TestOutcomeEvents_RefundRoundingToZero() {
	suite.k.SetMarket(suite.ctx, market)
	taker := sdk.AccAddress("addr2_______________")

	// 1 base unit at 0.1 is worth 0.1 quote units: nothing is sent, only dust is stored
	fillQm := suite.k.SetQueueMessage(suite.ctx, types.QueueMessage{
		MarketId:    getMarketId(),
		MessageType: types.MessageTypeFillBuy,
		Amount:      "1",
		Price:       "0.1",
		OrderType:   types.OrderTypeBuy,
		Owner:       taker.String(),
	})
	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

	refunded := refundedEvents(suite.processBlock())
	suite.Require().Len(refunded, 1)
	suite.Require().Equal(fillQm.MessageId, refunded[0].MessageId)
	suite.Require().Equal("1", refunded[0].Amount)
	suite.Require().Empty(refunded[0].Refund)
	suite.Require().Equal(types.RefundReasonNoMatch, refunded[0].Reason)
	suite.Require().Empty(suite.k.GetAllQueueMessage(suite.ctx))
}

func (suite *IntegrationTestSuite) TestOutcomeEvents_ProcessingErrorKeepsMessageQueued() {
	suite.k.SetMarket(suite.ctx, market)
	taker := sdk.AccAddress("addr2_______________")

	fillQm := suite.k.SetQueueMessage(suite.ctx, types.QueueMessage{
		MarketId:    getMarketId(),
		MessageType: types.MessageTypeFillBuy,
		Amount:      "5000",
		Price:       "2",
		OrderType:   types.OrderTypeBuy,
		Owner:       taker.String(),
	})
	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, taker, gomock.Any()).
		Return(errors.New("bank failure with node specific details")).Times(2)

	for block := 0; block < 2; block++ {
		events := suite.processBlock()

		failed := failedEvents(events)
		suite.Require().Len(failed, 1, "block %d", block)
		suite.Require().Equal(fillQm.MessageId, failed[0].MessageId)
		suite.Require().Equal(getMarketId(), failed[0].MarketId)
		suite.Require().Equal(types.MessageTypeFillBuy, failed[0].MessageType)
		suite.Require().Equal(taker.String(), failed[0].Owner)
		suite.Require().Equal(types.MessageFailedReasonProcessingError, failed[0].Reason)

		// the refund event lived in the discarded cache context
		suite.Require().Empty(refundedEvents(events), "block %d", block)

		qms := suite.k.GetAllQueueMessage(suite.ctx)
		suite.Require().Len(qms, 1)
		suite.Require().Equal(fillQm.MessageId, qms[0].MessageId)
	}
}

// TestOutcomeEvents_EveryRemovedMessageHasAnEvent processes a mixed queue in one block and checks that
// every message removed from the queue appears in at least one engine event of that block.
func (suite *IntegrationTestSuite) TestOutcomeEvents_EveryRemovedMessageHasAnEvent() {
	suite.k.SetMarket(suite.ctx, market)
	maker := sdk.AccAddress("addr1_______________")
	taker := sdk.AccAddress("addr2_______________")
	price := "1"
	minAmount, err := keeper.CalculateMinAmount(price)
	suite.Require().NoError(err)
	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	// resting orders from an earlier block
	suite.restingSell(maker, minAmount.MulRaw(3), price)
	suite.restingSell(maker, minAmount, "3")
	suite.processBlock()
	toCancel := suite.k.GetAllOrder(suite.ctx)
	suite.Require().Len(toCancel, 2)

	queue := []types.QueueMessage{
		// create that rests (no match at its price)
		{MessageType: types.OrderTypeBuy, OrderType: types.OrderTypeBuy, Amount: minAmount.String(), Price: "0.5", Owner: taker.String()},
		// create fully matched
		{MessageType: types.OrderTypeBuy, OrderType: types.OrderTypeBuy, Amount: minAmount.String(), Price: price, Owner: taker.String()},
		// create with a remainder below the min amount
		{MessageType: types.OrderTypeBuy, OrderType: types.OrderTypeBuy, Amount: minAmount.AddRaw(1).String(), Price: price, Owner: taker.String()},
		// fill partly matched (one min amount left at the price)
		{MessageType: types.MessageTypeFillBuy, OrderType: types.OrderTypeBuy, Amount: minAmount.MulRaw(4).String(), Price: price, Owner: taker.String()},
		// fill with no match
		{MessageType: types.MessageTypeFillBuy, OrderType: types.OrderTypeBuy, Amount: minAmount.String(), Price: "7", Owner: taker.String()},
		// cancel of an existing order
		{MessageType: types.MessageTypeCancel, OrderType: types.OrderTypeSell, OrderId: toCancel[0].Id, Owner: maker.String()},
		// cancel of the same order again: it no longer exists
		{MessageType: types.MessageTypeCancel, OrderType: types.OrderTypeSell, OrderId: toCancel[0].Id, Owner: maker.String()},
		// cancel of an order that never existed
		{MessageType: types.MessageTypeCancel, OrderType: types.OrderTypeBuy, OrderId: "never-existed", Owner: taker.String()},
	}
	queued := map[string]struct{}{}
	for _, qm := range queue {
		qm.MarketId = getMarketId()
		saved := suite.k.SetQueueMessage(suite.ctx, qm)
		queued[saved.MessageId] = struct{}{}
	}

	events := suite.processBlock()
	suite.Require().Empty(suite.k.GetAllQueueMessage(suite.ctx))

	withEvent := map[string]struct{}{}
	for _, ev := range events {
		if id, ok := engineEventMessageId(ev); ok {
			withEvent[id] = struct{}{}
		}
	}
	for id := range queued {
		_, ok := withEvent[id]
		suite.Require().True(ok, "message %s was removed from the queue without an event", id)
	}
	suite.Require().Len(refundedEvents(events), 3)
}
