package keeper_test

import (
	"testing"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/bze-alphateam/bze/x/txfeecollector/types"
)

func TestMsgUpdateParams(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	params := types.DefaultParams()
	require.NoError(t, k.SetParams(ctx, params))
	wctx := sdk.UnwrapSDKContext(ctx)

	// default params
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
			name: "empty params should fail validation",
			input: &types.MsgUpdateParams{
				Authority: k.GetAuthority(),
				Params:    types.Params{},
			},
			expErr:    true,
			expErrMsg: "validator min gas fee denom must be ubze",
		},
		{
			name: "invalid cw deploy fee destination",
			input: &types.MsgUpdateParams{
				Authority: k.GetAuthority(),
				Params: types.NewParams(
					params.ValidatorMinGasFee,
					params.MaxBalanceIterations,
					"invalid_destination",
					sdk.NewCoins(),
				),
			},
			expErr:    true,
			expErrMsg: "invalid cw_deploy_fee_destination",
		},
		{
			name: "zero max balance iterations",
			input: &types.MsgUpdateParams{
				Authority: k.GetAuthority(),
				Params: types.NewParams(
					params.ValidatorMinGasFee,
					0,
					params.CwDeployFeeDestination,
					params.CwDeployFee,
				),
			},
			expErr:    true,
			expErrMsg: "max balance iterations must be greater than 0",
		},
		{
			name: "negative validator min gas fee",
			input: &types.MsgUpdateParams{
				Authority: k.GetAuthority(),
				Params: types.NewParams(
					sdk.DecCoin{Denom: "ubze", Amount: sdkmath.LegacyNewDec(-1)},
					params.MaxBalanceIterations,
					params.CwDeployFeeDestination,
					params.CwDeployFee,
				),
			},
			expErr:    true,
			expErrMsg: "validator min gas fee amount cannot be negative",
		},
		{
			name: "all good",
			input: &types.MsgUpdateParams{
				Authority: k.GetAuthority(),
				Params:    params,
			},
			expErr: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ms.UpdateParams(wctx, tc.input)

			if tc.expErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expErrMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
