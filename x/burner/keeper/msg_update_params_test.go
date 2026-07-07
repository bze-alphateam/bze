package keeper_test

import (
	"github.com/bze-alphateam/bze/x/burner/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
)

func (suite *IntegrationTestSuite) TestUpdateParams_ValidAuthority() {
	authority := authtypes.NewModuleAddress(govtypes.ModuleName).String()

	msg := &types.MsgUpdateParams{
		Authority: authority,
		Params:    types.NewParams(2),
	}

	res, err := suite.msgServer.UpdateParams(suite.ctx, msg)
	suite.Require().NoError(err)
	suite.Require().NotNil(res)

	// Verify params were updated
	params := suite.k.GetParams(suite.ctx)
	suite.Require().Equal(int64(2), params.PeriodicBurningWeeks)
}

func (suite *IntegrationTestSuite) TestUpdateParams_InvalidAuthority() {
	msg := &types.MsgUpdateParams{
		Authority: "invalid-authority",
		Params:    types.NewParams(2),
	}

	res, err := suite.msgServer.UpdateParams(suite.ctx, msg)
	suite.Require().Error(err)
	suite.Require().Nil(res)
	suite.Require().Contains(err.Error(), "invalid authority")
}

func (suite *IntegrationTestSuite) TestUpdateParams_InvalidParams() {
	authority := authtypes.NewModuleAddress(govtypes.ModuleName).String()

	msg := &types.MsgUpdateParams{
		Authority: authority,
		Params:    types.NewParams(0), // Invalid: PeriodicBurningWeeks must be > 0
	}

	res, err := suite.msgServer.UpdateParams(suite.ctx, msg)
	suite.Require().Error(err)
	suite.Require().Nil(res)
}

func (suite *IntegrationTestSuite) TestUpdateParams_NegativeParams() {
	authority := authtypes.NewModuleAddress(govtypes.ModuleName).String()

	msg := &types.MsgUpdateParams{
		Authority: authority,
		Params:    types.NewParams(-1), // Negative value
	}

	res, err := suite.msgServer.UpdateParams(suite.ctx, msg)
	suite.Require().Error(err)
	suite.Require().Nil(res)
}
