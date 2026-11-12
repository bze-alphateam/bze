package types_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/bze-alphateam/bze/x/guardian/types"
)

var channelCases = []struct {
	name    string
	channel string
	valid   bool
}{
	{"first channel", "channel-0", true},
	{"large index", "channel-12345", true},
	{"empty", "", false},
	{"no index", "channel-", false},
	{"not a channel id", "connection-0", false},
	{"space", "channel 0", false},
	{"path separator", "channel-0/transfer", false},
	{"leading zero", "channel-07", false},
	{"zero padded zero", "channel-00", false},
}

func TestMsgCloseChannel_ValidateBasic(t *testing.T) {
	signer := sdk.AccAddress("signer______________").String()

	require.Error(t, (&types.MsgCloseChannel{Signer: "nope", ChannelId: "channel-0"}).ValidateBasic())
	require.Error(t, (&types.MsgCloseChannel{Signer: "", ChannelId: "channel-0"}).ValidateBasic())

	for _, c := range channelCases {
		t.Run(c.name, func(t *testing.T) {
			err := (&types.MsgCloseChannel{Signer: signer, ChannelId: c.channel}).ValidateBasic()
			if c.valid {
				require.NoError(t, err)
				return
			}
			require.ErrorIs(t, err, types.ErrInvalidChannel)
		})
	}
}

func TestMsgReopenChannel_ValidateBasic(t *testing.T) {
	authority := sdk.AccAddress("authority___________").String()

	require.Error(t, (&types.MsgReopenChannel{Authority: "nope", ChannelId: "channel-0"}).ValidateBasic())
	require.Error(t, (&types.MsgReopenChannel{Authority: "", ChannelId: "channel-0"}).ValidateBasic())

	for _, c := range channelCases {
		t.Run(c.name, func(t *testing.T) {
			err := (&types.MsgReopenChannel{Authority: authority, ChannelId: c.channel}).ValidateBasic()
			if c.valid {
				require.NoError(t, err)
				return
			}
			require.ErrorIs(t, err, types.ErrInvalidChannel)
		})
	}
}

func TestGenesis_ValidateClosedChannels(t *testing.T) {
	good := types.GenesisState{Params: types.DefaultParams(), ClosedChannels: []string{"channel-0", "channel-5"}}
	require.NoError(t, good.Validate())

	invalid := types.GenesisState{Params: types.DefaultParams(), ClosedChannels: []string{"channel-0", "nope"}}
	require.ErrorIs(t, invalid.Validate(), types.ErrInvalidChannel)

	padded := types.GenesisState{Params: types.DefaultParams(), ClosedChannels: []string{"channel-07"}}
	require.ErrorIs(t, padded.Validate(), types.ErrInvalidChannel)

	duplicate := types.GenesisState{Params: types.DefaultParams(), ClosedChannels: []string{"channel-0", "channel-0"}}
	require.Error(t, duplicate.Validate())
}
