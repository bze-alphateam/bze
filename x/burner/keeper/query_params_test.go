package keeper_test

import (
	"github.com/bze-alphateam/bze/x/burner/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (suite *IntegrationTestSuite) TestParams_ValidRequest() {
	req := &types.QueryParamsRequest{}

	res, err := suite.k.Params(suite.ctx, req)

	suite.Require().NoError(err)
	suite.Require().NotNil(res)
	suite.Require().NotNil(res.Params)
}

func (suite *IntegrationTestSuite) TestParams_NilRequest() {
	res, err := suite.k.Params(suite.ctx, nil)

	suite.Require().Error(err)
	suite.Require().Nil(res)
	suite.Require().Equal(codes.InvalidArgument, status.Code(err))
	suite.Require().Contains(err.Error(), "invalid request")
}

func (suite *IntegrationTestSuite) TestParams_ContextUnwrapping() {
	req := &types.QueryParamsRequest{}

	// Test with different context types
	res, err := suite.k.Params(suite.ctx, req)
	suite.Require().NoError(err)
	suite.Require().NotNil(res)

	res, err = suite.k.Params(suite.ctx, req)
	suite.Require().NoError(err)
	suite.Require().NotNil(res)
}
