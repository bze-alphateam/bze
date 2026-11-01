package keeper_test

import (
	"cosmossdk.io/math"
	v2types "github.com/bze-alphateam/bze/x/tradebin/v2types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (suite *IntegrationTestSuite) TestParams_GetAndSet() {
	params := v2types.Params{
		CreateMarketFee:                 sdk.NewInt64Coin(denomBze, 1000),
		MarketMakerFee:                  sdk.NewInt64Coin(denomBze, 100),
		MarketTakerFee:                  sdk.NewInt64Coin(denomBze, 200),
		MakerFeeDestination:             v2types.FeeDestinationCommunityPool,
		TakerFeeDestination:             v2types.FeeDestinationBurnerModule,
		NativeDenom:                     denomBze,
		PriceLevelOwnersWindow:          200,
		PriceLevelExtraGas:              30000,
		FillOrdersExtraGas:              6000,
		MinNativeLiquidityForModuleSwap: math.NewInt(60000000000),
		MaxFillsPerOrder:                750,
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
	suite.Require().Equal(params.PriceLevelOwnersWindow, retrievedParams.PriceLevelOwnersWindow)
	suite.Require().Equal(params.PriceLevelExtraGas, retrievedParams.PriceLevelExtraGas)
	suite.Require().Equal(params.FillOrdersExtraGas, retrievedParams.FillOrdersExtraGas)
	suite.Require().Equal(params.MinNativeLiquidityForModuleSwap, retrievedParams.MinNativeLiquidityForModuleSwap)
	suite.Require().Equal(params.MaxFillsPerOrder, retrievedParams.MaxFillsPerOrder)
}

func (suite *IntegrationTestSuite) TestParams_GetDefault() {
	retrievedParams := suite.k.GetParams(suite.ctx)

	suite.Require().Equal(v2types.DefaultCreateMarketFee, retrievedParams.CreateMarketFee)
	suite.Require().Equal(v2types.DefaultMarketMakerFee, retrievedParams.MarketMakerFee)
	suite.Require().Equal(v2types.DefaultMarketTakerFee, retrievedParams.MarketTakerFee)
	suite.Require().Equal(v2types.DefaultMakerFeeDestination, retrievedParams.MakerFeeDestination)
	suite.Require().Equal(v2types.DefaultTakerFeeDestination, retrievedParams.TakerFeeDestination)
	suite.Require().Equal(v2types.DefaultPriceLevelOwnersWindow, retrievedParams.PriceLevelOwnersWindow)
	suite.Require().Equal(v2types.DefaultPriceLevelExtraGas, retrievedParams.PriceLevelExtraGas)
	suite.Require().Equal(v2types.DefaultFillOrdersExtraGas, retrievedParams.FillOrdersExtraGas)
	suite.Require().EqualValues(v2types.DefaultMinNativeLiquidityForModuleSwap, retrievedParams.MinNativeLiquidityForModuleSwap)
	suite.Require().Equal(v2types.DefaultMaxFillsPerOrder, retrievedParams.MaxFillsPerOrder)
}

func (suite *IntegrationTestSuite) TestParams_SetMultipleTimes() {
	params1 := v2types.DefaultParams()
	params1.CreateMarketFee = sdk.NewInt64Coin(denomBze, 500)
	params1.MarketMakerFee = sdk.NewInt64Coin(denomBze, 50)
	params1.MarketTakerFee = sdk.NewInt64Coin(denomBze, 100)
	params1.PriceLevelOwnersWindow = 150
	params1.PriceLevelExtraGas = 20000
	params1.FillOrdersExtraGas = 4000
	params1.MinNativeLiquidityForModuleSwap = math.NewInt(40000000000)

	params2 := v2types.DefaultParams()
	params2.CreateMarketFee = sdk.NewInt64Coin(denomBze, 2000)
	params2.MarketMakerFee = sdk.NewInt64Coin(denomBze, 200)
	params2.MarketTakerFee = sdk.NewInt64Coin(denomBze, 400)
	params2.MakerFeeDestination = v2types.FeeDestinationBurnerModule
	params2.TakerFeeDestination = v2types.FeeDestinationCommunityPool
	params2.PriceLevelOwnersWindow = 250
	params2.PriceLevelExtraGas = 35000
	params2.FillOrdersExtraGas = 7000
	params2.MinNativeLiquidityForModuleSwap = math.NewInt(70000000000)

	// Set first params
	err := suite.k.SetParams(suite.ctx, params1)
	suite.Require().NoError(err)

	retrieved1 := suite.k.GetParams(suite.ctx)
	suite.Require().Equal(params1.CreateMarketFee, retrieved1.CreateMarketFee)
	suite.Require().Equal(params1.MarketMakerFee, retrieved1.MarketMakerFee)

	// Set second params (update)
	err = suite.k.SetParams(suite.ctx, params2)
	suite.Require().NoError(err)

	retrieved2 := suite.k.GetParams(suite.ctx)
	suite.Require().Equal(params2.CreateMarketFee, retrieved2.CreateMarketFee)
	suite.Require().Equal(params2.MarketMakerFee, retrieved2.MarketMakerFee)
	suite.Require().Equal(params2.MarketTakerFee, retrieved2.MarketTakerFee)
	suite.Require().Equal(params2.MakerFeeDestination, retrieved2.MakerFeeDestination)
	suite.Require().Equal(params2.TakerFeeDestination, retrieved2.TakerFeeDestination)
	suite.Require().Equal(params2.PriceLevelOwnersWindow, retrieved2.PriceLevelOwnersWindow)
	suite.Require().Equal(params2.PriceLevelExtraGas, retrieved2.PriceLevelExtraGas)
	suite.Require().Equal(params2.FillOrdersExtraGas, retrieved2.FillOrdersExtraGas)
	suite.Require().Equal(params2.MinNativeLiquidityForModuleSwap, retrieved2.MinNativeLiquidityForModuleSwap)
}

func (suite *IntegrationTestSuite) TestParams_CreateMarketFee() {
	params := v2types.DefaultParams()
	params.CreateMarketFee = sdk.NewInt64Coin(denomBze, 1500)

	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	fee := suite.k.CreateMarketFee(suite.ctx)
	suite.Require().Equal(sdk.NewInt64Coin(denomBze, 1500), fee)
}

func (suite *IntegrationTestSuite) TestParams_MarketMakerFee() {
	params := v2types.DefaultParams()
	params.MarketMakerFee = sdk.NewInt64Coin(denomBze, 1500)

	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	fee := suite.k.MarketMakerFee(suite.ctx)
	suite.Require().Equal(sdk.NewInt64Coin(denomBze, 1500), fee)
}

func (suite *IntegrationTestSuite) TestParams_MarketTakerFee() {
	params := v2types.DefaultParams()
	params.MarketTakerFee = sdk.NewInt64Coin(denomBze, 2500)

	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	fee := suite.k.MarketTakerFee(suite.ctx)
	suite.Require().Equal(sdk.NewInt64Coin(denomBze, 2500), fee)
}

func (suite *IntegrationTestSuite) TestParams_MakerFeeDestination() {
	params := v2types.DefaultParams()
	params.MakerFeeDestination = v2types.FeeDestinationCommunityPool

	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	destination := suite.k.MakerFeeDestination(suite.ctx)
	suite.Require().Equal(v2types.FeeDestinationCommunityPool, destination)
}

func (suite *IntegrationTestSuite) TestParams_TakerFeeDestination() {
	params := v2types.DefaultParams()
	params.TakerFeeDestination = v2types.FeeDestinationCommunityPool

	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	destination := suite.k.TakerFeeDestination(suite.ctx)
	suite.Require().Equal(v2types.FeeDestinationCommunityPool, destination)
}

func (suite *IntegrationTestSuite) TestParams_AllGettersDefault() {
	createFee := suite.k.CreateMarketFee(suite.ctx)
	suite.Require().Equal(v2types.DefaultCreateMarketFee, createFee)

	makerFee := suite.k.MarketMakerFee(suite.ctx)
	suite.Require().Equal(v2types.DefaultMarketMakerFee, makerFee)

	takerFee := suite.k.MarketTakerFee(suite.ctx)
	suite.Require().Equal(v2types.DefaultMarketTakerFee, takerFee)

	makerDest := suite.k.MakerFeeDestination(suite.ctx)
	suite.Require().Equal(v2types.DefaultMakerFeeDestination, makerDest)

	takerDest := suite.k.TakerFeeDestination(suite.ctx)
	suite.Require().Equal(v2types.DefaultTakerFeeDestination, takerDest)
}

func (suite *IntegrationTestSuite) TestParams_ZeroFees() {
	params := v2types.DefaultParams()
	params.CreateMarketFee = sdk.NewInt64Coin(denomBze, 0)
	params.MarketMakerFee = sdk.NewInt64Coin(denomBze, 0)
	params.MarketTakerFee = sdk.NewInt64Coin(denomBze, 0)

	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	retrievedParams := suite.k.GetParams(suite.ctx)
	suite.Require().True(retrievedParams.CreateMarketFee.IsZero())
	suite.Require().True(retrievedParams.MarketMakerFee.IsZero())
	suite.Require().True(retrievedParams.MarketTakerFee.IsZero())
}

func (suite *IntegrationTestSuite) TestParams_UpdateIndividualFields() {
	initialParams := v2types.DefaultParams()
	initialParams.CreateMarketFee = sdk.NewInt64Coin(denomBze, 1000)
	initialParams.MarketMakerFee = sdk.NewInt64Coin(denomBze, 100)

	err := suite.k.SetParams(suite.ctx, initialParams)
	suite.Require().NoError(err)

	// Update only some fields
	updatedParams := initialParams
	updatedParams.CreateMarketFee = sdk.NewInt64Coin(denomBze, 1500)
	updatedParams.MarketMakerFee = sdk.NewInt64Coin(denomBze, 150)

	err = suite.k.SetParams(suite.ctx, updatedParams)
	suite.Require().NoError(err)

	retrievedParams := suite.k.GetParams(suite.ctx)
	suite.Require().Equal(sdk.NewInt64Coin(denomBze, 1500), retrievedParams.CreateMarketFee)
	suite.Require().Equal(sdk.NewInt64Coin(denomBze, 150), retrievedParams.MarketMakerFee)
	suite.Require().Equal(initialParams.MarketTakerFee, retrievedParams.MarketTakerFee)           // unchanged
	suite.Require().Equal(initialParams.MakerFeeDestination, retrievedParams.MakerFeeDestination) // unchanged
	suite.Require().Equal(initialParams.TakerFeeDestination, retrievedParams.TakerFeeDestination) // unchanged
}

func (suite *IntegrationTestSuite) TestParams_Persistence() {
	params := v2types.DefaultParams()
	params.CreateMarketFee = sdk.NewInt64Coin(denomBze, 800)
	params.MarketMakerFee = sdk.NewInt64Coin(denomBze, 80)
	params.MarketTakerFee = sdk.NewInt64Coin(denomBze, 160)
	params.PriceLevelOwnersWindow = 120
	params.PriceLevelExtraGas = 28000
	params.FillOrdersExtraGas = 5500
	params.MinNativeLiquidityForModuleSwap = math.NewInt(55000000000)

	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	for i := 0; i < 5; i++ {
		retrievedParams := suite.k.GetParams(suite.ctx)
		suite.Require().Equal(params.CreateMarketFee, retrievedParams.CreateMarketFee)
		suite.Require().Equal(params.MarketMakerFee, retrievedParams.MarketMakerFee)
		suite.Require().Equal(params.MarketTakerFee, retrievedParams.MarketTakerFee)
		suite.Require().Equal(params.MakerFeeDestination, retrievedParams.MakerFeeDestination)
		suite.Require().Equal(params.TakerFeeDestination, retrievedParams.TakerFeeDestination)
		suite.Require().Equal(params.PriceLevelOwnersWindow, retrievedParams.PriceLevelOwnersWindow)
		suite.Require().Equal(params.PriceLevelExtraGas, retrievedParams.PriceLevelExtraGas)
		suite.Require().Equal(params.FillOrdersExtraGas, retrievedParams.FillOrdersExtraGas)
		suite.Require().Equal(params.MinNativeLiquidityForModuleSwap, retrievedParams.MinNativeLiquidityForModuleSwap)
	}
}

func (suite *IntegrationTestSuite) TestParams_PriceLevelOwnersWindow() {
	params := v2types.DefaultParams()
	params.PriceLevelOwnersWindow = 150

	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	retrievedParams := suite.k.GetParams(suite.ctx)
	suite.Require().Equal(uint64(150), retrievedParams.PriceLevelOwnersWindow)
}

func (suite *IntegrationTestSuite) TestParams_PriceLevelExtraGas() {
	params := v2types.DefaultParams()
	params.PriceLevelExtraGas = 30000

	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	retrievedParams := suite.k.GetParams(suite.ctx)
	suite.Require().Equal(uint64(30000), retrievedParams.PriceLevelExtraGas)
}

func (suite *IntegrationTestSuite) TestParams_FillOrdersExtraGas() {
	params := v2types.DefaultParams()
	params.FillOrdersExtraGas = 6000

	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	retrievedParams := suite.k.GetParams(suite.ctx)
	suite.Require().Equal(uint64(6000), retrievedParams.FillOrdersExtraGas)
}

func (suite *IntegrationTestSuite) TestParams_MinNativeLiquidityForModuleSwap() {
	params := v2types.DefaultParams()
	params.MinNativeLiquidityForModuleSwap = math.NewInt(75000000000)

	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	retrievedParams := suite.k.GetParams(suite.ctx)
	suite.Require().Equal("75000000000", retrievedParams.MinNativeLiquidityForModuleSwap.String())
}

func (suite *IntegrationTestSuite) TestParams_MaxFillsPerOrder() {
	params := v2types.DefaultParams()
	params.MaxFillsPerOrder = 1000

	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	retrievedParams := suite.k.GetParams(suite.ctx)
	suite.Require().Equal(uint64(1000), retrievedParams.MaxFillsPerOrder)
}

func (suite *IntegrationTestSuite) TestParams_GetDefault_AllFields() {
	retrievedParams := suite.k.GetParams(suite.ctx)

	suite.Require().Equal(v2types.DefaultCreateMarketFee, retrievedParams.CreateMarketFee)
	suite.Require().Equal(v2types.DefaultMarketMakerFee, retrievedParams.MarketMakerFee)
	suite.Require().Equal(v2types.DefaultMarketTakerFee, retrievedParams.MarketTakerFee)
	suite.Require().Equal(v2types.DefaultMakerFeeDestination, retrievedParams.MakerFeeDestination)
	suite.Require().Equal(v2types.DefaultTakerFeeDestination, retrievedParams.TakerFeeDestination)
	suite.Require().Equal(v2types.DefaultPriceLevelOwnersWindow, retrievedParams.PriceLevelOwnersWindow)
	suite.Require().Equal(v2types.DefaultPriceLevelExtraGas, retrievedParams.PriceLevelExtraGas)
	suite.Require().Equal(v2types.DefaultFillOrdersExtraGas, retrievedParams.FillOrdersExtraGas)
	suite.Require().EqualValues(v2types.DefaultMinNativeLiquidityForModuleSwap, retrievedParams.MinNativeLiquidityForModuleSwap)
	suite.Require().Equal(v2types.DefaultMaxFillsPerOrder, retrievedParams.MaxFillsPerOrder)
}
