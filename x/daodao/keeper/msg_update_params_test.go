package keeper_test

import (
	"github.com/bze-alphateam/bze/x/daodao/types"
)

func (suite *IntegrationTestSuite) TestMsgUpdateParams() {
	params := types.DefaultParams()
	suite.Require().NoError(suite.k.SetParams(suite.ctx, params))

	// "Invalid params under a valid authority" should be rejected by
	// Params.Validate before SetParams runs, so live state stays at
	// `params`. We build this once here to assert un-changed state after
	// the failure case.
	invalidParams := types.DefaultParams()
	invalidParams.DaoCreationFeeDestination = "not_a_real_destination"

	testCases := []struct {
		name      string
		input     *types.MsgUpdateParams
		expErr    bool
		expErrMsg string
	}{
		{
			name: "invalid authority",
			input: &types.MsgUpdateParams{
				Authority: "invalid",
				Params:    params,
			},
			expErr:    true,
			expErrMsg: "invalid authority",
		},
		{
			name: "valid authority but invalid params",
			input: &types.MsgUpdateParams{
				Authority: suite.k.GetAuthority(),
				Params:    invalidParams,
			},
			expErr:    true,
			expErrMsg: "dao_creation_fee_destination",
		},
		{
			name: "all good",
			input: &types.MsgUpdateParams{
				Authority: suite.k.GetAuthority(),
				Params:    params,
			},
			expErr: false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			_, err := suite.msgServer.UpdateParams(suite.ctx, tc.input)
			if tc.expErr {
				suite.Require().Error(err)
				suite.Require().Contains(err.Error(), tc.expErrMsg)
				// On error, previous (valid) params must remain.
				suite.Require().Equal(params, suite.k.GetParams(suite.ctx))
			} else {
				suite.Require().NoError(err)
			}
		})
	}
}
