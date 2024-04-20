package keeper_test

import (
	"github.com/bze-alphateam/bze/x/rewards/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (suite *IntegrationTestSuite) TestQueryGetParams_InvalidRequest() {
	goCtx := sdk.WrapSDKContext(suite.ctx)
	_, err := suite.k.Params(goCtx, nil)
	suite.Require().Error(err)
}

func (suite *IntegrationTestSuite) TestQueryGetParams_Success() {
	goCtx := sdk.WrapSDKContext(suite.ctx)

	resp, err := suite.k.Params(goCtx, &types.QueryParamsRequest{})
	suite.Require().NoError(err)

	def := types.DefaultGenesis()
	suite.Require().Equal(resp.Params, def.GetParams())
}
