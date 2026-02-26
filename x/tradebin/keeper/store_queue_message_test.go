package keeper_test

import (
	"cosmossdk.io/math"
	"github.com/bze-alphateam/bze/x/tradebin/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// TestStore_QueueMessage_CompositeKey tests that queue messages are stored with composite keys
// and can be retrieved efficiently by market ID
func (suite *IntegrationTestSuite) TestStore_QueueMessage_CompositeKey() {
	market1 := "market1"
	market2 := "market2"
	addr1 := sdk.AccAddress("addr1_______________")

	// Create messages for market1
	msg1 := types.QueueMessage{
		MarketId:    market1,
		MessageType: types.OrderTypeBuy,
		Amount:      math.NewInt(100),
		Price:       math.LegacyMustNewDecFromStr("10"),
		OrderType:   types.OrderTypeBuy,
		Owner:       addr1.String(),
	}
	msg2 := types.QueueMessage{
		MarketId:    market1,
		MessageType: types.OrderTypeSell,
		Amount:      math.NewInt(200),
		Price:       math.LegacyMustNewDecFromStr("20"),
		OrderType:   types.OrderTypeSell,
		Owner:       addr1.String(),
	}

	// Create messages for market2
	msg3 := types.QueueMessage{
		MarketId:    market2,
		MessageType: types.OrderTypeBuy,
		Amount:      math.NewInt(300),
		Price:       math.LegacyMustNewDecFromStr("30"),
		OrderType:   types.OrderTypeBuy,
		Owner:       addr1.String(),
	}

	// Add all messages
	suite.k.SetQueueMessage(suite.ctx, msg1)
	suite.k.SetQueueMessage(suite.ctx, msg2)
	suite.k.SetQueueMessage(suite.ctx, msg3)

	// Verify counter incremented correctly
	counter := suite.k.GetQueueMessageCounter(suite.ctx)
	suite.Require().Equal(uint64(3), counter)

	// Test GetAllQueueMessage returns all messages
	allMessages := suite.k.GetAllQueueMessage(suite.ctx)
	suite.Require().Len(allMessages, 3)

	// Test GetQueueMessagesByMarket for market1
	market1Messages := suite.k.GetQueueMessagesByMarket(suite.ctx, market1)
	suite.Require().Len(market1Messages, 2)

	// Verify market1 messages are correct
	for _, msg := range market1Messages {
		suite.Require().Equal(market1, msg.MarketId)
	}

	// Verify messages are in temporal order (by message ID)
	suite.Require().True(market1Messages[0].MessageId < market1Messages[1].MessageId)

	// Test GetQueueMessagesByMarket for market2
	market2Messages := suite.k.GetQueueMessagesByMarket(suite.ctx, market2)
	suite.Require().Len(market2Messages, 1)
	suite.Require().Equal(market2, market2Messages[0].MarketId)
	suite.Require().Equal(math.NewInt(300), market2Messages[0].Amount)
}

// TestStore_QueueMessage_TemporalOrderingWithinMarket tests that messages within a market
// are ordered by their temporal sequence (message ID)
func (suite *IntegrationTestSuite) TestStore_QueueMessage_TemporalOrderingWithinMarket() {
	marketId := "test-market"
	addr1 := sdk.AccAddress("addr1_______________")

	// Create messages in a specific order
	messages := []types.QueueMessage{
		{
			MarketId:    marketId,
			MessageType: types.OrderTypeBuy,
			Amount:      math.NewInt(100),
			Price:       math.LegacyMustNewDecFromStr("10"),
			OrderType:   types.OrderTypeBuy,
			Owner:       addr1.String(),
		},
		{
			MarketId:    marketId,
			MessageType: types.OrderTypeSell,
			Amount:      math.NewInt(200),
			Price:       math.LegacyMustNewDecFromStr("20"),
			OrderType:   types.OrderTypeSell,
			Owner:       addr1.String(),
		},
		{
			MarketId:    marketId,
			MessageType: types.OrderTypeBuy,
			Amount:      math.NewInt(300),
			Price:       math.LegacyMustNewDecFromStr("30"),
			OrderType:   types.OrderTypeBuy,
			Owner:       addr1.String(),
		},
		{
			MarketId:    marketId,
			MessageType: types.MessageTypeCancel,
			OrderId:     "order1",
			OrderType:   types.OrderTypeBuy,
			Owner:       addr1.String(),
		},
	}

	// Add all messages
	for _, msg := range messages {
		suite.k.SetQueueMessage(suite.ctx, msg)
	}

	// Retrieve messages for this market
	retrievedMessages := suite.k.GetQueueMessagesByMarket(suite.ctx, marketId)
	suite.Require().Len(retrievedMessages, 4)

	// Verify temporal ordering (message IDs should be in ascending order)
	for i := 0; i < len(retrievedMessages)-1; i++ {
		suite.Require().True(
			retrievedMessages[i].MessageId < retrievedMessages[i+1].MessageId,
			"Messages should be ordered by message ID",
		)
	}

	// Verify message types are preserved in order
	suite.Require().Equal(types.OrderTypeBuy, retrievedMessages[0].MessageType)
	suite.Require().Equal(types.OrderTypeSell, retrievedMessages[1].MessageType)
	suite.Require().Equal(types.OrderTypeBuy, retrievedMessages[2].MessageType)
	suite.Require().Equal(types.MessageTypeCancel, retrievedMessages[3].MessageType)
}

// TestStore_QueueMessage_RemoveByMarketAndId tests that messages can be removed
// using the composite key (market ID + message ID)
func (suite *IntegrationTestSuite) TestStore_QueueMessage_RemoveByMarketAndId() {
	market1 := "market1"
	market2 := "market2"
	addr1 := sdk.AccAddress("addr1_______________")

	// Create messages
	msg1 := types.QueueMessage{
		MarketId:    market1,
		MessageType: types.OrderTypeBuy,
		Amount:      math.NewInt(100),
		Price:       math.LegacyMustNewDecFromStr("10"),
		OrderType:   types.OrderTypeBuy,
		Owner:       addr1.String(),
	}
	msg2 := types.QueueMessage{
		MarketId:    market1,
		MessageType: types.OrderTypeSell,
		Amount:      math.NewInt(200),
		Price:       math.LegacyMustNewDecFromStr("20"),
		OrderType:   types.OrderTypeSell,
		Owner:       addr1.String(),
	}
	msg3 := types.QueueMessage{
		MarketId:    market2,
		MessageType: types.OrderTypeBuy,
		Amount:      math.NewInt(300),
		Price:       math.LegacyMustNewDecFromStr("30"),
		OrderType:   types.OrderTypeBuy,
		Owner:       addr1.String(),
	}

	// Add messages
	suite.k.SetQueueMessage(suite.ctx, msg1)
	suite.k.SetQueueMessage(suite.ctx, msg2)
	suite.k.SetQueueMessage(suite.ctx, msg3)

	// Get all messages to find message IDs
	allMessages := suite.k.GetAllQueueMessage(suite.ctx)
	suite.Require().Len(allMessages, 3)

	// Store message IDs
	msg1Id := allMessages[0].MessageId
	msg2Id := allMessages[1].MessageId

	// Remove first message from market1
	suite.k.RemoveQueueMessage(suite.ctx, market1, msg1Id)

	// Verify market1 now has only 1 message
	market1Messages := suite.k.GetQueueMessagesByMarket(suite.ctx, market1)
	suite.Require().Len(market1Messages, 1)
	suite.Require().Equal(msg2Id, market1Messages[0].MessageId)

	// Verify market2 is unaffected
	market2Messages := suite.k.GetQueueMessagesByMarket(suite.ctx, market2)
	suite.Require().Len(market2Messages, 1)

	// Verify total message count
	allMessages = suite.k.GetAllQueueMessage(suite.ctx)
	suite.Require().Len(allMessages, 2)
}

// TestStore_QueueMessage_MultipleMarketsIndependent tests that messages from different
// markets are stored and retrieved independently
func (suite *IntegrationTestSuite) TestStore_QueueMessage_MultipleMarketsIndependent() {
	addr1 := sdk.AccAddress("addr1_______________")

	// Create messages for 3 different markets
	markets := []string{"btc-usd", "eth-usd", "atom-usd"}
	messagesPerMarket := 5

	for _, marketId := range markets {
		for i := 0; i < messagesPerMarket; i++ {
			msg := types.QueueMessage{
				MarketId:    marketId,
				MessageType: types.OrderTypeBuy,
				Amount:      math.NewInt(100),
				Price:       math.LegacyMustNewDecFromStr("10"),
				OrderType:   types.OrderTypeBuy,
				Owner:       addr1.String(),
			}
			suite.k.SetQueueMessage(suite.ctx, msg)
		}
	}

	// Verify each market has exactly 5 messages
	for _, marketId := range markets {
		marketMessages := suite.k.GetQueueMessagesByMarket(suite.ctx, marketId)
		suite.Require().Len(marketMessages, messagesPerMarket, "Market %s should have %d messages", marketId, messagesPerMarket)

		// Verify all messages belong to correct market
		for _, msg := range marketMessages {
			suite.Require().Equal(marketId, msg.MarketId)
		}
	}

	// Verify total count
	allMessages := suite.k.GetAllQueueMessage(suite.ctx)
	suite.Require().Len(allMessages, len(markets)*messagesPerMarket)
}

// TestStore_QueueMessage_EmptyMarketQuery tests querying a market with no messages
func (suite *IntegrationTestSuite) TestStore_QueueMessage_EmptyMarketQuery() {
	marketId := "empty-market"

	// Query non-existent market
	messages := suite.k.GetQueueMessagesByMarket(suite.ctx, marketId)
	suite.Require().Empty(messages)
}

// TestStore_QueueMessage_FilterByMessageType tests that filtering by message type
// works correctly when combined with market filtering
func (suite *IntegrationTestSuite) TestStore_QueueMessage_FilterByMessageType() {
	marketId := "test-market"
	addr1 := sdk.AccAddress("addr1_______________")

	// Create mixed message types
	buyCount := 3
	sellCount := 2
	cancelCount := 1

	for i := 0; i < buyCount; i++ {
		suite.k.SetQueueMessage(suite.ctx, types.QueueMessage{
			MarketId:    marketId,
			MessageType: types.OrderTypeBuy,
			Amount:      math.NewInt(100),
			Price:       math.LegacyMustNewDecFromStr("10"),
			OrderType:   types.OrderTypeBuy,
			Owner:       addr1.String(),
		})
	}

	for i := 0; i < sellCount; i++ {
		suite.k.SetQueueMessage(suite.ctx, types.QueueMessage{
			MarketId:    marketId,
			MessageType: types.OrderTypeSell,
			Amount:      math.NewInt(200),
			Price:       math.LegacyMustNewDecFromStr("20"),
			OrderType:   types.OrderTypeSell,
			Owner:       addr1.String(),
		})
	}

	for i := 0; i < cancelCount; i++ {
		suite.k.SetQueueMessage(suite.ctx, types.QueueMessage{
			MarketId:    marketId,
			MessageType: types.MessageTypeCancel,
			OrderId:     "order1",
			OrderType:   types.OrderTypeBuy,
			Owner:       addr1.String(),
		})
	}

	// Get all messages for the market
	allMarketMessages := suite.k.GetQueueMessagesByMarket(suite.ctx, marketId)
	suite.Require().Len(allMarketMessages, buyCount+sellCount+cancelCount)

	// Filter by message type manually (as done in checkPriceInQueueMessages)
	buyMessages := []types.QueueMessage{}
	sellMessages := []types.QueueMessage{}
	cancelMessages := []types.QueueMessage{}

	for _, msg := range allMarketMessages {
		switch msg.MessageType {
		case types.OrderTypeBuy:
			buyMessages = append(buyMessages, msg)
		case types.OrderTypeSell:
			sellMessages = append(sellMessages, msg)
		case types.MessageTypeCancel:
			cancelMessages = append(cancelMessages, msg)
		}
	}

	// Verify counts
	suite.Require().Len(buyMessages, buyCount)
	suite.Require().Len(sellMessages, sellCount)
	suite.Require().Len(cancelMessages, cancelCount)
}
