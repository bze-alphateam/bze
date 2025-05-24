package keeper_test

import (
	"cosmossdk.io/math"
	"github.com/bze-alphateam/bze/x/rewards/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (suite *IntegrationTestSuite) TestQueryParams_ValidRequest() {
	// Set some params first
	params := types.Params{
		CreateStakingRewardFee: sdk.NewCoin("ubze", math.NewInt(1000)),
		CreateTradingRewardFee: sdk.NewCoin("ubze", math.NewInt(2000)),
	}

	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	// Query params
	req := &types.QueryParamsRequest{}
	response, err := suite.k.Params(suite.ctx, req)

	suite.Require().NoError(err)
	suite.Require().NotNil(response)
	suite.Require().Equal(params.CreateStakingRewardFee, response.Params.CreateStakingRewardFee)
	suite.Require().Equal(params.CreateTradingRewardFee, response.Params.CreateTradingRewardFee)
}

func (suite *IntegrationTestSuite) TestQueryParams_NilRequest() {
	response, err := suite.k.Params(suite.ctx, nil)

	suite.Require().Error(err)
	suite.Require().Nil(response)
	suite.Require().Equal(codes.InvalidArgument, status.Code(err))
	suite.Require().Contains(err.Error(), "invalid request")
}

func (suite *IntegrationTestSuite) TestQueryParams_DefaultParams() {
	// Query params without setting any (should return default/zero values)
	req := &types.QueryParamsRequest{}
	response, err := suite.k.Params(suite.ctx, req)

	suite.Require().NoError(err)
	suite.Require().NotNil(response)
	suite.Require().Equal(types.DefaultCreateRewardFee, response.Params.CreateStakingRewardFee)
	suite.Require().Equal(types.DefaultCreateRewardFee, response.Params.CreateTradingRewardFee)
}

func (suite *IntegrationTestSuite) TestQueryParams_ZeroParams() {
	// Set zero value params
	params := types.Params{
		CreateStakingRewardFee: sdk.NewCoin("ubze", math.ZeroInt()),
		CreateTradingRewardFee: sdk.NewCoin("ubze", math.ZeroInt()),
	}

	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	// Query params
	req := &types.QueryParamsRequest{}
	response, err := suite.k.Params(suite.ctx, req)

	suite.Require().NoError(err)
	suite.Require().NotNil(response)
	suite.Require().Equal("ubze", response.Params.CreateStakingRewardFee.Denom)
	suite.Require().Equal(math.ZeroInt(), response.Params.CreateStakingRewardFee.Amount)
	suite.Require().Equal("ubze", response.Params.CreateTradingRewardFee.Denom)
	suite.Require().Equal(math.ZeroInt(), response.Params.CreateTradingRewardFee.Amount)
}

func (suite *IntegrationTestSuite) TestQueryParams_DifferentDenominations() {
	// Set params with different denominations
	params := types.Params{
		CreateStakingRewardFee: sdk.NewCoin("ubze", math.NewInt(500)),
		CreateTradingRewardFee: sdk.NewCoin("utoken", math.NewInt(1000)),
	}

	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	// Query params
	req := &types.QueryParamsRequest{}
	response, err := suite.k.Params(suite.ctx, req)

	suite.Require().NoError(err)
	suite.Require().NotNil(response)
	suite.Require().Equal("ubze", response.Params.CreateStakingRewardFee.Denom)
	suite.Require().Equal(math.NewInt(500), response.Params.CreateStakingRewardFee.Amount)
	suite.Require().Equal("utoken", response.Params.CreateTradingRewardFee.Denom)
	suite.Require().Equal(math.NewInt(1000), response.Params.CreateTradingRewardFee.Amount)
}

func (suite *IntegrationTestSuite) TestQueryParams_LargeValues() {
	// Set params with large values
	largeAmount := math.NewIntFromUint64(18446744073709551615) // Max uint64
	params := types.Params{
		CreateStakingRewardFee: sdk.NewCoin("ubze", largeAmount),
		CreateTradingRewardFee: sdk.NewCoin("utoken", largeAmount),
	}

	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	// Query params
	req := &types.QueryParamsRequest{}
	response, err := suite.k.Params(suite.ctx, req)

	suite.Require().NoError(err)
	suite.Require().NotNil(response)
	suite.Require().Equal(largeAmount, response.Params.CreateStakingRewardFee.Amount)
	suite.Require().Equal(largeAmount, response.Params.CreateTradingRewardFee.Amount)
}

func (suite *IntegrationTestSuite) TestQueryParams_MultipleQueries() {
	// Set params
	params := types.Params{
		CreateStakingRewardFee: sdk.NewCoin("ubze", math.NewInt(750)),
		CreateTradingRewardFee: sdk.NewCoin("utoken", math.NewInt(1250)),
	}

	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	// Query params multiple times
	req := &types.QueryParamsRequest{}

	for i := 0; i < 5; i++ {
		response, err := suite.k.Params(suite.ctx, req)

		suite.Require().NoError(err)
		suite.Require().NotNil(response)
		suite.Require().Equal(params.CreateStakingRewardFee, response.Params.CreateStakingRewardFee)
		suite.Require().Equal(params.CreateTradingRewardFee, response.Params.CreateTradingRewardFee)
	}
}

func (suite *IntegrationTestSuite) TestQueryParams_AfterUpdate() {
	// Set initial params
	initialParams := types.Params{
		CreateStakingRewardFee: sdk.NewCoin("ubze", math.NewInt(100)),
		CreateTradingRewardFee: sdk.NewCoin("ubze", math.NewInt(200)),
	}

	err := suite.k.SetParams(suite.ctx, initialParams)
	suite.Require().NoError(err)

	// Query initial params
	req := &types.QueryParamsRequest{}
	response1, err := suite.k.Params(suite.ctx, req)

	suite.Require().NoError(err)
	suite.Require().Equal(initialParams.CreateStakingRewardFee, response1.Params.CreateStakingRewardFee)

	// Update params
	updatedParams := types.Params{
		CreateStakingRewardFee: sdk.NewCoin("utoken", math.NewInt(300)),
		CreateTradingRewardFee: sdk.NewCoin("utoken", math.NewInt(400)),
	}

	err = suite.k.SetParams(suite.ctx, updatedParams)
	suite.Require().NoError(err)

	// Query updated params
	response2, err := suite.k.Params(suite.ctx, req)

	suite.Require().NoError(err)
	suite.Require().Equal(updatedParams.CreateStakingRewardFee, response2.Params.CreateStakingRewardFee)
	suite.Require().Equal(updatedParams.CreateTradingRewardFee, response2.Params.CreateTradingRewardFee)
}
