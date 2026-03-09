package keeper_test

import (
	"github.com/bze-alphateam/bze/x/tradebin/types"
	v2types "github.com/bze-alphateam/bze/x/tradebin/v2types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (suite *IntegrationTestSuite) TestQueryParams_ValidRequest() {
	params := v2types.DefaultParams()
	params.CreateMarketFee = sdk.NewInt64Coin(denomBze, 1000)
	params.MarketMakerFee = sdk.NewInt64Coin(denomBze, 100)
	params.MarketTakerFee = sdk.NewInt64Coin(denomBze, 200)
	params.MakerFeeDestination = v2types.FeeDestinationCommunityPool

	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

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
	req := &types.QueryParamsRequest{}
	response, err := suite.k.Params(suite.ctx, req)

	suite.Require().NoError(err)
	suite.Require().NotNil(response)
	suite.Require().Equal(v2types.DefaultCreateMarketFee, response.Params.CreateMarketFee)
	suite.Require().Equal(v2types.DefaultMarketMakerFee, response.Params.MarketMakerFee)
	suite.Require().Equal(v2types.DefaultMarketTakerFee, response.Params.MarketTakerFee)
	suite.Require().Equal(v2types.DefaultMakerFeeDestination, response.Params.MakerFeeDestination)
	suite.Require().Equal(v2types.DefaultTakerFeeDestination, response.Params.TakerFeeDestination)
}

func (suite *IntegrationTestSuite) TestQueryParams_ZeroFees() {
	params := v2types.DefaultParams()
	params.CreateMarketFee = sdk.NewInt64Coin(denomBze, 0)
	params.MarketMakerFee = sdk.NewInt64Coin(denomBze, 0)
	params.MarketTakerFee = sdk.NewInt64Coin(denomBze, 0)

	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	req := &types.QueryParamsRequest{}
	response, err := suite.k.Params(suite.ctx, req)

	suite.Require().NoError(err)
	suite.Require().NotNil(response)
	suite.Require().True(response.Params.CreateMarketFee.IsZero())
	suite.Require().True(response.Params.MarketMakerFee.IsZero())
	suite.Require().True(response.Params.MarketTakerFee.IsZero())
}

func (suite *IntegrationTestSuite) TestQueryParams_DifferentDestinations() {
	params := v2types.DefaultParams()
	params.MakerFeeDestination = v2types.FeeDestinationBurnerModule
	params.TakerFeeDestination = v2types.FeeDestinationCommunityPool

	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	req := &types.QueryParamsRequest{}
	response, err := suite.k.Params(suite.ctx, req)

	suite.Require().NoError(err)
	suite.Require().NotNil(response)
	suite.Require().Equal(v2types.FeeDestinationBurnerModule, response.Params.MakerFeeDestination)
	suite.Require().Equal(v2types.FeeDestinationCommunityPool, response.Params.TakerFeeDestination)
}

func (suite *IntegrationTestSuite) TestQueryParams_MultipleQueries() {
	params := v2types.DefaultParams()
	params.CreateMarketFee = sdk.NewInt64Coin(denomBze, 750)

	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	req := &types.QueryParamsRequest{}
	for i := 0; i < 5; i++ {
		response, err := suite.k.Params(suite.ctx, req)

		suite.Require().NoError(err)
		suite.Require().NotNil(response)
		suite.Require().Equal(params.CreateMarketFee, response.Params.CreateMarketFee)
		suite.Require().Equal(params.MarketMakerFee, response.Params.MarketMakerFee)
		suite.Require().Equal(params.MarketTakerFee, response.Params.MarketTakerFee)
	}
}

func (suite *IntegrationTestSuite) TestQueryParams_AfterUpdate() {
	initialParams := v2types.DefaultParams()
	initialParams.CreateMarketFee = sdk.NewInt64Coin(denomBze, 100)

	err := suite.k.SetParams(suite.ctx, initialParams)
	suite.Require().NoError(err)

	req := &types.QueryParamsRequest{}
	response1, err := suite.k.Params(suite.ctx, req)
	suite.Require().NoError(err)
	suite.Require().Equal(initialParams.CreateMarketFee, response1.Params.CreateMarketFee)

	// Update params
	updatedParams := v2types.DefaultParams()
	updatedParams.CreateMarketFee = sdk.NewInt64Coin(denomBze, 300)
	updatedParams.MakerFeeDestination = v2types.FeeDestinationCommunityPool

	err = suite.k.SetParams(suite.ctx, updatedParams)
	suite.Require().NoError(err)

	response2, err := suite.k.Params(suite.ctx, req)
	suite.Require().NoError(err)
	suite.Require().Equal(updatedParams.CreateMarketFee, response2.Params.CreateMarketFee)
	suite.Require().Equal(updatedParams.MakerFeeDestination, response2.Params.MakerFeeDestination)
}
