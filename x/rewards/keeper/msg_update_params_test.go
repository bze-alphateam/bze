package keeper_test

import (
	"cosmossdk.io/math"
	"github.com/bze-alphateam/bze/x/rewards/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
)

// validUpdateParams returns a fully valid Params value to mutate per test.
// MsgUpdateParams replaces the WHOLE params object, so every field must pass
// validation — partial params (unset boost fields) are rejected.
func validUpdateParams() types.Params {
	return types.DefaultParams()
}

func (suite *IntegrationTestSuite) TestMsgUpdateParams_ValidAuthority() {
	authority := authtypes.NewModuleAddress(govtypes.ModuleName).String()
	params := validUpdateParams()
	params.CreateStakingRewardFee = sdk.NewCoin("ubze", math.NewInt(1000))
	params.CreateTradingRewardFee = sdk.NewCoin("ubze", math.NewInt(2000))

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
	msg := &types.MsgUpdateParams{
		Authority: "bze1invalidauthority",
		Params:    validUpdateParams(),
	}

	response, err := suite.msgServer.UpdateParams(suite.ctx, msg)
	suite.Require().Error(err)
	suite.Require().Nil(response)
	suite.Require().Contains(err.Error(), "invalid authority")
}

func (suite *IntegrationTestSuite) TestMsgUpdateParams_EmptyAuthority() {
	msg := &types.MsgUpdateParams{
		Authority: "",
		Params:    validUpdateParams(),
	}

	response, err := suite.msgServer.UpdateParams(suite.ctx, msg)
	suite.Require().Error(err)
	suite.Require().Nil(response)
	suite.Require().Contains(err.Error(), "invalid authority")
}

func (suite *IntegrationTestSuite) TestMsgUpdateParams_ZeroFees() {
	authority := authtypes.NewModuleAddress(govtypes.ModuleName).String()
	params := validUpdateParams()
	params.CreateStakingRewardFee = sdk.NewCoin("ubze", math.ZeroInt())
	params.CreateTradingRewardFee = sdk.NewCoin("ubze", math.ZeroInt())

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
	params := validUpdateParams()
	params.CreateStakingRewardFee = sdk.NewCoin("ubze", math.NewInt(500))
	params.CreateTradingRewardFee = sdk.NewCoin("utoken", math.NewInt(1000))

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

	params := validUpdateParams()
	params.CreateStakingRewardFee = sdk.NewCoin("ubze", largeAmount)
	params.CreateTradingRewardFee = sdk.NewCoin("ubze", largeAmount)

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
	params1 := validUpdateParams()
	params1.CreateStakingRewardFee = sdk.NewCoin("ubze", math.NewInt(100))
	params1.CreateTradingRewardFee = sdk.NewCoin("ubze", math.NewInt(200))

	response1, err := suite.msgServer.UpdateParams(suite.ctx, &types.MsgUpdateParams{
		Authority: authority,
		Params:    params1,
	})
	suite.Require().NoError(err)
	suite.Require().NotNil(response1)

	// Verify first update
	retrievedParams1 := suite.k.GetParams(suite.ctx)
	suite.Require().Equal(params1.CreateStakingRewardFee, retrievedParams1.CreateStakingRewardFee)

	// Second update
	params2 := validUpdateParams()
	params2.CreateStakingRewardFee = sdk.NewCoin("utoken", math.NewInt(300))
	params2.CreateTradingRewardFee = sdk.NewCoin("utoken", math.NewInt(400))

	response2, err := suite.msgServer.UpdateParams(suite.ctx, &types.MsgUpdateParams{
		Authority: authority,
		Params:    params2,
	})
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
	initialParams := validUpdateParams()
	initialParams.CreateStakingRewardFee = sdk.NewCoin("ubze", math.NewInt(100))
	initialParams.CreateTradingRewardFee = sdk.NewCoin("ubze", math.NewInt(200))

	err := suite.k.SetParams(suite.ctx, initialParams)
	suite.Require().NoError(err)

	// Update with new params (the whole object must be provided)
	updatedParams := initialParams
	updatedParams.CreateStakingRewardFee = sdk.NewCoin("ubze", math.NewInt(150))

	response, err := suite.msgServer.UpdateParams(suite.ctx, &types.MsgUpdateParams{
		Authority: authority,
		Params:    updatedParams,
	})
	suite.Require().NoError(err)
	suite.Require().NotNil(response)

	// Verify update
	retrievedParams := suite.k.GetParams(suite.ctx)
	suite.Require().Equal(math.NewInt(150), retrievedParams.CreateStakingRewardFee.Amount)
	suite.Require().Equal(math.NewInt(200), retrievedParams.CreateTradingRewardFee.Amount)
}

// TestMsgUpdateParams_EmptyParamsRejected: an all-zero Params object no longer
// slips past the handler — validation runs before the store write.
func (suite *IntegrationTestSuite) TestMsgUpdateParams_EmptyParamsRejected() {
	authority := authtypes.NewModuleAddress(govtypes.ModuleName).String()
	before := suite.k.GetParams(suite.ctx)

	response, err := suite.msgServer.UpdateParams(suite.ctx, &types.MsgUpdateParams{
		Authority: authority,
		Params:    types.Params{},
	})
	suite.Require().Error(err)
	suite.Require().Nil(response)

	// stored params untouched
	suite.Require().Equal(before, suite.k.GetParams(suite.ctx))
}

// TestMsgUpdateParams_InvalidParamsRejected: params violating a single
// invariant are rejected even with a valid authority — a zero
// CleanupBatchSize would brick MsgCleanupBoost (its batch-limit logic assumes
// limit >= 1), so it must never reach the store.
func (suite *IntegrationTestSuite) TestMsgUpdateParams_InvalidParamsRejected() {
	authority := authtypes.NewModuleAddress(govtypes.ModuleName).String()
	before := suite.k.GetParams(suite.ctx)

	invalid := validUpdateParams()
	invalid.CleanupBatchSize = 0

	response, err := suite.msgServer.UpdateParams(suite.ctx, &types.MsgUpdateParams{
		Authority: authority,
		Params:    invalid,
	})
	suite.Require().Error(err)
	suite.Require().Nil(response)
	suite.Require().Contains(err.Error(), "CleanupBatchSize")

	// stored params untouched
	suite.Require().Equal(before, suite.k.GetParams(suite.ctx))
}
