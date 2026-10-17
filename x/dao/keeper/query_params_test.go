package keeper_test

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/bze-alphateam/bze/x/dao/types"
)

func (suite *IntegrationTestSuite) TestQueryParams_ReturnsStoredParams() {
	res, err := suite.k.Params(suite.ctx, &types.QueryParamsRequest{})

	suite.Require().NoError(err)
	suite.Require().Equal(suite.k.GetParams(suite.ctx), res.Params)
}

func (suite *IntegrationTestSuite) TestQueryParams_NilRequest() {
	res, err := suite.k.Params(suite.ctx, nil)

	suite.Require().Nil(res)
	suite.Require().Equal(codes.InvalidArgument, status.Code(err))
}
