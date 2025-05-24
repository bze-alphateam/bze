package keeper_test

import (
	"cosmossdk.io/math"
	"github.com/bze-alphateam/bze/x/rewards/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (suite *IntegrationTestSuite) TestParams_GetAndSet() {
	params := types.Params{
		CreateStakingRewardFee: sdk.NewCoin("ubze", math.NewInt(1000)),
		CreateTradingRewardFee: sdk.NewCoin("ubze", math.NewInt(2000)),
	}

	// Test SetParams
	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	// Test GetParams
	retrievedParams := suite.k.GetParams(suite.ctx)
	suite.Require().Equal(params.CreateStakingRewardFee, retrievedParams.CreateStakingRewardFee)
	suite.Require().Equal(params.CreateTradingRewardFee, retrievedParams.CreateTradingRewardFee)
}

func (suite *IntegrationTestSuite) TestParams_GetDefault() {
	// Test GetParams when no params are set (should return zero values)
	retrievedParams := suite.k.GetParams(suite.ctx)

	// Default values should be empty/zero
	suite.Require().Equal(types.DefaultCreateRewardFee, retrievedParams.CreateStakingRewardFee)
	suite.Require().Equal(types.DefaultCreateRewardFee, retrievedParams.CreateTradingRewardFee)
}

func (suite *IntegrationTestSuite) TestParams_SetMultipleTimes() {
	// Test setting params multiple times
	params1 := types.Params{
		CreateStakingRewardFee: sdk.NewCoin("ubze", math.NewInt(500)),
		CreateTradingRewardFee: sdk.NewCoin("ubze", math.NewInt(1000)),
	}

	params2 := types.Params{
		CreateStakingRewardFee: sdk.NewCoin("utoken", math.NewInt(750)),
		CreateTradingRewardFee: sdk.NewCoin("utoken", math.NewInt(1500)),
	}

	// Set first params
	err := suite.k.SetParams(suite.ctx, params1)
	suite.Require().NoError(err)

	// Verify first params
	retrieved1 := suite.k.GetParams(suite.ctx)
	suite.Require().Equal(params1.CreateStakingRewardFee, retrieved1.CreateStakingRewardFee)
	suite.Require().Equal(params1.CreateTradingRewardFee, retrieved1.CreateTradingRewardFee)

	// Set second params (update)
	err = suite.k.SetParams(suite.ctx, params2)
	suite.Require().NoError(err)

	// Verify second params
	retrieved2 := suite.k.GetParams(suite.ctx)
	suite.Require().Equal(params2.CreateStakingRewardFee, retrieved2.CreateStakingRewardFee)
	suite.Require().Equal(params2.CreateTradingRewardFee, retrieved2.CreateTradingRewardFee)
}

func (suite *IntegrationTestSuite) TestParams_SetZeroValues() {
	// Test setting params with zero values
	params := types.Params{
		CreateStakingRewardFee: sdk.NewCoin("ubze", math.ZeroInt()),
		CreateTradingRewardFee: sdk.NewCoin("ubze", math.ZeroInt()),
	}

	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	retrievedParams := suite.k.GetParams(suite.ctx)
	suite.Require().Equal("ubze", retrievedParams.CreateStakingRewardFee.Denom)
	suite.Require().Equal(math.ZeroInt(), retrievedParams.CreateStakingRewardFee.Amount)
	suite.Require().Equal("ubze", retrievedParams.CreateTradingRewardFee.Denom)
	suite.Require().Equal(math.ZeroInt(), retrievedParams.CreateTradingRewardFee.Amount)
}

func (suite *IntegrationTestSuite) TestParams_SetDifferentDenominations() {
	// Test setting params with different denominations
	params := types.Params{
		CreateStakingRewardFee: sdk.NewCoin("ubze", math.NewInt(1000)),
		CreateTradingRewardFee: sdk.NewCoin("utoken", math.NewInt(2000)),
	}

	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	retrievedParams := suite.k.GetParams(suite.ctx)
	suite.Require().Equal("ubze", retrievedParams.CreateStakingRewardFee.Denom)
	suite.Require().Equal(math.NewInt(1000), retrievedParams.CreateStakingRewardFee.Amount)
	suite.Require().Equal("utoken", retrievedParams.CreateTradingRewardFee.Denom)
	suite.Require().Equal(math.NewInt(2000), retrievedParams.CreateTradingRewardFee.Amount)
}

func (suite *IntegrationTestSuite) TestParams_SetLargeValues() {
	// Test setting params with large values
	largeAmount := math.NewIntFromUint64(18446744073709551615) // Max uint64

	params := types.Params{
		CreateStakingRewardFee: sdk.NewCoin("ubze", largeAmount),
		CreateTradingRewardFee: sdk.NewCoin("utoken", largeAmount),
	}

	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	retrievedParams := suite.k.GetParams(suite.ctx)
	suite.Require().Equal(largeAmount, retrievedParams.CreateStakingRewardFee.Amount)
	suite.Require().Equal(largeAmount, retrievedParams.CreateTradingRewardFee.Amount)
}

func (suite *IntegrationTestSuite) TestParams_SetSameFees() {
	// Test setting both fees to the same value
	sameFee := sdk.NewCoin("ubze", math.NewInt(1500))

	params := types.Params{
		CreateStakingRewardFee: sameFee,
		CreateTradingRewardFee: sameFee,
	}

	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	retrievedParams := suite.k.GetParams(suite.ctx)
	suite.Require().Equal(sameFee, retrievedParams.CreateStakingRewardFee)
	suite.Require().Equal(sameFee, retrievedParams.CreateTradingRewardFee)
}

func (suite *IntegrationTestSuite) TestParams_UpdateIndividualFields() {
	// Test updating params by changing individual fields
	initialParams := types.Params{
		CreateStakingRewardFee: sdk.NewCoin("ubze", math.NewInt(1000)),
		CreateTradingRewardFee: sdk.NewCoin("ubze", math.NewInt(2000)),
	}

	// Set initial params
	err := suite.k.SetParams(suite.ctx, initialParams)
	suite.Require().NoError(err)

	// Update only staking fee
	updatedParams := initialParams
	updatedParams.CreateStakingRewardFee = sdk.NewCoin("ubze", math.NewInt(1500))

	err = suite.k.SetParams(suite.ctx, updatedParams)
	suite.Require().NoError(err)

	// Verify update
	retrievedParams := suite.k.GetParams(suite.ctx)
	suite.Require().Equal(math.NewInt(1500), retrievedParams.CreateStakingRewardFee.Amount)
	suite.Require().Equal(math.NewInt(2000), retrievedParams.CreateTradingRewardFee.Amount) // Should remain unchanged
}

func (suite *IntegrationTestSuite) TestParams_Persistence() {
	// Test that params persist across multiple get operations
	params := types.Params{
		CreateStakingRewardFee: sdk.NewCoin("ubze", math.NewInt(800)),
		CreateTradingRewardFee: sdk.NewCoin("utoken", math.NewInt(1200)),
	}

	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	// Get params multiple times
	for i := 0; i < 5; i++ {
		retrievedParams := suite.k.GetParams(suite.ctx)
		suite.Require().Equal(params.CreateStakingRewardFee, retrievedParams.CreateStakingRewardFee)
		suite.Require().Equal(params.CreateTradingRewardFee, retrievedParams.CreateTradingRewardFee)
	}
}

func (suite *IntegrationTestSuite) TestParams_EmptyDenominations() {
	// Test setting params with empty denominations (edge case)
	params := types.Params{
		CreateStakingRewardFee: sdk.Coin{Denom: "", Amount: math.NewInt(1000)},
		CreateTradingRewardFee: sdk.Coin{Denom: "", Amount: math.NewInt(2000)},
	}

	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	retrievedParams := suite.k.GetParams(suite.ctx)
	suite.Require().Equal("", retrievedParams.CreateStakingRewardFee.Denom)
	suite.Require().Equal(math.NewInt(1000), retrievedParams.CreateStakingRewardFee.Amount)
	suite.Require().Equal("", retrievedParams.CreateTradingRewardFee.Denom)
	suite.Require().Equal(math.NewInt(2000), retrievedParams.CreateTradingRewardFee.Amount)
}
