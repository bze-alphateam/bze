package types

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDefaultParamsBlockInboundNothing(t *testing.T) {
	params := DefaultParams()

	require.NoError(t, params.Validate())
	require.Empty(t, params.BlockedIbcInbound)
	require.False(t, params.IsInboundBlocked("channel-3", "uusdc"))
}

func TestValidateBlockedIbcInbound(t *testing.T) {
	tests := []struct {
		name      string
		blocked   []BlockedIbcTransfer
		expErr    bool
		expErrMsg string
	}{
		{
			name:    "empty list is valid",
			blocked: []BlockedIbcTransfer{},
		},
		{
			name:    "nil list is valid",
			blocked: nil,
		},
		{
			name: "single entry",
			blocked: []BlockedIbcTransfer{
				{ChannelId: "channel-3", BaseDenom: "uusdc"},
			},
		},
		{
			name: "several entries on the same channel",
			blocked: []BlockedIbcTransfer{
				{ChannelId: "channel-3", BaseDenom: "uusdc"},
				{ChannelId: "channel-3", BaseDenom: "ufrienzies"},
			},
		},
		{
			name: "ibc voucher path as base denom",
			blocked: []BlockedIbcTransfer{
				{ChannelId: "channel-3", BaseDenom: "transfer/channel-95/uatom"},
			},
		},
		{
			name: "empty channel id",
			blocked: []BlockedIbcTransfer{
				{ChannelId: "", BaseDenom: "uusdc"},
			},
			expErr:    true,
			expErrMsg: "invalid blocked ibc inbound channel id",
		},
		{
			name: "malformed channel id",
			blocked: []BlockedIbcTransfer{
				{ChannelId: "not a channel", BaseDenom: "uusdc"},
			},
			expErr:    true,
			expErrMsg: "invalid blocked ibc inbound channel id",
		},
		{
			name: "empty base denom",
			blocked: []BlockedIbcTransfer{
				{ChannelId: "channel-3", BaseDenom: ""},
			},
			expErr:    true,
			expErrMsg: "invalid blocked ibc inbound base denom",
		},
		{
			name: "invalid base denom",
			blocked: []BlockedIbcTransfer{
				{ChannelId: "channel-3", BaseDenom: "!!!"},
			},
			expErr:    true,
			expErrMsg: "invalid blocked ibc inbound base denom",
		},
		{
			name: "duplicate entry",
			blocked: []BlockedIbcTransfer{
				{ChannelId: "channel-3", BaseDenom: "uusdc"},
				{ChannelId: "channel-3", BaseDenom: "uusdc"},
			},
			expErr:    true,
			expErrMsg: "duplicate blocked ibc inbound entry",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := DefaultParams()
			params.BlockedIbcInbound = tt.blocked

			err := params.Validate()
			if tt.expErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.expErrMsg)

				return
			}

			require.NoError(t, err)
		})
	}
}

func TestIsInboundBlocked(t *testing.T) {
	params := DefaultParams()
	params.BlockedIbcInbound = []BlockedIbcTransfer{
		{ChannelId: "channel-3", BaseDenom: "uusdc"},
	}
	require.NoError(t, params.Validate())

	require.True(t, params.IsInboundBlocked("channel-3", "uusdc"))
	// same denom on another channel is not blocked
	require.False(t, params.IsInboundBlocked("channel-13", "uusdc"))
	// another denom on the blocked channel is not blocked
	require.False(t, params.IsInboundBlocked("channel-3", "uatom"))
	// the voucher form of the denom is not the packet denom and must not match
	require.False(t, params.IsInboundBlocked("channel-3", "transfer/channel-3/uusdc"))
	require.False(t, params.IsInboundBlocked("", ""))
}
