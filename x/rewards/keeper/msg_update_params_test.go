package keeper_test

import (
	"cosmossdk.io/math"
	"github.com/bze-alphateam/bze/x/rewards/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
)

func (suite *IntegrationTestSuite) TestMsgUpdateParams_ValidAuthority() {
	authority := authtypes.NewModuleAddress(govtypes.ModuleName).String()
	params := types.Params{
		CreateStakingRewardFee: sdk.NewCoin("ubze", math.NewInt(1000)),
		CreateTradingRewardFee: sdk.NewCoin("ubze", math.NewInt(2000)),
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
	suite.Require().Equal(params.CreateStakingRewardFee, retrievedParams.CreateStakingRewardFee)
	suite.Require().Equal(params.CreateTradingRewardFee, retrievedParams.CreateTradingRewardFee)
}

func (suite *IntegrationTestSuite) TestMsgUpdateParams_InvalidAuthority() {
	invalidAuthority := "bze1invalidauthority"
	params := types.Params{
		CreateStakingRewardFee: sdk.NewCoin("ubze", math.NewInt(1000)),
		CreateTradingRewardFee: sdk.NewCoin("ubze", math.NewInt(2000)),
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
		CreateStakingRewardFee: sdk.NewCoin("ubze", math.NewInt(1000)),
		CreateTradingRewardFee: sdk.NewCoin("ubze", math.NewInt(2000)),
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
		CreateStakingRewardFee: sdk.NewCoin("ubze", math.ZeroInt()),
		CreateTradingRewardFee: sdk.NewCoin("ubze", math.ZeroInt()),
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
	suite.Require().Equal(math.ZeroInt(), retrievedParams.CreateStakingRewardFee.Amount)
	suite.Require().Equal(math.ZeroInt(), retrievedParams.CreateTradingRewardFee.Amount)
}

func (suite *IntegrationTestSuite) TestMsgUpdateParams_DifferentDenominations() {
	authority := authtypes.NewModuleAddress(govtypes.ModuleName).String()
	params := types.Params{
		CreateStakingRewardFee: sdk.NewCoin("ubze", math.NewInt(500)),
		CreateTradingRewardFee: sdk.NewCoin("utoken", math.NewInt(1000)),
	}

	msg := &types.MsgUpdateParams{
		Authority: authority,
		Params:    params,
	}

	response, err := suite.msgServer.UpdateParams(suite.ctx, msg)
	suite.Require().NoError(err)
	suite.Require().NotNil(response)

	// Verify different denominations were set
	retrievedParams := suite.k.GetParams(suite.ctx)
	suite.Require().Equal("ubze", retrievedParams.CreateStakingRewardFee.Denom)
	suite.Require().Equal("utoken", retrievedParams.CreateTradingRewardFee.Denom)
}

func (suite *IntegrationTestSuite) TestMsgUpdateParams_LargeFees() {
	authority := authtypes.NewModuleAddress(govtypes.ModuleName).String()
	largeAmount := math.NewIntFromUint64(18446744073709551615) // Max uint64

	params := types.Params{
		CreateStakingRewardFee: sdk.NewCoin("ubze", largeAmount),
		CreateTradingRewardFee: sdk.NewCoin("ubze", largeAmount),
	}

	msg := &types.MsgUpdateParams{
		Authority: authority,
		Params:    params,
	}

	response, err := suite.msgServer.UpdateParams(suite.ctx, msg)
	suite.Require().NoError(err)
	suite.Require().NotNil(response)

	// Verify large fees were set
	retrievedParams := suite.k.GetParams(suite.ctx)
	suite.Require().Equal(largeAmount, retrievedParams.CreateStakingRewardFee.Amount)
	suite.Require().Equal(largeAmount, retrievedParams.CreateTradingRewardFee.Amount)
}

func (suite *IntegrationTestSuite) TestMsgUpdateParams_MultipleUpdates() {
	authority := authtypes.NewModuleAddress(govtypes.ModuleName).String()

	// First update
	params1 := types.Params{
		CreateStakingRewardFee: sdk.NewCoin("ubze", math.NewInt(100)),
		CreateTradingRewardFee: sdk.NewCoin("ubze", math.NewInt(200)),
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
	suite.Require().Equal(params1.CreateStakingRewardFee, retrievedParams1.CreateStakingRewardFee)

	// Second update
	params2 := types.Params{
		CreateStakingRewardFee: sdk.NewCoin("utoken", math.NewInt(300)),
		CreateTradingRewardFee: sdk.NewCoin("utoken", math.NewInt(400)),
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
	suite.Require().Equal(params2.CreateStakingRewardFee, retrievedParams2.CreateStakingRewardFee)
	suite.Require().Equal(params2.CreateTradingRewardFee, retrievedParams2.CreateTradingRewardFee)
}

func (suite *IntegrationTestSuite) TestMsgUpdateParams_PartialUpdate() {
	authority := authtypes.NewModuleAddress(govtypes.ModuleName).String()

	// Set initial params
	initialParams := types.Params{
		CreateStakingRewardFee: sdk.NewCoin("ubze", math.NewInt(100)),
		CreateTradingRewardFee: sdk.NewCoin("ubze", math.NewInt(200)),
	}

	err := suite.k.SetParams(suite.ctx, initialParams)
	suite.Require().NoError(err)

	// Update with new params (both fields must be provided)
	updatedParams := types.Params{
		CreateStakingRewardFee: sdk.NewCoin("ubze", math.NewInt(150)),
		CreateTradingRewardFee: sdk.NewCoin("ubze", math.NewInt(200)), // Keep same
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
	suite.Require().Equal(math.NewInt(150), retrievedParams.CreateStakingRewardFee.Amount)
	suite.Require().Equal(math.NewInt(200), retrievedParams.CreateTradingRewardFee.Amount)
}

func (suite *IntegrationTestSuite) TestMsgUpdateParams_NilParams() {
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
	suite.Require().Equal("", retrievedParams.CreateStakingRewardFee.Denom)
	suite.Require().Equal(math.ZeroInt(), retrievedParams.CreateStakingRewardFee.Amount)
}
