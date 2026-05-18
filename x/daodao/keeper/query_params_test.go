package keeper_test

import (
	"github.com/bze-alphateam/bze/x/daodao/types"
)

func (suite *IntegrationTestSuite) TestParamsQuery() {
	params := types.DefaultParams()
	suite.Require().NoError(suite.k.SetParams(suite.ctx, params))

	response, err := suite.k.Params(suite.ctx, &types.QueryParamsRequest{})
	suite.Require().NoError(err)
	suite.Require().Equal(&types.QueryParamsResponse{Params: params}, response)
}
