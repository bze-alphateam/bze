package keeper_test

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/bze-alphateam/bze/x/dao/types"
)

func (suite *IntegrationTestSuite) TestUpdateParams_Authority() {
	res, err := suite.msgServer.UpdateParams(suite.ctx, &types.MsgUpdateParams{
		Authority: suite.k.GetAuthority(),
		Params:    types.DefaultParams(),
	})

	suite.Require().NoError(err)
	suite.Require().NotNil(res)
	suite.Require().Equal(types.DefaultParams(), suite.k.GetParams(suite.ctx))
}

func (suite *IntegrationTestSuite) TestUpdateParams_RejectsOtherSigners() {
	for _, authority := range []string{"", sdk.AccAddress("not_the_gov_account_").String()} {
		res, err := suite.msgServer.UpdateParams(suite.ctx, &types.MsgUpdateParams{
			Authority: authority,
			Params:    types.DefaultParams(),
		})

		suite.Require().ErrorIs(err, types.ErrInvalidSigner)
		suite.Require().ErrorContains(err, "invalid authority")
		suite.Require().Nil(res)
	}
}

func (suite *IntegrationTestSuite) TestUpdateParams_RejectsInvalidParams() {
	params := types.DefaultParams()
	params.MaxStaticMembers = 0

	res, err := suite.msgServer.UpdateParams(suite.ctx, &types.MsgUpdateParams{
		Authority: suite.k.GetAuthority(),
		Params:    params,
	})

	suite.Require().ErrorContains(err, "max_static_members")
	suite.Require().Nil(res)
	suite.Require().Equal(types.DefaultParams(), suite.k.GetParams(suite.ctx))
}
