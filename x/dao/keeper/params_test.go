package keeper_test

import (
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/bze-alphateam/bze/x/dao/types"
)

func (suite *IntegrationTestSuite) TestParams_DefaultsAfterSetup() {
	suite.Require().Equal(types.DefaultParams(), suite.k.GetParams(suite.ctx))
}

func (suite *IntegrationTestSuite) TestParams_SetAndGetRoundTrip() {
	params := types.NewParams(2, 3, 6, 4, 500, 70, 900, 2*time.Hour, 48*time.Hour, sdk.NewInt64Coin("ubze", 7), types.FeeDestinationCommunityPool, 9)

	suite.Require().NoError(suite.k.SetParams(suite.ctx, params))
	suite.Require().Equal(params, suite.k.GetParams(suite.ctx))
}
