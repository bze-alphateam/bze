package keeper_test

import (
	"github.com/bze-alphateam/bze/x/tradebin/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
)

func (suite *IntegrationTestSuite) TestMsgUpdateParams_ValidAuthority() {
	authority := authtypes.NewModuleAddress(govtypes.ModuleName).String()
	params := types.Params{
		CreateMarketFee:     "1000",
		MarketMakerFee:      "0.001",
		MarketTakerFee:      "0.002",
		MakerFeeDestination: "community_pool",
		TakerFeeDestination: "burn",
	}

	msg := &types.MsgUpdateParams{
		Authority: authority,
		Params:    params,
	}

	response, err := suite.msgServer.UpdateParams(suite.ctx, msg)
	suite.Require().NoError(err)
	suite.Require().NotNil(response)

	// Verify params were updated
	retrievedParams := suite.k.GetParams(suite.ctx)
	suite.Require().Equal(params.CreateMarketFee, retrievedParams.CreateMarketFee)
	suite.Require().Equal(params.MarketMakerFee, retrievedParams.MarketMakerFee)
	suite.Require().Equal(params.MarketTakerFee, retrievedParams.MarketTakerFee)
	suite.Require().Equal(params.MakerFeeDestination, retrievedParams.MakerFeeDestination)
	suite.Require().Equal(params.TakerFeeDestination, retrievedParams.TakerFeeDestination)
}

func (suite *IntegrationTestSuite) TestMsgUpdateParams_InvalidAuthority() {
	invalidAuthority := "bze1invalidauthority"
	params := types.Params{
		CreateMarketFee:     "1000",
		MarketMakerFee:      "0.001",
		MarketTakerFee:      "0.002",
		MakerFeeDestination: "community_pool",
		TakerFeeDestination: "burn",
	}

	msg := &types.MsgUpdateParams{
		Authority: invalidAuthority,
		Params:    params,
	}

	response, err := suite.msgServer.UpdateParams(suite.ctx, msg)
	suite.Require().Error(err)
	suite.Require().Nil(response)
	suite.Require().Contains(err.Error(), "invalid authority")
}

func (suite *IntegrationTestSuite) TestMsgUpdateParams_EmptyAuthority() {
	params := types.Params{
		CreateMarketFee:     "1000",
		MarketMakerFee:      "0.001",
		MarketTakerFee:      "0.002",
		MakerFeeDestination: "community_pool",
		TakerFeeDestination: "burn",
	}

	msg := &types.MsgUpdateParams{
		Authority: "",
		Params:    params,
	}

	response, err := suite.msgServer.UpdateParams(suite.ctx, msg)
	suite.Require().Error(err)
	suite.Require().Nil(response)
	suite.Require().Contains(err.Error(), "invalid authority")
}

func (suite *IntegrationTestSuite) TestMsgUpdateParams_ZeroFees() {
	authority := authtypes.NewModuleAddress(govtypes.ModuleName).String()
	params := types.Params{
		CreateMarketFee:     "0",
		MarketMakerFee:      "0",
		MarketTakerFee:      "0",
		MakerFeeDestination: "",
		TakerFeeDestination: "",
	}

	msg := &types.MsgUpdateParams{
		Authority: authority,
		Params:    params,
	}

	response, err := suite.msgServer.UpdateParams(suite.ctx, msg)
	suite.Require().NoError(err)
	suite.Require().NotNil(response)

	// Verify zero fees were set
	retrievedParams := suite.k.GetParams(suite.ctx)
	suite.Require().Equal("0", retrievedParams.CreateMarketFee)
	suite.Require().Equal("0", retrievedParams.MarketMakerFee)
	suite.Require().Equal("0", retrievedParams.MarketTakerFee)
	suite.Require().Equal("", retrievedParams.MakerFeeDestination)
	suite.Require().Equal("", retrievedParams.TakerFeeDestination)
}

func (suite *IntegrationTestSuite) TestMsgUpdateParams_DifferentDestinations() {
	authority := authtypes.NewModuleAddress(govtypes.ModuleName).String()
	params := types.Params{
		CreateMarketFee:     "500",
		MarketMakerFee:      "0.0005",
		MarketTakerFee:      "0.001",
		MakerFeeDestination: "validator_rewards",
		TakerFeeDestination: "community_pool",
	}

	msg := &types.MsgUpdateParams{
		Authority: authority,
		Params:    params,
	}

	response, err := suite.msgServer.UpdateParams(suite.ctx, msg)
	suite.Require().NoError(err)
	suite.Require().NotNil(response)

	// Verify different destinations were set
	retrievedParams := suite.k.GetParams(suite.ctx)
	suite.Require().Equal("validator_rewards", retrievedParams.MakerFeeDestination)
	suite.Require().Equal("community_pool", retrievedParams.TakerFeeDestination)
}

func (suite *IntegrationTestSuite) TestMsgUpdateParams_HighFees() {
	authority := authtypes.NewModuleAddress(govtypes.ModuleName).String()
	params := types.Params{
		CreateMarketFee:     "10000",
		MarketMakerFee:      "0.01", // 1%
		MarketTakerFee:      "0.02", // 2%
		MakerFeeDestination: "burn",
		TakerFeeDestination: "burn",
	}

	msg := &types.MsgUpdateParams{
		Authority: authority,
		Params:    params,
	}

	response, err := suite.msgServer.UpdateParams(suite.ctx, msg)
	suite.Require().NoError(err)
	suite.Require().NotNil(response)

	// Verify high fees were set
	retrievedParams := suite.k.GetParams(suite.ctx)
	suite.Require().Equal("10000", retrievedParams.CreateMarketFee)
	suite.Require().Equal("0.01", retrievedParams.MarketMakerFee)
	suite.Require().Equal("0.02", retrievedParams.MarketTakerFee)
}

func (suite *IntegrationTestSuite) TestMsgUpdateParams_MultipleUpdates() {
	authority := authtypes.NewModuleAddress(govtypes.ModuleName).String()

	// First update
	params1 := types.Params{
		CreateMarketFee:     "100",
		MarketMakerFee:      "0.0001",
		MarketTakerFee:      "0.0002",
		MakerFeeDestination: "community_pool",
		TakerFeeDestination: "burn",
	}

	msg1 := &types.MsgUpdateParams{
		Authority: authority,
		Params:    params1,
	}

	response1, err := suite.msgServer.UpdateParams(suite.ctx, msg1)
	suite.Require().NoError(err)
	suite.Require().NotNil(response1)

	// Verify first update
	retrievedParams1 := suite.k.GetParams(suite.ctx)
	suite.Require().Equal(params1.CreateMarketFee, retrievedParams1.CreateMarketFee)

	// Second update
	params2 := types.Params{
		CreateMarketFee:     "300",
		MarketMakerFee:      "0.0003",
		MarketTakerFee:      "0.0006",
		MakerFeeDestination: "validator_rewards",
		TakerFeeDestination: "community_pool",
	}

	msg2 := &types.MsgUpdateParams{
		Authority: authority,
		Params:    params2,
	}

	response2, err := suite.msgServer.UpdateParams(suite.ctx, msg2)
	suite.Require().NoError(err)
	suite.Require().NotNil(response2)

	// Verify second update
	retrievedParams2 := suite.k.GetParams(suite.ctx)
	suite.Require().Equal(params2.CreateMarketFee, retrievedParams2.CreateMarketFee)
	suite.Require().Equal(params2.MarketMakerFee, retrievedParams2.MarketMakerFee)
	suite.Require().Equal(params2.MarketTakerFee, retrievedParams2.MarketTakerFee)
	suite.Require().Equal(params2.MakerFeeDestination, retrievedParams2.MakerFeeDestination)
	suite.Require().Equal(params2.TakerFeeDestination, retrievedParams2.TakerFeeDestination)
}

func (suite *IntegrationTestSuite) TestMsgUpdateParams_PartialUpdate() {
	authority := authtypes.NewModuleAddress(govtypes.ModuleName).String()

	// Set initial params
	initialParams := types.Params{
		CreateMarketFee:     "100",
		MarketMakerFee:      "0.001",
		MarketTakerFee:      "0.002",
		MakerFeeDestination: "community_pool",
		TakerFeeDestination: "burn",
	}

	err := suite.k.SetParams(suite.ctx, initialParams)
	suite.Require().NoError(err)

	// Update with new params (all fields must be provided)
	updatedParams := types.Params{
		CreateMarketFee:     "150",               // Changed
		MarketMakerFee:      "0.001",             // Keep same
		MarketTakerFee:      "0.002",             // Keep same
		MakerFeeDestination: "validator_rewards", // Changed
		TakerFeeDestination: "burn",              // Keep same
	}

	msg := &types.MsgUpdateParams{
		Authority: authority,
		Params:    updatedParams,
	}

	response, err := suite.msgServer.UpdateParams(suite.ctx, msg)
	suite.Require().NoError(err)
	suite.Require().NotNil(response)

	// Verify update
	retrievedParams := suite.k.GetParams(suite.ctx)
	suite.Require().Equal("150", retrievedParams.CreateMarketFee)
	suite.Require().Equal("0.001", retrievedParams.MarketMakerFee)
	suite.Require().Equal("0.002", retrievedParams.MarketTakerFee)
	suite.Require().Equal("validator_rewards", retrievedParams.MakerFeeDestination)
	suite.Require().Equal("burn", retrievedParams.TakerFeeDestination)
}

func (suite *IntegrationTestSuite) TestMsgUpdateParams_EmptyParams() {
	authority := authtypes.NewModuleAddress(govtypes.ModuleName).String()

	msg := &types.MsgUpdateParams{
		Authority: authority,
		Params:    types.Params{}, // Empty params
	}

	response, err := suite.msgServer.UpdateParams(suite.ctx, msg)
	suite.Require().NoError(err)
	suite.Require().NotNil(response)

	// Verify empty params were set
	retrievedParams := suite.k.GetParams(suite.ctx)
	suite.Require().Equal("", retrievedParams.CreateMarketFee)
	suite.Require().Equal("", retrievedParams.MarketMakerFee)
	suite.Require().Equal("", retrievedParams.MarketTakerFee)
	suite.Require().Equal("", retrievedParams.MakerFeeDestination)
	suite.Require().Equal("", retrievedParams.TakerFeeDestination)
}
