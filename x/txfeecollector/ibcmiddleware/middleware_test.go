package ibcmiddleware_test

import (
	"context"
	"strings"
	"testing"

	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/gogoproto/proto"
	capabilitytypes "github.com/cosmos/ibc-go/modules/capability/types"
	ibctransfertypes "github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	porttypes "github.com/cosmos/ibc-go/v8/modules/core/05-port/types"
	ibcexported "github.com/cosmos/ibc-go/v8/modules/core/exported"
	"github.com/stretchr/testify/require"

	"github.com/bze-alphateam/bze/x/txfeecollector/ibcmiddleware"
	"github.com/bze-alphateam/bze/x/txfeecollector/types"
)

const (
	blockedChannel = "channel-3"
	blockedDenom   = "uusdc"
	// counterpartyChannel is the Noble side of the same connection: the source channel
	// carried by every packet that reaches us on blockedChannel.
	counterpartyChannel = "channel-95"
)

// recordingApp is the wrapped IBC application: it records the callbacks it receives so
// the tests can prove what did and did not reach the transfer module.
type recordingApp struct {
	porttypes.IBCModule

	recvCalls    int
	ackCalls     int
	timeoutCalls int
	openAckCalls int
	closeCalls   int

	upgradeInitCalls int
	upgradeTryCalls  int
	upgradeAckCalls  int
	upgradeOpenCalls int
	unmarshalCalls   int
}

// The transfer application implements these two optional interfaces and the ICS-29 fee
// middleware above the filter reaches for them by type assertion, so the fake app
// implements them too.
var (
	_ porttypes.UpgradableModule      = (*recordingApp)(nil)
	_ porttypes.PacketDataUnmarshaler = (*recordingApp)(nil)
)

func (a *recordingApp) OnChanUpgradeInit(_ sdk.Context, _, _ string, _ channeltypes.Order, _ []string, version string) (string, error) {
	a.upgradeInitCalls++

	return version, nil
}

func (a *recordingApp) OnChanUpgradeTry(_ sdk.Context, _, _ string, _ channeltypes.Order, _ []string, version string) (string, error) {
	a.upgradeTryCalls++

	return version, nil
}

func (a *recordingApp) OnChanUpgradeAck(_ sdk.Context, _, _, _ string) error {
	a.upgradeAckCalls++

	return nil
}

func (a *recordingApp) OnChanUpgradeOpen(_ sdk.Context, _, _ string, _ channeltypes.Order, _ []string, _ string) {
	a.upgradeOpenCalls++
}

func (a *recordingApp) UnmarshalPacketData(bz []byte) (interface{}, error) {
	a.unmarshalCalls++

	return string(bz), nil
}

// plainApp implements nothing but porttypes.IBCModule, to prove the filter reports a
// missing route instead of panicking on a nil type assertion.
type plainApp struct {
	porttypes.IBCModule
}

func (a *recordingApp) OnRecvPacket(_ sdk.Context, _ channeltypes.Packet, _ sdk.AccAddress) ibcexported.Acknowledgement {
	a.recvCalls++

	return channeltypes.NewResultAcknowledgement([]byte{byte(1)})
}

func (a *recordingApp) OnAcknowledgementPacket(_ sdk.Context, _ channeltypes.Packet, _ []byte, _ sdk.AccAddress) error {
	a.ackCalls++

	return nil
}

func (a *recordingApp) OnTimeoutPacket(_ sdk.Context, _ channeltypes.Packet, _ sdk.AccAddress) error {
	a.timeoutCalls++

	return nil
}

func (a *recordingApp) OnChanOpenAck(_ sdk.Context, _, _, _, _ string) error {
	a.openAckCalls++

	return nil
}

func (a *recordingApp) OnChanCloseInit(_ sdk.Context, _, _ string) error {
	a.closeCalls++

	return nil
}

// paramsKeeper is the smallest thing the middleware needs: the module params.
type paramsKeeper struct {
	params types.Params
}

func (p paramsKeeper) GetParams(_ context.Context) types.Params {
	return p.params
}

// unusedICS4Wrapper proves the send path is never exercised by the filter: any call to
// it fails the test.
type unusedICS4Wrapper struct {
	t *testing.T
}

func (w unusedICS4Wrapper) SendPacket(sdk.Context, *capabilitytypes.Capability, string, string, clienttypes.Height, uint64, []byte) (uint64, error) {
	w.t.Fatal("SendPacket must not be called by the inbound filter")

	return 0, nil
}

func (w unusedICS4Wrapper) WriteAcknowledgement(sdk.Context, *capabilitytypes.Capability, ibcexported.PacketI, ibcexported.Acknowledgement) error {
	w.t.Fatal("WriteAcknowledgement must not be called by the inbound filter")

	return nil
}

func (w unusedICS4Wrapper) GetAppVersion(sdk.Context, string, string) (string, bool) {
	w.t.Fatal("GetAppVersion must not be called by the inbound filter")

	return "", false
}

func setup(t *testing.T, blocked []types.BlockedIbcTransfer) (sdk.Context, ibcmiddleware.IBCMiddleware, *recordingApp) {
	t.Helper()

	storeKey := storetypes.NewKVStoreKey(types.ModuleName)
	tKey := storetypes.NewTransientStoreKey("transient_test")
	ctx := testutil.DefaultContext(storeKey, tKey)

	params := types.DefaultParams()
	params.BlockedIbcInbound = blocked
	require.NoError(t, params.Validate())

	app := &recordingApp{}
	im := ibcmiddleware.NewIBCMiddleware(app, unusedICS4Wrapper{t: t}, paramsKeeper{params: params})

	return ctx, im, app
}

// transferPacket builds an inbound ICS-20 packet as the counterparty would send it.
func transferPacket(denom, destChannel string) channeltypes.Packet {
	data := ibctransfertypes.NewFungibleTokenPacketData(
		denom,
		"1000000",
		"noble1sender",
		"bze1receiver",
		"",
	)

	return channeltypes.NewPacket(
		data.GetBytes(),
		1,
		ibctransfertypes.PortID,
		counterpartyChannel,
		ibctransfertypes.PortID,
		destChannel,
		clienttypes.NewHeight(1, 1000),
		0,
	)
}

func nobleUsdcBlocked() []types.BlockedIbcTransfer {
	return []types.BlockedIbcTransfer{
		{ChannelId: blockedChannel, BaseDenom: blockedDenom},
	}
}

// blockedEvents returns the typed BlockedIbcInboundEvent events on the context. Typed
// events carry the proto message name as their type.
func blockedEvents(ctx sdk.Context) []sdk.Event {
	evType := proto.MessageName(&types.BlockedIbcInboundEvent{})

	var found []sdk.Event
	for _, event := range ctx.EventManager().Events() {
		if event.Type == evType {
			found = append(found, event)
		}
	}

	return found
}

// A fresh mint of the blocked denom on the blocked channel is refused with an error
// acknowledgement and never reaches the transfer module, so nothing is minted.
func TestOnRecvPacket_BlockedDenomIsRefused(t *testing.T) {
	ctx, im, app := setup(t, nobleUsdcBlocked())

	ack := im.OnRecvPacket(ctx, transferPacket(blockedDenom, blockedChannel), sdk.AccAddress{})

	require.NotNil(t, ack)
	require.False(t, ack.Success())
	require.Equal(t, 0, app.recvCalls)

	events := blockedEvents(ctx)
	require.Len(t, events, 1)

	// typed-event attribute values are JSON encoded, so strings arrive quoted
	attrs := map[string]string{}
	for _, attr := range events[0].Attributes {
		attrs[attr.Key] = strings.Trim(attr.Value, "\"")
	}
	require.Equal(t, blockedChannel, attrs["channel"])
	require.Equal(t, blockedDenom, attrs["denom"])
	require.Equal(t, "1000000", attrs["amount"])
	require.Equal(t, "noble1sender", attrs["sender"])
	require.Equal(t, "bze1receiver", attrs["receiver"])
}

// BZE-origin tokens coming home are unescrowed, not minted: they must pass even on the
// blocked channel, otherwise funds would be stranded on the counterparty chain.
func TestOnRecvPacket_ReturningNativeTokensPass(t *testing.T) {
	ctx, im, app := setup(t, nobleUsdcBlocked())

	returning := ibctransfertypes.GetPrefixedDenom(ibctransfertypes.PortID, counterpartyChannel, "ubze")
	ack := im.OnRecvPacket(ctx, transferPacket(returning, blockedChannel), sdk.AccAddress{})

	require.True(t, ack.Success())
	require.Equal(t, 1, app.recvCalls)
	require.Empty(t, blockedEvents(ctx))
}

// A voucher of the blocked denom coming back home is also a return, not a fresh mint.
func TestOnRecvPacket_ReturningBlockedDenomVoucherPasses(t *testing.T) {
	ctx, im, app := setup(t, nobleUsdcBlocked())

	returning := ibctransfertypes.GetPrefixedDenom(ibctransfertypes.PortID, counterpartyChannel, blockedDenom)
	ack := im.OnRecvPacket(ctx, transferPacket(returning, blockedChannel), sdk.AccAddress{})

	require.True(t, ack.Success())
	require.Equal(t, 1, app.recvCalls)
	require.Empty(t, blockedEvents(ctx))
}

// The same denom arriving on a channel that is not listed is not affected.
func TestOnRecvPacket_OtherChannelPasses(t *testing.T) {
	ctx, im, app := setup(t, nobleUsdcBlocked())

	ack := im.OnRecvPacket(ctx, transferPacket(blockedDenom, "channel-13"), sdk.AccAddress{})

	require.True(t, ack.Success())
	require.Equal(t, 1, app.recvCalls)
	require.Empty(t, blockedEvents(ctx))
}

// Another denom on the blocked channel is not affected either: only the listed pair is.
func TestOnRecvPacket_OtherDenomPasses(t *testing.T) {
	ctx, im, app := setup(t, nobleUsdcBlocked())

	ack := im.OnRecvPacket(ctx, transferPacket("ustake", blockedChannel), sdk.AccAddress{})

	require.True(t, ack.Success())
	require.Equal(t, 1, app.recvCalls)
	require.Empty(t, blockedEvents(ctx))
}

// With an empty param - the state every chain is in right after the migration - the
// middleware is transparent.
func TestOnRecvPacket_EmptyParamIsTransparent(t *testing.T) {
	ctx, im, app := setup(t, nil)

	ack := im.OnRecvPacket(ctx, transferPacket(blockedDenom, blockedChannel), sdk.AccAddress{})

	require.True(t, ack.Success())
	require.Equal(t, 1, app.recvCalls)
	require.Empty(t, blockedEvents(ctx))
}

// Packets the filter cannot read as ICS-20 transfers are none of its business.
func TestOnRecvPacket_NonTransferPacketPasses(t *testing.T) {
	ctx, im, app := setup(t, nobleUsdcBlocked())

	packet := transferPacket(blockedDenom, blockedChannel)
	packet.Data = []byte("not a transfer packet")

	ack := im.OnRecvPacket(ctx, packet, sdk.AccAddress{})

	require.True(t, ack.Success())
	require.Equal(t, 1, app.recvCalls)
	require.Empty(t, blockedEvents(ctx))
}

// The exit path must keep working: acknowledgements and timeouts of packets this chain
// sent reach the transfer module unchanged, whatever the param says.
func TestOutgoingPacketCallbacksPassThrough(t *testing.T) {
	ctx, im, app := setup(t, nobleUsdcBlocked())

	// an outgoing USDC.n withdrawal: this chain is the source, the denom is the voucher
	outgoing := channeltypes.NewPacket(
		ibctransfertypes.NewFungibleTokenPacketData(blockedDenom, "1000000", "bze1sender", "noble1receiver", "").GetBytes(),
		1,
		ibctransfertypes.PortID,
		blockedChannel,
		ibctransfertypes.PortID,
		counterpartyChannel,
		clienttypes.NewHeight(1, 1000),
		0,
	)

	require.NoError(t, im.OnAcknowledgementPacket(ctx, outgoing, channeltypes.NewResultAcknowledgement([]byte{byte(1)}).Acknowledgement(), sdk.AccAddress{}))
	require.Equal(t, 1, app.ackCalls)

	require.NoError(t, im.OnTimeoutPacket(ctx, outgoing, sdk.AccAddress{}))
	require.Equal(t, 1, app.timeoutCalls)

	require.Empty(t, blockedEvents(ctx))
}

// Channel handshake callbacks are pass-through too: the filter never interferes with
// channel lifecycle, so channel-3 can never be closed by it.
func TestHandshakeCallbacksPassThrough(t *testing.T) {
	ctx, im, app := setup(t, nobleUsdcBlocked())

	require.NoError(t, im.OnChanOpenAck(ctx, ibctransfertypes.PortID, blockedChannel, counterpartyChannel, ibctransfertypes.Version))
	require.Equal(t, 1, app.openAckCalls)

	require.NoError(t, im.OnChanCloseInit(ctx, ibctransfertypes.PortID, blockedChannel))
	require.Equal(t, 1, app.closeCalls)
}

// The filter sits between the ICS-29 fee middleware and the transfer application, and
// the fee middleware finds the transfer module's optional interfaces by type-asserting
// the module it wraps. Were these not forwarded, channel upgrades on the transfer stack
// would fail with "upgrade route not found" and ADR-008 unmarshalling would break.
func TestUpgradeCallbacksForwardToTheApp(t *testing.T) {
	ctx, im, app := setup(t, nobleUsdcBlocked())

	hops := []string{"connection-0"}

	version, err := im.OnChanUpgradeInit(ctx, ibctransfertypes.PortID, blockedChannel, channeltypes.UNORDERED, hops, ibctransfertypes.Version)
	require.NoError(t, err)
	require.Equal(t, ibctransfertypes.Version, version)
	require.Equal(t, 1, app.upgradeInitCalls)

	version, err = im.OnChanUpgradeTry(ctx, ibctransfertypes.PortID, blockedChannel, channeltypes.UNORDERED, hops, ibctransfertypes.Version)
	require.NoError(t, err)
	require.Equal(t, ibctransfertypes.Version, version)
	require.Equal(t, 1, app.upgradeTryCalls)

	require.NoError(t, im.OnChanUpgradeAck(ctx, ibctransfertypes.PortID, blockedChannel, ibctransfertypes.Version))
	require.Equal(t, 1, app.upgradeAckCalls)

	im.OnChanUpgradeOpen(ctx, ibctransfertypes.PortID, blockedChannel, channeltypes.UNORDERED, hops, ibctransfertypes.Version)
	require.Equal(t, 1, app.upgradeOpenCalls)
}

func TestUnmarshalPacketDataForwardsToTheApp(t *testing.T) {
	_, im, app := setup(t, nobleUsdcBlocked())

	got, err := im.UnmarshalPacketData([]byte("packet"))
	require.NoError(t, err)
	require.Equal(t, "packet", got)
	require.Equal(t, 1, app.unmarshalCalls)
}

// An application that implements neither optional interface produces the same errors
// the ICS-29 fee middleware produces in that situation, never a nil dereference.
func TestOptionalInterfacesMissingOnTheApp(t *testing.T) {
	storeKey := storetypes.NewKVStoreKey(types.ModuleName)
	tKey := storetypes.NewTransientStoreKey("transient_test")
	ctx := testutil.DefaultContext(storeKey, tKey)

	im := ibcmiddleware.NewIBCMiddleware(plainApp{}, unusedICS4Wrapper{t: t}, paramsKeeper{params: types.DefaultParams()})
	hops := []string{"connection-0"}

	_, err := im.OnChanUpgradeInit(ctx, ibctransfertypes.PortID, blockedChannel, channeltypes.UNORDERED, hops, ibctransfertypes.Version)
	require.Error(t, err)

	_, err = im.OnChanUpgradeTry(ctx, ibctransfertypes.PortID, blockedChannel, channeltypes.UNORDERED, hops, ibctransfertypes.Version)
	require.Error(t, err)

	require.Error(t, im.OnChanUpgradeAck(ctx, ibctransfertypes.PortID, blockedChannel, ibctransfertypes.Version))

	require.Panics(t, func() {
		im.OnChanUpgradeOpen(ctx, ibctransfertypes.PortID, blockedChannel, channeltypes.UNORDERED, hops, ibctransfertypes.Version)
	})

	_, err = im.UnmarshalPacketData([]byte("packet"))
	require.Error(t, err)
}
