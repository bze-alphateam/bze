package keeper_test

import (
	"github.com/bze-alphateam/bze/x/dao/types"
)

func (suite *IntegrationTestSuite) TestParams_DefaultsAfterSetup() {
	suite.Require().Equal(types.DefaultParams(), suite.k.GetParams(suite.ctx))
}

func (suite *IntegrationTestSuite) TestParams_SetAndGetRoundTrip() {
	params := types.NewParams()

	suite.Require().NoError(suite.k.SetParams(suite.ctx, params))
	suite.Require().Equal(params, suite.k.GetParams(suite.ctx))
}
