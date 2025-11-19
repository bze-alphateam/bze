package keeper_test

import (
	"github.com/bze-alphateam/bze/x/tradebin/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (suite *IntegrationTestSuite) TestQueryParams_ValidRequest() {
	// Set some params first
	params := types.Params{
		CreateMarketFee:     "1000",
		MarketMakerFee:      "0.001",
		MarketTakerFee:      "0.002",
		MakerFeeDestination: "community_pool",
		TakerFeeDestination: "burn",
	}

	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	// Query params
	req := &types.QueryParamsRequest{}
	response, err := suite.k.Params(suite.ctx, req)

	suite.Require().NoError(err)
	suite.Require().NotNil(response)
	suite.Require().Equal(params.CreateMarketFee, response.Params.CreateMarketFee)
	suite.Require().Equal(params.MarketMakerFee, response.Params.MarketMakerFee)
	suite.Require().Equal(params.MarketTakerFee, response.Params.MarketTakerFee)
	suite.Require().Equal(params.MakerFeeDestination, response.Params.MakerFeeDestination)
	suite.Require().Equal(params.TakerFeeDestination, response.Params.TakerFeeDestination)
}

func (suite *IntegrationTestSuite) TestQueryParams_NilRequest() {
	response, err := suite.k.Params(suite.ctx, nil)

	suite.Require().Error(err)
	suite.Require().Nil(response)
	suite.Require().Equal(codes.InvalidArgument, status.Code(err))
	suite.Require().Contains(err.Error(), "invalid request")
}

func (suite *IntegrationTestSuite) TestQueryParams_DefaultParams() {
	// Query params without setting any (should return default values)
	req := &types.QueryParamsRequest{}
	response, err := suite.k.Params(suite.ctx, req)

	suite.Require().NoError(err)
	suite.Require().NotNil(response)
	suite.Require().Equal("25000000000ubze", response.Params.CreateMarketFee)
	suite.Require().Equal("1000ubze", response.Params.MarketMakerFee)
	suite.Require().Equal("100000ubze", response.Params.MarketTakerFee)
	suite.Require().Equal("burner", response.Params.MakerFeeDestination)
	suite.Require().Equal("burner", response.Params.TakerFeeDestination)
}

func (suite *IntegrationTestSuite) TestQueryParams_ZeroParams() {
	// Set zero value params
	params := types.Params{
		CreateMarketFee:     "0",
		MarketMakerFee:      "0",
		MarketTakerFee:      "0",
		MakerFeeDestination: "",
		TakerFeeDestination: "",
	}

	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	// Query params
	req := &types.QueryParamsRequest{}
	response, err := suite.k.Params(suite.ctx, req)

	suite.Require().NoError(err)
	suite.Require().NotNil(response)
	suite.Require().Equal("0", response.Params.CreateMarketFee)
	suite.Require().Equal("0", response.Params.MarketMakerFee)
	suite.Require().Equal("0", response.Params.MarketTakerFee)
	suite.Require().Equal("", response.Params.MakerFeeDestination)
	suite.Require().Equal("", response.Params.TakerFeeDestination)
}

func (suite *IntegrationTestSuite) TestQueryParams_DifferentDestinations() {
	// Set params with different fee destinations
	params := types.Params{
		CreateMarketFee:     "500",
		MarketMakerFee:      "0.0005",
		MarketTakerFee:      "0.001",
		MakerFeeDestination: "validator_rewards",
		TakerFeeDestination: "community_pool",
	}

	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	// Query params
	req := &types.QueryParamsRequest{}
	response, err := suite.k.Params(suite.ctx, req)

	suite.Require().NoError(err)
	suite.Require().NotNil(response)
	suite.Require().Equal("500", response.Params.CreateMarketFee)
	suite.Require().Equal("0.0005", response.Params.MarketMakerFee)
	suite.Require().Equal("0.001", response.Params.MarketTakerFee)
	suite.Require().Equal("validator_rewards", response.Params.MakerFeeDestination)
	suite.Require().Equal("community_pool", response.Params.TakerFeeDestination)
}

func (suite *IntegrationTestSuite) TestQueryParams_HighFees() {
	// Set params with high fee values
	params := types.Params{
		CreateMarketFee:     "10000",
		MarketMakerFee:      "0.01", // 1%
		MarketTakerFee:      "0.02", // 2%
		MakerFeeDestination: "burn",
		TakerFeeDestination: "burn",
	}

	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	// Query params
	req := &types.QueryParamsRequest{}
	response, err := suite.k.Params(suite.ctx, req)

	suite.Require().NoError(err)
	suite.Require().NotNil(response)
	suite.Require().Equal("10000", response.Params.CreateMarketFee)
	suite.Require().Equal("0.01", response.Params.MarketMakerFee)
	suite.Require().Equal("0.02", response.Params.MarketTakerFee)
	suite.Require().Equal("burn", response.Params.MakerFeeDestination)
	suite.Require().Equal("burn", response.Params.TakerFeeDestination)
}

func (suite *IntegrationTestSuite) TestQueryParams_MultipleQueries() {
	// Set params
	params := types.Params{
		CreateMarketFee:     "750",
		MarketMakerFee:      "0.00075",
		MarketTakerFee:      "0.0015",
		MakerFeeDestination: "community_pool",
		TakerFeeDestination: "validator_rewards",
	}

	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	// Query params multiple times
	req := &types.QueryParamsRequest{}

	for i := 0; i < 5; i++ {
		response, err := suite.k.Params(suite.ctx, req)

		suite.Require().NoError(err)
		suite.Require().NotNil(response)
		suite.Require().Equal(params.CreateMarketFee, response.Params.CreateMarketFee)
		suite.Require().Equal(params.MarketMakerFee, response.Params.MarketMakerFee)
		suite.Require().Equal(params.MarketTakerFee, response.Params.MarketTakerFee)
		suite.Require().Equal(params.MakerFeeDestination, response.Params.MakerFeeDestination)
		suite.Require().Equal(params.TakerFeeDestination, response.Params.TakerFeeDestination)
	}
}

func (suite *IntegrationTestSuite) TestQueryParams_AfterUpdate() {
	// Set initial params
	initialParams := types.Params{
		CreateMarketFee:     "100",
		MarketMakerFee:      "0.0001",
		MarketTakerFee:      "0.0002",
		MakerFeeDestination: "community_pool",
		TakerFeeDestination: "burn",
	}

	err := suite.k.SetParams(suite.ctx, initialParams)
	suite.Require().NoError(err)

	// Query initial params
	req := &types.QueryParamsRequest{}
	response1, err := suite.k.Params(suite.ctx, req)

	suite.Require().NoError(err)
	suite.Require().Equal(initialParams.CreateMarketFee, response1.Params.CreateMarketFee)

	// Update params
	updatedParams := types.Params{
		CreateMarketFee:     "300",
		MarketMakerFee:      "0.0003",
		MarketTakerFee:      "0.0006",
		MakerFeeDestination: "validator_rewards",
		TakerFeeDestination: "community_pool",
	}

	err = suite.k.SetParams(suite.ctx, updatedParams)
	suite.Require().NoError(err)

	// Query updated params
	response2, err := suite.k.Params(suite.ctx, req)

	suite.Require().NoError(err)
	suite.Require().Equal(updatedParams.CreateMarketFee, response2.Params.CreateMarketFee)
	suite.Require().Equal(updatedParams.MarketMakerFee, response2.Params.MarketMakerFee)
	suite.Require().Equal(updatedParams.MarketTakerFee, response2.Params.MarketTakerFee)
	suite.Require().Equal(updatedParams.MakerFeeDestination, response2.Params.MakerFeeDestination)
	suite.Require().Equal(updatedParams.TakerFeeDestination, response2.Params.TakerFeeDestination)
}
