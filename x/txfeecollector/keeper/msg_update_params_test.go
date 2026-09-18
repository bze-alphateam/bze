package keeper_test

import (
	"testing"

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

// Governance can add and remove blocked inbound transfers after the upgrade has set
// them: the value written by MsgUpdateParams is what the IBC filter reads back.
func TestMsgUpdateParams_BlockedIbcInbound(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	wctx := sdk.UnwrapSDKContext(ctx)

	params := types.DefaultParams()
	require.NoError(t, k.SetParams(ctx, params))
	require.Empty(t, k.GetParams(wctx).BlockedIbcInbound)

	blocked := types.DefaultParams()
	blocked.BlockedIbcInbound = []types.BlockedIbcTransfer{
		{ChannelId: "channel-3", BaseDenom: "uusdc"},
		{ChannelId: "channel-13", BaseDenom: "uatom"},
	}
	_, err := ms.UpdateParams(wctx, &types.MsgUpdateParams{Authority: k.GetAuthority(), Params: blocked})
	require.NoError(t, err)

	stored := k.GetParams(wctx)
	require.Equal(t, blocked.BlockedIbcInbound, stored.BlockedIbcInbound)
	require.True(t, stored.IsInboundBlocked("channel-3", "uusdc"))
	require.True(t, stored.IsInboundBlocked("channel-13", "uatom"))
	require.False(t, stored.IsInboundBlocked("channel-3", "uatom"))

	// an invalid entry is rejected and the stored value is left alone
	invalid := types.DefaultParams()
	invalid.BlockedIbcInbound = []types.BlockedIbcTransfer{
		{ChannelId: "not a channel", BaseDenom: "uusdc"},
	}
	_, err = ms.UpdateParams(wctx, &types.MsgUpdateParams{Authority: k.GetAuthority(), Params: invalid})
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid blocked ibc inbound channel id")
	require.Equal(t, blocked.BlockedIbcInbound, k.GetParams(wctx).BlockedIbcInbound)

	// governance can lift the block again
	lifted := types.DefaultParams()
	_, err = ms.UpdateParams(wctx, &types.MsgUpdateParams{Authority: k.GetAuthority(), Params: lifted})
	require.NoError(t, err)
	require.Empty(t, k.GetParams(wctx).BlockedIbcInbound)
}
