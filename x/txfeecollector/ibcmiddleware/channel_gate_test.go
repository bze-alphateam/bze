package ibcmiddleware_test

import (
	"context"
	"testing"

	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	ibctransfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	ibcexported "github.com/cosmos/ibc-go/v10/modules/core/exported"
	"github.com/stretchr/testify/require"

	"github.com/bze-alphateam/bze/x/txfeecollector/ibcmiddleware"
	"github.com/bze-alphateam/bze/x/txfeecollector/types"
)

const (
	closedChannel = "channel-7"
	openChannel   = "channel-8"
)

// channelGuard answers the closed-channel check from a fixed set. The zero value has
// every channel open.
type channelGuard struct {
	closed map[string]bool
}

func (g channelGuard) IsChannelClosed(_ context.Context, channelID string) bool {
	return g.closed[channelID]
}

func closedGuard() channelGuard {
	return channelGuard{closed: map[string]bool{closedChannel: true}}
}

// recordingICS4 is the next ICS-4 wrapper on the send path (the rate-limiting keeper
// in the app): it records what reaches it.
type recordingICS4 struct {
	sendCalls     int
	writeAckCalls int
	versionCalls  int
	lastChannel   string
}

func (w *recordingICS4) SendPacket(_ sdk.Context, _ string, sourceChannel string, _ clienttypes.Height, _ uint64, _ []byte) (uint64, error) {
	w.sendCalls++
	w.lastChannel = sourceChannel

	return 42, nil
}

func (w *recordingICS4) WriteAcknowledgement(_ sdk.Context, _ ibcexported.PacketI, _ ibcexported.Acknowledgement) error {
	w.writeAckCalls++

	return nil
}

func (w *recordingICS4) GetAppVersion(_ sdk.Context, _, _ string) (string, bool) {
	w.versionCalls++

	return ibctransfertypes.V1, true
}

func gateContext() sdk.Context {
	return testutil.DefaultContext(storetypes.NewKVStoreKey(types.ModuleName), storetypes.NewTransientStoreKey("transient_test"))
}

// setupClosed is the middleware with closedChannel closed and no blocked denom.
func setupClosed(t *testing.T) (sdk.Context, ibcmiddleware.IBCMiddleware, *recordingApp, *recordingICS4) {
	t.Helper()

	app := &recordingApp{}
	ics4 := &recordingICS4{}
	im := ibcmiddleware.NewIBCMiddleware(app, ics4, paramsKeeper{params: types.DefaultParams()}, closedGuard())

	return gateContext(), im, app, ics4
}

// outgoingPacket is a transfer this chain sent on sourceChannel.
func outgoingPacket(sourceChannel string) channeltypes.Packet {
	return channeltypes.NewPacket(
		ibctransfertypes.NewFungibleTokenPacketData("ubze", "1000", "bze1sender", "cosmos1receiver", "").GetBytes(),
		1,
		ibctransfertypes.PortID,
		sourceChannel,
		ibctransfertypes.PortID,
		counterpartyChannel,
		clienttypes.NewHeight(1, 1000),
		0,
	)
}

// Every transfer arriving on a closed channel is refused with an error acknowledgement,
// so the sending chain refunds it and nothing is minted or unescrowed here.
func TestClosedChannel_RecvIsRefused(t *testing.T) {
	ctx, im, app, _ := setupClosed(t)

	ack := im.OnRecvPacket(ctx, ibctransfertypes.V1, transferPacket("uatom", closedChannel), sdk.AccAddress{})

	require.NotNil(t, ack)
	require.False(t, ack.Success())
	require.Equal(t, 0, app.recvCalls)
}

// The channel switch has no exception for BZE-origin tokens coming home: the whole
// channel is stopped, returning vouchers included.
func TestClosedChannel_RecvOfReturningTokensIsRefused(t *testing.T) {
	ctx, im, app, _ := setupClosed(t)

	returning := ibctransfertypes.GetPrefixedDenom(ibctransfertypes.PortID, counterpartyChannel, "ubze")
	ack := im.OnRecvPacket(ctx, ibctransfertypes.V1, transferPacket(returning, closedChannel), sdk.AccAddress{})

	require.False(t, ack.Success())
	require.Equal(t, 0, app.recvCalls)
}

// Packets that do not parse as ICS-20 do not slip through a closed channel either.
func TestClosedChannel_RecvOfUnparsablePacketIsRefused(t *testing.T) {
	ctx, im, app, _ := setupClosed(t)

	packet := transferPacket("uatom", closedChannel)
	packet.Data = []byte("not a transfer packet")

	require.False(t, im.OnRecvPacket(ctx, ibctransfertypes.V1, packet, sdk.AccAddress{}).Success())
	require.Equal(t, 0, app.recvCalls)
}

// The error acknowledgement carries the closed-channel error, not the blocked-denom one.
func TestClosedChannel_RecvAckNamesTheClosedChannel(t *testing.T) {
	ctx, im, _, _ := setupClosed(t)

	ack := im.OnRecvPacket(ctx, ibctransfertypes.V1, transferPacket("uatom", closedChannel), sdk.AccAddress{})

	require.Equal(t, channeltypes.NewErrorAcknowledgement(types.ErrChannelClosed), ack)
	require.Empty(t, blockedEvents(ctx))
}

// Other channels are untouched while one is closed.
func TestClosedChannel_RecvOnOpenChannelPasses(t *testing.T) {
	ctx, im, app, _ := setupClosed(t)

	ack := im.OnRecvPacket(ctx, ibctransfertypes.V1, transferPacket("uatom", openChannel), sdk.AccAddress{})

	require.True(t, ack.Success())
	require.Equal(t, 1, app.recvCalls)
}

// Acknowledgements and timeouts of transfers sent before the close must still reach the
// transfer module, otherwise in-flight transfers could never be refunded.
func TestClosedChannel_AckAndTimeoutStillReachTheApp(t *testing.T) {
	ctx, im, app, _ := setupClosed(t)
	packet := outgoingPacket(closedChannel)

	errAck := channeltypes.NewErrorAcknowledgement(types.ErrChannelClosed).Acknowledgement()
	require.NoError(t, im.OnAcknowledgementPacket(ctx, ibctransfertypes.V1, packet, errAck, sdk.AccAddress{}))
	require.Equal(t, 1, app.ackCalls)

	require.NoError(t, im.OnTimeoutPacket(ctx, ibctransfertypes.V1, packet, sdk.AccAddress{}))
	require.Equal(t, 1, app.timeoutCalls)
}

// A send on a closed channel fails before reaching the next wrapper, so the transfer
// transaction fails and nothing is escrowed.
func TestClosedChannel_MiddlewareSendFails(t *testing.T) {
	ctx, im, _, ics4 := setupClosed(t)

	_, err := im.SendPacket(ctx, ibctransfertypes.PortID, closedChannel, clienttypes.NewHeight(1, 1000), 0, []byte("data"))

	require.ErrorIs(t, err, types.ErrChannelClosed)
	require.Equal(t, 0, ics4.sendCalls)
}

func TestClosedChannel_MiddlewareSendOnOpenChannelPasses(t *testing.T) {
	ctx, im, _, ics4 := setupClosed(t)

	seq, err := im.SendPacket(ctx, ibctransfertypes.PortID, openChannel, clienttypes.NewHeight(1, 1000), 0, []byte("data"))

	require.NoError(t, err)
	require.Equal(t, uint64(42), seq)
	require.Equal(t, 1, ics4.sendCalls)
	require.Equal(t, openChannel, ics4.lastChannel)
}

// The send gate is the ICS-4 wrapper the transfer keeper sends through in the app.
func TestSendGate_SendOnClosedChannelFails(t *testing.T) {
	ics4 := &recordingICS4{}
	gate := ibcmiddleware.NewSendGate(ics4, closedGuard())

	_, err := gate.SendPacket(gateContext(), ibctransfertypes.PortID, closedChannel, clienttypes.NewHeight(1, 1000), 0, []byte("data"))

	require.ErrorIs(t, err, types.ErrChannelClosed)
	require.Equal(t, 0, ics4.sendCalls)
}

func TestSendGate_SendOnOpenChannelPasses(t *testing.T) {
	ics4 := &recordingICS4{}
	gate := ibcmiddleware.NewSendGate(ics4, closedGuard())

	seq, err := gate.SendPacket(gateContext(), ibctransfertypes.PortID, openChannel, clienttypes.NewHeight(1, 1000), 0, []byte("data"))

	require.NoError(t, err)
	require.Equal(t, uint64(42), seq)
	require.Equal(t, 1, ics4.sendCalls)
}

// With no channel closed the gate is transparent.
func TestSendGate_NothingClosedIsTransparent(t *testing.T) {
	ics4 := &recordingICS4{}
	gate := ibcmiddleware.NewSendGate(ics4, channelGuard{})

	_, err := gate.SendPacket(gateContext(), ibctransfertypes.PortID, closedChannel, clienttypes.NewHeight(1, 1000), 0, []byte("data"))

	require.NoError(t, err)
	require.Equal(t, 1, ics4.sendCalls)
}

// Writing acknowledgements and reading the app version are not sends: they pass
// through even on a closed channel.
func TestSendGate_OtherICS4CallsPassThrough(t *testing.T) {
	ics4 := &recordingICS4{}
	gate := ibcmiddleware.NewSendGate(ics4, closedGuard())
	ctx := gateContext()

	packet := outgoingPacket(closedChannel)
	require.NoError(t, gate.WriteAcknowledgement(ctx, packet, channeltypes.NewResultAcknowledgement([]byte{1})))
	require.Equal(t, 1, ics4.writeAckCalls)

	version, ok := gate.GetAppVersion(ctx, ibctransfertypes.PortID, closedChannel)
	require.True(t, ok)
	require.Equal(t, ibctransfertypes.V1, version)
	require.Equal(t, 1, ics4.versionCalls)
}
