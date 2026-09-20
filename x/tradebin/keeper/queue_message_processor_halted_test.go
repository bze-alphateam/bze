package keeper_test

import (
	"cosmossdk.io/math"
	"github.com/bze-alphateam/bze/x/tradebin/keeper"
	"github.com/bze-alphateam/bze/x/tradebin/types"
	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/gogoproto/proto"
	"go.uber.org/mock/gomock"
)

// refundedEvents decodes every QueueMessageRefundedEvent emitted in ctx.
func (suite *IntegrationTestSuite) refundedEvents(ctx sdk.Context) (out []*types.QueueMessageRefundedEvent) {
	eventType := proto.MessageName(&types.QueueMessageRefundedEvent{})
	for _, ev := range ctx.EventManager().Events() {
		if ev.Type != eventType {
			continue
		}
		parsed, err := sdk.ParseTypedEvent(abci.Event(ev))
		suite.Require().NoError(err)
		typed, ok := parsed.(*types.QueueMessageRefundedEvent)
		suite.Require().True(ok)
		out = append(out, typed)
	}

	return out
}

func (suite *IntegrationTestSuite) countEvents(ctx sdk.Context, msg proto.Message) int {
	n := 0
	for _, ev := range ctx.EventManager().Events() {
		if ev.Type == proto.MessageName(msg) {
			n++
		}
	}

	return n
}

// restingBook seeds the halted-quote market (stake/uhalt) with one resting sell at price 2 and one
// resting buy at price 1, both of amount 1,000,000 owned by maker, through the engine itself.
func (suite *IntegrationTestSuite) restingBook(engine *keeper.ProcessingEngine, maker sdk.AccAddress) types.Market {
	m := haltedQuoteMarket()
	suite.k.SetMarket(suite.ctx, m)
	for _, qm := range []types.QueueMessage{
		{MarketId: marketIdOf(m), MessageType: types.OrderTypeSell, OrderType: types.OrderTypeSell, Amount: "1000000", Price: "2", Owner: maker.String()},
		{MarketId: marketIdOf(m), MessageType: types.OrderTypeBuy, OrderType: types.OrderTypeBuy, Amount: "1000000", Price: "1", Owner: maker.String()},
	} {
		suite.k.SetQueueMessage(suite.ctx, qm)
	}
	engine.ProcessQueueMessages(suite.ctx)

	suite.Require().Len(suite.k.GetAllOrder(suite.ctx), 2)
	suite.checkAggregatedOrder(marketIdOf(m), types.OrderTypeSell, "2", "1000000")
	suite.checkAggregatedOrder(marketIdOf(m), types.OrderTypeBuy, "1", "1000000")
	suite.Require().Empty(suite.k.GetAllQueueMessage(suite.ctx))

	return m
}

// crossingMessages returns one message of each non-cancel type, each priced to cross the resting book.
func crossingMessages(m types.Market, taker sdk.AccAddress) []types.QueueMessage {
	id := marketIdOf(m)
	return []types.QueueMessage{
		// a buy at the resting sell's price
		{MarketId: id, MessageType: types.OrderTypeBuy, OrderType: types.OrderTypeBuy, Amount: "500000", Price: "2", Owner: taker.String()},
		// a sell at the resting buy's price
		{MarketId: id, MessageType: types.OrderTypeSell, OrderType: types.OrderTypeSell, Amount: "500000", Price: "1", Owner: taker.String()},
		// fill the resting sell (the filler acts as a buyer)
		{MarketId: id, MessageType: types.MessageTypeFillSell, OrderType: types.OrderTypeBuy, Amount: "500000", Price: "2", Owner: taker.String()},
		// fill the resting buy (the filler acts as a seller)
		{MarketId: id, MessageType: types.MessageTypeFillBuy, OrderType: types.OrderTypeSell, Amount: "500000", Price: "1", Owner: taker.String()},
	}
}

// expectFullRefund registers the bank call refundMessageFunds makes for the whole escrowed amount.
func (suite *IntegrationTestSuite) expectFullRefund(qm types.QueueMessage, m *types.Market, owner sdk.AccAddress) sdk.Coin {
	amount, ok := math.NewIntFromString(qm.Amount)
	suite.Require().True(ok)
	coin, _, err := suite.k.GetOrderSdkCoin(qm.OrderType, qm.Price, amount, m)
	suite.Require().NoError(err)
	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, owner, sdk.NewCoins(coin)).Return(nil).Times(1)

	return coin
}

// TestQueueMessageProcessor_HaltedMarket_RefundsAndKeepsExits: with the market halted, every queued
// buy / sell / fill_buy / fill_sell that would have crossed the resting book is refunded in full to
// its owner with a QueueMessageRefundedEvent, leaves the queue, and touches neither the resting
// orders nor the aggregates; a cancel on the same market is processed normally; and once the denom
// is un-halted a new message matches the resting order again.
func (suite *IntegrationTestSuite) TestQueueMessageProcessor_HaltedMarket_RefundsAndKeepsExits() {
	engine, err := keeper.NewProcessingEngine(suite.k, suite.bankMock, suite.k.Logger())
	suite.Require().NoError(err)
	maker := sdk.AccAddress("addr1_______________")
	taker := sdk.AccAddress("addr2_______________")
	m := suite.restingBook(engine, maker)
	restingBefore := suite.k.GetAllOrder(suite.ctx)

	// --- halt the quote denom, queue crossing messages of every type ---
	suite.setHaltedDenoms(denomHalted)
	msgs := crossingMessages(m, taker)
	for _, qm := range msgs {
		suite.k.SetQueueMessage(suite.ctx, qm)
		suite.expectFullRefund(qm, &m, taker)
	}

	ctx := suite.ctx.WithEventManager(sdk.NewEventManager())
	engine.ProcessQueueMessages(ctx)

	// the queue is drained and the counter reset
	suite.Require().Empty(suite.k.GetAllQueueMessage(suite.ctx))
	suite.Require().Zero(suite.k.GetQueueMessageCounter(suite.ctx))

	// the book is exactly as it was
	suite.Require().ElementsMatch(restingBefore, suite.k.GetAllOrder(suite.ctx))
	suite.checkAggregatedOrder(marketIdOf(m), types.OrderTypeSell, "2", "1000000")
	suite.checkAggregatedOrder(marketIdOf(m), types.OrderTypeBuy, "1", "1000000")
	suite.Require().Empty(suite.k.GetAllHistoryOrder(suite.ctx), "nothing executed")

	// one refund event per message, nothing executed or saved
	events := suite.refundedEvents(ctx)
	suite.Require().Len(events, len(msgs))
	for i, ev := range events {
		suite.Require().Equal(marketIdOf(m), ev.MarketId)
		suite.Require().Equal(msgs[i].MessageType, ev.MessageType)
		suite.Require().Equal(msgs[i].OrderType, ev.OrderType)
		suite.Require().Equal(msgs[i].Amount, ev.Amount)
		suite.Require().Equal(msgs[i].Price, ev.Price)
		suite.Require().Equal(taker.String(), ev.Owner)
		suite.Require().Equal(types.RefundReasonMarketHalted, ev.Reason)
	}
	suite.Require().Zero(suite.countEvents(ctx, &types.OrderExecutedEvent{}))
	suite.Require().Zero(suite.countEvents(ctx, &types.OrderSavedEvent{}))

	// --- a cancel on the halted market is processed normally ---
	var restingSell types.Order
	for _, o := range restingBefore {
		if o.OrderType == types.OrderTypeSell {
			restingSell = o
		}
	}
	suite.Require().NotEmpty(restingSell.Id)
	suite.k.SetPendingCancel(suite.ctx, m.Base+"/"+m.Quote, restingSell.OrderType, restingSell.Id)
	suite.k.SetQueueMessage(suite.ctx, types.QueueMessage{
		MarketId:    marketIdOf(m),
		MessageType: types.MessageTypeCancel,
		OrderId:     restingSell.Id,
		OrderType:   restingSell.OrderType,
		Owner:       maker.String(),
	})
	sellAmount, _ := math.NewIntFromString(restingSell.Amount)
	cancelRefund, _, err := suite.k.GetOrderSdkCoin(restingSell.OrderType, restingSell.Price, sellAmount, &m)
	suite.Require().NoError(err)
	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, maker, sdk.NewCoins(cancelRefund)).Return(nil).Times(1)

	ctx = suite.ctx.WithEventManager(sdk.NewEventManager())
	engine.ProcessQueueMessages(ctx)

	_, found := suite.k.GetOrder(suite.ctx, marketIdOf(m), restingSell.OrderType, restingSell.Id)
	suite.Require().False(found, "the cancelled order is gone")
	_, found = suite.k.GetAggregatedOrder(suite.ctx, marketIdOf(m), types.OrderTypeSell, "2")
	suite.Require().False(found)
	suite.checkAggregatedOrder(marketIdOf(m), types.OrderTypeBuy, "1", "1000000")
	suite.Require().Empty(suite.refundedEvents(ctx), "a cancel is not a refund")
	suite.Require().Equal(1, suite.countEvents(ctx, &types.OrderCanceledEvent{}))

	// --- un-halt: a new sell matches the resting buy (nothing stays frozen) ---
	suite.setHaltedDenoms()
	qmSell := types.QueueMessage{
		MarketId: marketIdOf(m), MessageType: types.OrderTypeSell, OrderType: types.OrderTypeSell,
		Amount: "500000", Price: "1", Owner: taker.String(),
	}
	suite.k.SetQueueMessage(suite.ctx, qmSell)
	tradeAmount := math.NewInt(500000)
	makerCoins, _, err := suite.k.GetOrderSdkCoin(types.OrderTypeSell, "1", tradeAmount, &m) // the resting buyer receives base
	suite.Require().NoError(err)
	takerCoins, _, err := suite.k.GetOrderSdkCoin(types.OrderTypeBuy, "1", tradeAmount, &m) // the seller receives quote
	suite.Require().NoError(err)
	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, maker, sdk.NewCoins(makerCoins)).Return(nil).Times(1)
	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, taker, sdk.NewCoins(takerCoins)).Return(nil).Times(1)

	ctx = suite.ctx.WithEventManager(sdk.NewEventManager())
	engine.ProcessQueueMessages(ctx)

	suite.checkAggregatedOrder(marketIdOf(m), types.OrderTypeBuy, "1", "500000")
	suite.Require().Equal(1, suite.countEvents(ctx, &types.OrderExecutedEvent{}))
	suite.Require().Empty(suite.refundedEvents(ctx))
	suite.Require().Len(suite.k.GetAllHistoryOrder(suite.ctx), 1)
}

// A halt backlog larger than order_book_per_block_messages is refunded across blocks, in order.
func (suite *IntegrationTestSuite) TestQueueMessageProcessor_HaltedMarket_BacklogAcrossBlocks() {
	engine, err := keeper.NewProcessingEngine(suite.k, suite.bankMock, suite.k.Logger())
	suite.Require().NoError(err)
	maker := sdk.AccAddress("addr1_______________")
	taker := sdk.AccAddress("addr2_______________")
	m := suite.restingBook(engine, maker)

	params := suite.k.GetParams(suite.ctx)
	params.OrderBookPerBlockMessages = 3
	params.HaltedDenoms = []string{denomHalted}
	suite.Require().NoError(suite.k.SetParams(suite.ctx, params))

	msgs := crossingMessages(m, taker)
	for _, qm := range msgs {
		suite.k.SetQueueMessage(suite.ctx, qm)
		suite.expectFullRefund(qm, &m, taker)
	}

	// block 1: three refunds, one message left
	ctx := suite.ctx.WithEventManager(sdk.NewEventManager())
	engine.ProcessQueueMessages(ctx)
	suite.Require().Len(suite.refundedEvents(ctx), 3)
	suite.Require().Len(suite.k.GetAllQueueMessage(suite.ctx), 1)

	// block 2: the last one
	ctx = suite.ctx.WithEventManager(sdk.NewEventManager())
	engine.ProcessQueueMessages(ctx)
	suite.Require().Len(suite.refundedEvents(ctx), 1)
	suite.Require().Empty(suite.k.GetAllQueueMessage(suite.ctx))

	// the book never moved
	suite.checkAggregatedOrder(marketIdOf(m), types.OrderTypeSell, "2", "1000000")
	suite.checkAggregatedOrder(marketIdOf(m), types.OrderTypeBuy, "1", "1000000")
}

// The refund path only triggers for a halted market: with the list empty the existing matching
// behaviour is untouched (the crossing messages execute instead of being refunded).
func (suite *IntegrationTestSuite) TestQueueMessageProcessor_EmptyHaltList_MatchesAsBefore() {
	engine, err := keeper.NewProcessingEngine(suite.k, suite.bankMock, suite.k.Logger())
	suite.Require().NoError(err)
	maker := sdk.AccAddress("addr1_______________")
	taker := sdk.AccAddress("addr2_______________")
	m := suite.restingBook(engine, maker)

	suite.k.SetQueueMessage(suite.ctx, crossingMessages(m, taker)[0]) // buy 500000 @ 2 against the resting sell
	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, gomock.Any(), gomock.Any()).Return(nil).Times(2)

	ctx := suite.ctx.WithEventManager(sdk.NewEventManager())
	engine.ProcessQueueMessages(ctx)

	suite.Require().Empty(suite.refundedEvents(ctx))
	suite.Require().Equal(1, suite.countEvents(ctx, &types.OrderExecutedEvent{}))
	suite.checkAggregatedOrder(marketIdOf(m), types.OrderTypeSell, "2", "500000")
}
