package keeper_test

import (
	"cosmossdk.io/math"
	"github.com/bze-alphateam/bze/x/tradebin/types"
)

func (suite *IntegrationTestSuite) TestParams_GetAndSet() {
	params := types.Params{
		CreateMarketFee:                   "1000",
		MarketMakerFee:                    "0.001",
		MarketTakerFee:                    "0.002",
		MakerFeeDestination:               "community_pool",
		TakerFeeDestination:               "burn",
		OrderBookExtraGasWindow:           200,
		OrderBookQueueExtraGas:            30000,
		FillOrdersExtraGas:                6000,
		OrderBookQueueMessageScanExtraGas: 5500,
		MinNativeLiquidityForModuleSwap:   math.NewInt(60000000000),
		OrderBookPerBlockMessages:         750,
	}

	// Test SetParams
	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	// Test GetParams
	retrievedParams := suite.k.GetParams(suite.ctx)
	suite.Require().Equal(params.CreateMarketFee, retrievedParams.CreateMarketFee)
	suite.Require().Equal(params.MarketMakerFee, retrievedParams.MarketMakerFee)
	suite.Require().Equal(params.MarketTakerFee, retrievedParams.MarketTakerFee)
	suite.Require().Equal(params.MakerFeeDestination, retrievedParams.MakerFeeDestination)
	suite.Require().Equal(params.TakerFeeDestination, retrievedParams.TakerFeeDestination)
	suite.Require().Equal(params.OrderBookExtraGasWindow, retrievedParams.OrderBookExtraGasWindow)
	suite.Require().Equal(params.OrderBookQueueExtraGas, retrievedParams.OrderBookQueueExtraGas)
	suite.Require().Equal(params.FillOrdersExtraGas, retrievedParams.FillOrdersExtraGas)
	suite.Require().Equal(params.OrderBookQueueMessageScanExtraGas, retrievedParams.OrderBookQueueMessageScanExtraGas)
	suite.Require().Equal(params.MinNativeLiquidityForModuleSwap, retrievedParams.MinNativeLiquidityForModuleSwap)
	suite.Require().Equal(params.OrderBookPerBlockMessages, retrievedParams.OrderBookPerBlockMessages)
}

func (suite *IntegrationTestSuite) TestParams_GetDefault() {
	// Test GetParams when no params are set (should return default values)
	retrievedParams := suite.k.GetParams(suite.ctx)

	// Default values from constants
	suite.Require().Equal("25000000000ubze", retrievedParams.CreateMarketFee)
	suite.Require().Equal("1000ubze", retrievedParams.MarketMakerFee)
	suite.Require().Equal("100000ubze", retrievedParams.MarketTakerFee)
	suite.Require().Equal("burner", retrievedParams.MakerFeeDestination)
	suite.Require().Equal("burner", retrievedParams.TakerFeeDestination)
	suite.Require().Equal(uint64(100), retrievedParams.OrderBookExtraGasWindow)
	suite.Require().Equal(uint64(25000), retrievedParams.OrderBookQueueExtraGas)
	suite.Require().Equal(uint64(5000), retrievedParams.FillOrdersExtraGas)
	suite.Require().Equal(uint64(5000), retrievedParams.OrderBookQueueMessageScanExtraGas)
	suite.Require().EqualValues(math.NewInt(100_000_000000), retrievedParams.MinNativeLiquidityForModuleSwap)
	suite.Require().Equal(uint64(500), retrievedParams.OrderBookPerBlockMessages)
}

func (suite *IntegrationTestSuite) TestParams_SetMultipleTimes() {
	params1 := types.Params{
		CreateMarketFee:                 "500",
		MarketMakerFee:                  "0.0005",
		MarketTakerFee:                  "0.001",
		MakerFeeDestination:             "community_pool",
		TakerFeeDestination:             "burn",
		OrderBookExtraGasWindow:         150,
		OrderBookQueueExtraGas:          20000,
		FillOrdersExtraGas:              4000,
		MinNativeLiquidityForModuleSwap: math.NewInt(40000000000),
	}

	params2 := types.Params{
		CreateMarketFee:                 "2000",
		MarketMakerFee:                  "0.002",
		MarketTakerFee:                  "0.004",
		MakerFeeDestination:             "burn",
		TakerFeeDestination:             "community_pool",
		OrderBookExtraGasWindow:         250,
		OrderBookQueueExtraGas:          35000,
		FillOrdersExtraGas:              7000,
		MinNativeLiquidityForModuleSwap: math.NewInt(70000000000),
	}

	// Set first params
	err := suite.k.SetParams(suite.ctx, params1)
	suite.Require().NoError(err)

	// Verify first params
	retrieved1 := suite.k.GetParams(suite.ctx)
	suite.Require().Equal(params1.CreateMarketFee, retrieved1.CreateMarketFee)
	suite.Require().Equal(params1.MarketMakerFee, retrieved1.MarketMakerFee)

	// Set second params (update)
	err = suite.k.SetParams(suite.ctx, params2)
	suite.Require().NoError(err)

	// Verify second params
	retrieved2 := suite.k.GetParams(suite.ctx)
	suite.Require().Equal(params2.CreateMarketFee, retrieved2.CreateMarketFee)
	suite.Require().Equal(params2.MarketMakerFee, retrieved2.MarketMakerFee)
	suite.Require().Equal(params2.MarketTakerFee, retrieved2.MarketTakerFee)
	suite.Require().Equal(params2.MakerFeeDestination, retrieved2.MakerFeeDestination)
	suite.Require().Equal(params2.TakerFeeDestination, retrieved2.TakerFeeDestination)
	suite.Require().Equal(params2.OrderBookExtraGasWindow, retrieved2.OrderBookExtraGasWindow)
	suite.Require().Equal(params2.OrderBookQueueExtraGas, retrieved2.OrderBookQueueExtraGas)
	suite.Require().Equal(params2.FillOrdersExtraGas, retrieved2.FillOrdersExtraGas)
	suite.Require().Equal(params2.MinNativeLiquidityForModuleSwap, retrieved2.MinNativeLiquidityForModuleSwap)
}

func (suite *IntegrationTestSuite) TestParams_CreateMarketFee() {
	params := types.Params{
		CreateMarketFee:     "1500",
		MarketMakerFee:      "0.001",
		MarketTakerFee:      "0.002",
		MakerFeeDestination: "community_pool",
		TakerFeeDestination: "burn",
	}

	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	// Test CreateMarketFee getter
	fee := suite.k.CreateMarketFee(suite.ctx)
	suite.Require().Equal("1500", fee)
}

func (suite *IntegrationTestSuite) TestParams_MarketMakerFee() {
	params := types.Params{
		CreateMarketFee:     "1000",
		MarketMakerFee:      "0.0015",
		MarketTakerFee:      "0.002",
		MakerFeeDestination: "community_pool",
		TakerFeeDestination: "burn",
	}

	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	// Test MarketMakerFee getter
	fee := suite.k.MarketMakerFee(suite.ctx)
	suite.Require().Equal("0.0015", fee)
}

func (suite *IntegrationTestSuite) TestParams_MarketTakerFee() {
	params := types.Params{
		CreateMarketFee:     "1000",
		MarketMakerFee:      "0.001",
		MarketTakerFee:      "0.0025",
		MakerFeeDestination: "community_pool",
		TakerFeeDestination: "burn",
	}

	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	// Test MarketTakerFee getter
	fee := suite.k.MarketTakerFee(suite.ctx)
	suite.Require().Equal("0.0025", fee)
}

func (suite *IntegrationTestSuite) TestParams_MakerFeeDestination() {
	params := types.Params{
		CreateMarketFee:     "1000",
		MarketMakerFee:      "0.001",
		MarketTakerFee:      "0.002",
		MakerFeeDestination: "validator_rewards",
		TakerFeeDestination: "burn",
	}

	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	// Test MakerFeeDestination getter
	destination := suite.k.MakerFeeDestination(suite.ctx)
	suite.Require().Equal("validator_rewards", destination)
}

func (suite *IntegrationTestSuite) TestParams_TakerFeeDestination() {
	params := types.Params{
		CreateMarketFee:     "1000",
		MarketMakerFee:      "0.001",
		MarketTakerFee:      "0.002",
		MakerFeeDestination: "community_pool",
		TakerFeeDestination: "validator_rewards",
	}

	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	// Test TakerFeeDestination getter
	destination := suite.k.TakerFeeDestination(suite.ctx)
	suite.Require().Equal("validator_rewards", destination)
}

func (suite *IntegrationTestSuite) TestParams_AllGettersDefault() {
	// Test all getter methods when no params are set (should return defaults)
	createFee := suite.k.CreateMarketFee(suite.ctx)
	suite.Require().Equal("25000000000ubze", createFee)

	makerFee := suite.k.MarketMakerFee(suite.ctx)
	suite.Require().Equal("1000ubze", makerFee)

	takerFee := suite.k.MarketTakerFee(suite.ctx)
	suite.Require().Equal("100000ubze", takerFee)

	makerDest := suite.k.MakerFeeDestination(suite.ctx)
	suite.Require().Equal("burner", makerDest)

	takerDest := suite.k.TakerFeeDestination(suite.ctx)
	suite.Require().Equal("burner", takerDest)
}

func (suite *IntegrationTestSuite) TestParams_ZeroValues() {
	params := types.Params{
		CreateMarketFee:     "0",
		MarketMakerFee:      "0",
		MarketTakerFee:      "0",
		MakerFeeDestination: "",
		TakerFeeDestination: "",
	}

	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	retrievedParams := suite.k.GetParams(suite.ctx)
	suite.Require().Equal("0", retrievedParams.CreateMarketFee)
	suite.Require().Equal("0", retrievedParams.MarketMakerFee)
	suite.Require().Equal("0", retrievedParams.MarketTakerFee)
	suite.Require().Equal("", retrievedParams.MakerFeeDestination)
	suite.Require().Equal("", retrievedParams.TakerFeeDestination)
}

func (suite *IntegrationTestSuite) TestParams_UpdateIndividualFields() {
	initialParams := types.Params{
		CreateMarketFee:     "1000",
		MarketMakerFee:      "0.001",
		MarketTakerFee:      "0.002",
		MakerFeeDestination: "community_pool",
		TakerFeeDestination: "burn",
	}

	// Set initial params
	err := suite.k.SetParams(suite.ctx, initialParams)
	suite.Require().NoError(err)

	// Update only some fields
	updatedParams := initialParams
	updatedParams.CreateMarketFee = "1500"
	updatedParams.MarketMakerFee = "0.0015"

	err = suite.k.SetParams(suite.ctx, updatedParams)
	suite.Require().NoError(err)

	// Verify update
	retrievedParams := suite.k.GetParams(suite.ctx)
	suite.Require().Equal("1500", retrievedParams.CreateMarketFee)
	suite.Require().Equal("0.0015", retrievedParams.MarketMakerFee)
	suite.Require().Equal("0.002", retrievedParams.MarketTakerFee)               // Should remain unchanged
	suite.Require().Equal("community_pool", retrievedParams.MakerFeeDestination) // Should remain unchanged
	suite.Require().Equal("burn", retrievedParams.TakerFeeDestination)           // Should remain unchanged
}

func (suite *IntegrationTestSuite) TestParams_Persistence() {
	params := types.Params{
		CreateMarketFee:                 "800",
		MarketMakerFee:                  "0.0008",
		MarketTakerFee:                  "0.0016",
		MakerFeeDestination:             "community_pool",
		TakerFeeDestination:             "burn",
		OrderBookExtraGasWindow:         120,
		OrderBookQueueExtraGas:          28000,
		FillOrdersExtraGas:              5500,
		MinNativeLiquidityForModuleSwap: math.NewInt(55000000000),
	}

	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	// Get params multiple times
	for i := 0; i < 5; i++ {
		retrievedParams := suite.k.GetParams(suite.ctx)
		suite.Require().Equal(params.CreateMarketFee, retrievedParams.CreateMarketFee)
		suite.Require().Equal(params.MarketMakerFee, retrievedParams.MarketMakerFee)
		suite.Require().Equal(params.MarketTakerFee, retrievedParams.MarketTakerFee)
		suite.Require().Equal(params.MakerFeeDestination, retrievedParams.MakerFeeDestination)
		suite.Require().Equal(params.TakerFeeDestination, retrievedParams.TakerFeeDestination)
		suite.Require().Equal(params.OrderBookExtraGasWindow, retrievedParams.OrderBookExtraGasWindow)
		suite.Require().Equal(params.OrderBookQueueExtraGas, retrievedParams.OrderBookQueueExtraGas)
		suite.Require().Equal(params.FillOrdersExtraGas, retrievedParams.FillOrdersExtraGas)
		suite.Require().Equal(params.MinNativeLiquidityForModuleSwap, retrievedParams.MinNativeLiquidityForModuleSwap)
	}
}

func (suite *IntegrationTestSuite) TestParams_OrderBookExtraGasWindow() {
	params := types.Params{
		CreateMarketFee:         "1000",
		MarketMakerFee:          "0.001",
		MarketTakerFee:          "0.002",
		MakerFeeDestination:     "community_pool",
		TakerFeeDestination:     "burn",
		OrderBookExtraGasWindow: 150,
		OrderBookQueueExtraGas:  25000,
		FillOrdersExtraGas:      5000,
	}

	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	retrievedParams := suite.k.GetParams(suite.ctx)
	suite.Require().Equal(uint64(150), retrievedParams.OrderBookExtraGasWindow)
}

func (suite *IntegrationTestSuite) TestParams_OrderBookQueueExtraGas() {
	params := types.Params{
		CreateMarketFee:         "1000",
		MarketMakerFee:          "0.001",
		MarketTakerFee:          "0.002",
		MakerFeeDestination:     "community_pool",
		TakerFeeDestination:     "burn",
		OrderBookExtraGasWindow: 100,
		OrderBookQueueExtraGas:  30000,
		FillOrdersExtraGas:      5000,
	}

	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	retrievedParams := suite.k.GetParams(suite.ctx)
	suite.Require().Equal(uint64(30000), retrievedParams.OrderBookQueueExtraGas)
}

func (suite *IntegrationTestSuite) TestParams_FillOrdersExtraGas() {
	params := types.Params{
		CreateMarketFee:                 "1000",
		MarketMakerFee:                  "0.001",
		MarketTakerFee:                  "0.002",
		MakerFeeDestination:             "community_pool",
		TakerFeeDestination:             "burn",
		OrderBookExtraGasWindow:         100,
		OrderBookQueueExtraGas:          25000,
		FillOrdersExtraGas:              6000,
		MinNativeLiquidityForModuleSwap: math.NewInt(50000000000),
	}

	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	retrievedParams := suite.k.GetParams(suite.ctx)
	suite.Require().Equal(uint64(6000), retrievedParams.FillOrdersExtraGas)
}

func (suite *IntegrationTestSuite) TestParams_MinNativeLiquidityForModuleSwap() {
	params := types.Params{
		CreateMarketFee:                 "1000",
		MarketMakerFee:                  "0.001",
		MarketTakerFee:                  "0.002",
		MakerFeeDestination:             "community_pool",
		TakerFeeDestination:             "burn",
		OrderBookExtraGasWindow:         100,
		OrderBookQueueExtraGas:          25000,
		FillOrdersExtraGas:              5000,
		MinNativeLiquidityForModuleSwap: math.NewInt(75000000000),
	}

	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	retrievedParams := suite.k.GetParams(suite.ctx)
	suite.Require().Equal("75000000000", retrievedParams.MinNativeLiquidityForModuleSwap.String())
}

func (suite *IntegrationTestSuite) TestParams_OrderBookPerBlockMessages() {
	params := types.Params{
		CreateMarketFee:                   "1000",
		MarketMakerFee:                    "0.001",
		MarketTakerFee:                    "0.002",
		MakerFeeDestination:               "community_pool",
		TakerFeeDestination:               "burn",
		OrderBookExtraGasWindow:           100,
		OrderBookQueueExtraGas:            25000,
		FillOrdersExtraGas:                5000,
		OrderBookQueueMessageScanExtraGas: 6000,
		MinNativeLiquidityForModuleSwap:   math.NewInt(50000000000),
		OrderBookPerBlockMessages:         1000,
	}

	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	retrievedParams := suite.k.GetParams(suite.ctx)
	suite.Require().Equal(uint64(1000), retrievedParams.OrderBookPerBlockMessages)
}

func (suite *IntegrationTestSuite) TestParams_OrderBookQueueMessageScanExtraGas() {
	params := types.Params{
		CreateMarketFee:                   "1000",
		MarketMakerFee:                    "0.001",
		MarketTakerFee:                    "0.002",
		MakerFeeDestination:               "community_pool",
		TakerFeeDestination:               "burn",
		OrderBookExtraGasWindow:           100,
		OrderBookQueueExtraGas:            25000,
		FillOrdersExtraGas:                5000,
		OrderBookQueueMessageScanExtraGas: 7500,
		MinNativeLiquidityForModuleSwap:   math.NewInt(50000000000),
		OrderBookPerBlockMessages:         500,
	}

	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	retrievedParams := suite.k.GetParams(suite.ctx)
	suite.Require().Equal(uint64(7500), retrievedParams.OrderBookQueueMessageScanExtraGas)
}

func (suite *IntegrationTestSuite) TestParams_GetDefault_AllFields() {
	// Test GetParams when no params are set (should return all default values)
	retrievedParams := suite.k.GetParams(suite.ctx)

	// Default values from constants
	suite.Require().Equal("25000000000ubze", retrievedParams.CreateMarketFee)
	suite.Require().Equal("1000ubze", retrievedParams.MarketMakerFee)
	suite.Require().Equal("100000ubze", retrievedParams.MarketTakerFee)
	suite.Require().Equal("burner", retrievedParams.MakerFeeDestination)
	suite.Require().Equal("burner", retrievedParams.TakerFeeDestination)
	suite.Require().Equal(uint64(100), retrievedParams.OrderBookExtraGasWindow)
	suite.Require().Equal(uint64(25000), retrievedParams.OrderBookQueueExtraGas)
	suite.Require().Equal(uint64(5000), retrievedParams.FillOrdersExtraGas)
	suite.Require().Equal(uint64(5000), retrievedParams.OrderBookQueueMessageScanExtraGas)
	suite.Require().EqualValues(math.NewInt(100_000_000000), retrievedParams.MinNativeLiquidityForModuleSwap)
	suite.Require().Equal(uint64(500), retrievedParams.OrderBookPerBlockMessages)
}
