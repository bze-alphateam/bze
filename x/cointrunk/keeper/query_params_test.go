package keeper_test

import (
	"cosmossdk.io/math"

	"github.com/bze-alphateam/bze/x/cointrunk/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (suite *IntegrationTestSuite) TestParams_ValidRequest() {
	// Set some params first
	params := types.Params{
		AnonArticleLimit: 100,
		AnonArticleCost:  sdk.NewInt64Coin("ubze", 1000),
		PublisherRespectParams: types.PublisherRespectParams{
			Tax:   math.LegacyMustNewDecFromStr("0.1"),
			Denom: "ubze",
		},
	}

	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	req := &types.QueryParamsRequest{}
	res, err := suite.k.Params(suite.ctx, req)

	suite.Require().NoError(err)
	suite.Require().NotNil(res)
	suite.Require().NotNil(res.Params)
	suite.Require().Equal(params.AnonArticleLimit, res.Params.AnonArticleLimit)
	suite.Require().True(params.AnonArticleCost.Equal(&res.Params.AnonArticleCost))
	suite.Require().True(params.PublisherRespectParams.Tax.Equal(res.Params.PublisherRespectParams.Tax))
	suite.Require().Equal(params.PublisherRespectParams.Denom, res.Params.PublisherRespectParams.Denom)
}

func (suite *IntegrationTestSuite) TestParams_NilRequest() {
	res, err := suite.k.Params(suite.ctx, nil)

	suite.Require().Error(err)
	suite.Require().Nil(res)
	suite.Require().Equal(codes.InvalidArgument, status.Code(err))
	suite.Require().Contains(err.Error(), "invalid request")
}

func (suite *IntegrationTestSuite) TestParams_ContextUnwrapping() {
	// Set some params
	params := types.Params{
		AnonArticleLimit: 50,
		AnonArticleCost:  sdk.NewInt64Coin("utoken", 500),
		PublisherRespectParams: types.PublisherRespectParams{
			Tax:   math.LegacyMustNewDecFromStr("0.05"),
			Denom: "utoken",
		},
	}

	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	req := &types.QueryParamsRequest{}

	// Test with SDK context
	res, err := suite.k.Params(suite.ctx, req)
	suite.Require().NoError(err)
	suite.Require().NotNil(res)
	suite.Require().Equal(params.AnonArticleLimit, res.Params.AnonArticleLimit)
}
