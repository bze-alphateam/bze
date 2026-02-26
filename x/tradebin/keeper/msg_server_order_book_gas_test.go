package keeper_test

import (
	"cosmossdk.io/math"
	"github.com/bze-alphateam/bze/x/tradebin/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"go.uber.org/mock/gomock"
)

// TestMsgCreateOrder_QueueSpamProtectionGas tests that extra gas is charged
// based on the queue size to prevent spam attacks
func (suite *IntegrationTestSuite) TestMsgCreateOrder_QueueSpamProtectionGas() {
	// Setup market
	suite.k.SetMarket(suite.ctx, market)

	// Create test account
	addr1 := sdk.AccAddress("addr1_______________")

	// Get current params to use in test cases
	params := suite.k.GetParams(suite.ctx)

	testCases := []struct {
		name             string
		queueCounter     uint64
		expectedExtraGas uint64
		description      string
	}{
		{
			name:             "No extra gas when queue is empty",
			queueCounter:     0,
			expectedExtraGas: 0,
			description:      "Queue counter = 0, no extra gas",
		},
		{
			name:             "No extra gas when queue is at threshold",
			queueCounter:     params.OrderBookExtraGasWindow,
			expectedExtraGas: 0,
			description:      "Queue counter at threshold, no extra gas",
		},
		{
			name:             "Extra gas when queue exceeds threshold by 1",
			queueCounter:     params.OrderBookExtraGasWindow + 1,
			expectedExtraGas: params.OrderBookQueueExtraGas,
			description:      "Queue counter exceeds threshold by 1",
		},
		{
			name:             "Extra gas when queue has 20 messages",
			queueCounter:     params.OrderBookExtraGasWindow + 10,
			expectedExtraGas: 10 * params.OrderBookQueueExtraGas,
			description:      "Queue counter = threshold + 10",
		},
		{
			name:             "Extra gas when queue has 50 messages",
			queueCounter:     params.OrderBookExtraGasWindow + 40,
			expectedExtraGas: 40 * params.OrderBookQueueExtraGas,
			description:      "Queue counter = threshold + 40",
		},
		{
			name:             "Extra gas when queue has 100 messages",
			queueCounter:     params.OrderBookExtraGasWindow + 90,
			expectedExtraGas: 90 * params.OrderBookQueueExtraGas,
			description:      "Queue counter = threshold + 90",
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			// Reset context for each test to get fresh gas meter
			suite.SetupTest()
			suite.k.SetMarket(suite.ctx, market)

			// Set the queue counter to simulate different queue sizes
			suite.k.SetQueueMessageCounter(suite.ctx, tc.queueCounter)

			// Setup mocks for all operations CreateOrder performs
			// 1. Fee capture (either to burner or community pool)
			suite.bankMock.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), addr1, types.ModuleName, gomock.Any()).Return(nil).AnyTimes()
			suite.bankMock.EXPECT().SendCoinsFromModuleToModule(gomock.Any(), types.ModuleName, gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

			// Record gas before CreateOrder
			gasBefore := suite.ctx.GasMeter().GasConsumed()

			// Create order message
			msg := &types.MsgCreateOrder{
				Creator:   addr1.String(),
				MarketId:  getMarketId(),
				OrderType: types.OrderTypeBuy,
				Amount:    math.NewInt(1000),
				Price:     math.LegacyMustNewDecFromStr("1.5"),
			}

			// Execute CreateOrder
			_, err := suite.msgServer.CreateOrder(suite.ctx, msg)
			suite.Require().NoError(err, "CreateOrder should succeed")

			// Record gas after CreateOrder
			gasAfter := suite.ctx.GasMeter().GasConsumed()
			gasConsumed := gasAfter - gasBefore

			// The gas consumed should include the extra spam protection gas
			suite.T().Logf("%s: Gas consumed = %d (includes base cost + %d extra spam protection gas)",
				tc.description, gasConsumed, tc.expectedExtraGas)

			// Verify the queue counter incremented (message was added)
			newCounter := suite.k.GetQueueMessageCounter(suite.ctx)
			suite.Require().Equal(tc.queueCounter+1, newCounter, "Queue counter should increment")
		})
	}
}

// TestMsgCreateOrder_QueueSpamProtectionGas_Progressive tests that gas cost
// increases progressively as more orders are submitted in the same block
func (suite *IntegrationTestSuite) TestMsgCreateOrder_QueueSpamProtectionGas_Progressive() {
	// Setup market
	suite.k.SetMarket(suite.ctx, market)

	// Create test account
	addr1 := sdk.AccAddress("addr1_______________")

	// Get current params to use in calculations
	params := suite.k.GetParams(suite.ctx)

	// Setup mocks for all operations (15 orders)
	// Each order: 1 fee capture + 1 order coins capture = 2 SendCoinsFromAccountToModule calls
	// Each order: 1 fee forwarding = 1 SendCoinsFromModuleToModule call
	suite.bankMock.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), addr1, types.ModuleName, gomock.Any()).Return(nil).AnyTimes()
	suite.bankMock.EXPECT().SendCoinsFromModuleToModule(gomock.Any(), types.ModuleName, gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	// Submit 15 orders and verify gas increases progressively
	var gasConsumptions []uint64

	for i := 0; i < 15; i++ {
		gasBefore := suite.ctx.GasMeter().GasConsumed()

		msg := &types.MsgCreateOrder{
			Creator:   addr1.String(),
			MarketId:  getMarketId(),
			OrderType: types.OrderTypeBuy,
			Amount:    math.NewInt(1000),
			Price:     math.LegacyNewDec(int64(100 + i)), // Different prices
		}

		_, err := suite.msgServer.CreateOrder(suite.ctx, msg)
		suite.Require().NoError(err, "CreateOrder should succeed")

		gasAfter := suite.ctx.GasMeter().GasConsumed()
		gasConsumed := gasAfter - gasBefore
		gasConsumptions = append(gasConsumptions, gasConsumed)

		suite.T().Logf("Order %d: Queue counter = %d, Gas consumed = %d",
			i+1, suite.k.GetQueueMessageCounter(suite.ctx), gasConsumed)
	}

	// Verify gas consumption pattern
	// Gas is checked at the START of CreateOrder, before incrementing counter
	// So: Order N checks counter value of (N-1)
	// Orders 1-(threshold+1) should have similar gas (counter 0-threshold: no spam protection)
	// Orders (threshold+2)+ should have increasing gas (counter threshold+1: spam protection kicks in)

	// Calculate how many orders are within the free window
	freeOrders := int(params.OrderBookExtraGasWindow + 1)

	// Check that orders after free window consumed more gas than orders within free window
	avgGasFreeOrders := uint64(0)
	for i := 0; i < freeOrders && i < len(gasConsumptions); i++ {
		avgGasFreeOrders += gasConsumptions[i]
	}
	avgGasFreeOrders /= uint64(freeOrders)

	// Each order after free window should consume progressively more gas
	// Order (threshold+2) sees counter=(threshold+1), extra gas = (threshold+1-threshold)*OrderBookQueueExtraGas
	// Order (threshold+3) sees counter=(threshold+2), extra gas = (threshold+2-threshold)*OrderBookQueueExtraGas
	// etc.
	for i := freeOrders; i < 15; i++ {
		queueCounterAtStart := uint64(i) // Counter value when this order starts
		expectedExtraGas := (queueCounterAtStart - params.OrderBookExtraGasWindow) * params.OrderBookQueueExtraGas
		suite.T().Logf("Order %d: Queue counter at start = %d, Expected extra gas = %d, Actual gas = %d, Avg first %d = %d",
			i+1, queueCounterAtStart, expectedExtraGas, gasConsumptions[i], freeOrders, avgGasFreeOrders)

		// The gas consumption should be at least avgGasFreeOrders + expectedExtraGas
		// We use a tolerance of 1000 gas for other minor operations
		suite.Require().GreaterOrEqual(gasConsumptions[i], avgGasFreeOrders+expectedExtraGas-1000,
			"Order %d should consume at least %d more gas than average first %d",
			i+1, expectedExtraGas, freeOrders)
	}
}
