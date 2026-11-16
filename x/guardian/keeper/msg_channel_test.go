package keeper_test

import (
	"testing"

	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/bze-alphateam/bze/x/guardian/testutil"
	"github.com/bze-alphateam/bze/x/guardian/types"
)

const transferPort = "transfer"

// newChannelEnv is newEnv with a mocked IBC channel keeper wired.
func newChannelEnv(t *testing.T) (*env, *testutil.MockChannelKeeper) {
	t.Helper()

	e := newEnv(t)
	ch := testutil.NewMockChannelKeeper(gomock.NewController(t))
	e.k.SetChannelKeeper(ch)

	return e, ch
}

func expectChannel(ch *testutil.MockChannelKeeper, channelID string, exists bool) {
	ch.EXPECT().GetChannel(gomock.Any(), transferPort, channelID).Return(channeltypes.Channel{}, exists)
}

func closeMsg(signer string) *types.MsgCloseChannel {
	return &types.MsgCloseChannel{Signer: signer, ChannelId: testChannel}
}

func reopenMsg(signer string) *types.MsgReopenChannel {
	return &types.MsgReopenChannel{Authority: signer, ChannelId: testChannel}
}

func TestCloseChannel_GuardianAndAuthorityClose(t *testing.T) {
	for name, signer := range map[string]string{"guardian": guardianAddr, "authority": authority} {
		t.Run(name, func(t *testing.T) {
			e, ch := newChannelEnv(t)
			expectChannel(ch, testChannel, true)

			_, err := e.msgServer.CloseChannel(e.ctx, closeMsg(signer))
			require.NoError(t, err)
			require.True(t, e.k.IsChannelClosed(e.ctx, testChannel))
			require.False(t, e.k.IsChannelClosed(e.ctx, "channel-1"))
			require.Equal(t, []string{"bze.guardian.v1.ChannelClosedEvent"}, eventTypes(e.ctx))
		})
	}
}

func TestCloseChannel_StrangerIsRefused(t *testing.T) {
	e, _ := newChannelEnv(t)

	_, err := e.msgServer.CloseChannel(e.ctx, closeMsg(strangerAddr))
	require.ErrorIs(t, err, types.ErrUnauthorized)
	require.False(t, e.k.IsChannelClosed(e.ctx, testChannel))
	require.Empty(t, eventTypes(e.ctx))
}

func TestCloseChannel_UnknownChannelIsRefused(t *testing.T) {
	e, ch := newChannelEnv(t)
	expectChannel(ch, testChannel, false)

	_, err := e.msgServer.CloseChannel(e.ctx, closeMsg(guardianAddr))
	require.ErrorIs(t, err, types.ErrChannelNotFound)
	require.False(t, e.k.IsChannelClosed(e.ctx, testChannel))
	require.Empty(t, eventTypes(e.ctx))
}

func TestCloseChannel_DoubleCloseIsRefused(t *testing.T) {
	e, ch := newChannelEnv(t)
	expectChannel(ch, testChannel, true)
	_, err := e.msgServer.CloseChannel(e.ctx, closeMsg(guardianAddr))
	require.NoError(t, err)

	expectChannel(ch, testChannel, true)
	_, err = e.msgServer.CloseChannel(e.ctx, closeMsg(authority))
	require.ErrorIs(t, err, types.ErrChannelAlreadyClosed)
	require.Len(t, eventTypes(e.ctx), 1)
	require.True(t, e.k.IsChannelClosed(e.ctx, testChannel))
}

func TestCloseChannel_InvalidMessageIsRefusedBeforeAnythingElse(t *testing.T) {
	e, _ := newChannelEnv(t)

	_, err := e.msgServer.CloseChannel(e.ctx, &types.MsgCloseChannel{Signer: guardianAddr, ChannelId: "nope"})
	require.ErrorIs(t, err, types.ErrInvalidChannel)
}

func TestCloseChannel_WithoutChannelKeeperIsRefused(t *testing.T) {
	e := newEnv(t)

	_, err := e.msgServer.CloseChannel(e.ctx, closeMsg(guardianAddr))
	require.ErrorIs(t, err, types.ErrChannelKeeperNotSet)
	require.False(t, e.k.IsChannelClosed(e.ctx, testChannel))
}

func TestCloseChannel_EmptyGuardianLeavesOnlyTheAuthority(t *testing.T) {
	e, ch := newChannelEnv(t)
	require.NoError(t, e.k.SetParams(e.ctx, types.DefaultParams()))

	_, err := e.msgServer.CloseChannel(e.ctx, closeMsg(guardianAddr))
	require.ErrorIs(t, err, types.ErrUnauthorized)

	expectChannel(ch, testChannel, true)
	_, err = e.msgServer.CloseChannel(e.ctx, closeMsg(authority))
	require.NoError(t, err)
}

func TestReopenChannel_AuthorityReopens(t *testing.T) {
	e, _ := newChannelEnv(t)
	e.k.SetClosedChannel(e.ctx, testChannel)

	_, err := e.msgServer.ReopenChannel(e.ctx, reopenMsg(authority))
	require.NoError(t, err)
	require.False(t, e.k.IsChannelClosed(e.ctx, testChannel))
	require.Equal(t, []string{"bze.guardian.v1.ChannelReopenedEvent"}, eventTypes(e.ctx))
}

func TestReopenChannel_GuardianAndStrangerAreRefused(t *testing.T) {
	for name, signer := range map[string]string{"guardian": guardianAddr, "stranger": strangerAddr} {
		t.Run(name, func(t *testing.T) {
			e, _ := newChannelEnv(t)
			e.k.SetClosedChannel(e.ctx, testChannel)

			_, err := e.msgServer.ReopenChannel(e.ctx, reopenMsg(signer))
			require.ErrorIs(t, err, types.ErrUnauthorized)
			require.True(t, e.k.IsChannelClosed(e.ctx, testChannel))
			require.Empty(t, eventTypes(e.ctx))
		})
	}
}

func TestReopenChannel_NotClosedIsRefused(t *testing.T) {
	e, _ := newChannelEnv(t)

	_, err := e.msgServer.ReopenChannel(e.ctx, reopenMsg(authority))
	require.ErrorIs(t, err, types.ErrChannelNotClosed)
	require.Empty(t, eventTypes(e.ctx))
}

func TestReopenChannel_InvalidMessageIsRefused(t *testing.T) {
	e, _ := newChannelEnv(t)

	_, err := e.msgServer.ReopenChannel(e.ctx, &types.MsgReopenChannel{Authority: authority, ChannelId: ""})
	require.ErrorIs(t, err, types.ErrInvalidChannel)
}

// Reopening one channel leaves the others closed.
func TestReopenChannel_OnlyTheNamedChannel(t *testing.T) {
	e, _ := newChannelEnv(t)
	e.k.SetClosedChannel(e.ctx, testChannel)
	e.k.SetClosedChannel(e.ctx, "channel-9")

	_, err := e.msgServer.ReopenChannel(e.ctx, reopenMsg(authority))
	require.NoError(t, err)
	require.True(t, e.k.IsChannelClosed(e.ctx, "channel-9"))
	require.Equal(t, []string{"channel-9"}, e.k.GetAllClosedChannels(e.ctx))
}

func TestClosedChannels_Query(t *testing.T) {
	e, _ := newChannelEnv(t)

	res, err := e.k.ClosedChannels(e.ctx, &types.QueryClosedChannelsRequest{})
	require.NoError(t, err)
	require.Empty(t, res.ChannelIds)

	e.k.SetClosedChannel(e.ctx, "channel-9")
	e.k.SetClosedChannel(e.ctx, testChannel)

	res, err = e.k.ClosedChannels(e.ctx, &types.QueryClosedChannelsRequest{})
	require.NoError(t, err)
	require.ElementsMatch(t, []string{testChannel, "channel-9"}, res.ChannelIds)

	_, err = e.k.ClosedChannels(e.ctx, nil)
	require.Error(t, err)
}

// The closed set lives under its own prefix: a channel id that is a prefix of another
// does not close it.
func TestIsChannelClosed_ExactMatch(t *testing.T) {
	e, _ := newChannelEnv(t)
	e.k.SetClosedChannel(e.ctx, "channel-1")

	require.True(t, e.k.IsChannelClosed(e.ctx, "channel-1"))
	require.False(t, e.k.IsChannelClosed(e.ctx, "channel-10"))
	require.False(t, e.k.IsChannelClosed(e.ctx, "channel-"))
}
