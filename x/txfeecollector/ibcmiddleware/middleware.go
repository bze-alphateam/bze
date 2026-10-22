// Package ibcmiddleware holds the ICS-20 inbound filter: a porttypes.Middleware that
// refuses inbound transfers listed in the x/txfeecollector BlockedIbcInbound param.
//
// It exists for the Noble USDC wind-down (BZE-143). Circle is retiring USDC on Noble,
// so new USDC.n must stop arriving on BZE while every exit path stays open. ICS-20
// escrows on the sending chain and mints on the receiving chain only once the receiver
// accepts the packet: answering with an error acknowledgement means nothing is minted
// here and the sending chain refunds the sender when the ack is relayed.
//
// Only OnRecvPacket inspects anything. Every other callback delegates to the wrapped
// application, so outgoing transfers, their acknowledgements and their timeout refunds
// behave exactly as they did before this middleware existed.
package ibcmiddleware

import (
	"context"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	ibctransfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	porttypes "github.com/cosmos/ibc-go/v10/modules/core/05-port/types"
	ibcexported "github.com/cosmos/ibc-go/v10/modules/core/exported"

	"github.com/bze-alphateam/bze/x/txfeecollector/types"
)

// ParamsKeeper is the slice of the x/txfeecollector keeper the filter needs: the
// module params holding the blocked (channel, base denom) pairs.
type ParamsKeeper interface {
	GetParams(ctx context.Context) types.Params
}

// IBCMiddleware wraps an IBC application and filters the packets it receives.
type IBCMiddleware struct {
	app    porttypes.IBCModule
	ics4   porttypes.ICS4Wrapper
	keeper ParamsKeeper
}

var _ porttypes.Middleware = (*IBCMiddleware)(nil)

// NewIBCMiddleware wraps app with the inbound filter.
//
// ics4 is only used by the ICS4Wrapper methods, which this middleware passes straight
// through: the send path of the transfer module is wired to the IBC channel keeper
// directly, so nothing this middleware does can affect outgoing packets.
func NewIBCMiddleware(app porttypes.IBCModule, ics4 porttypes.ICS4Wrapper, keeper ParamsKeeper) IBCMiddleware {
	return IBCMiddleware{
		app:    app,
		ics4:   ics4,
		keeper: keeper,
	}
}

// OnRecvPacket refuses the packet with an error acknowledgement when it is a fresh
// mint of a blocked (destination channel, base denom) pair. Anything else - including
// packets this middleware cannot parse as an ICS-20 transfer - reaches the wrapped
// application untouched.
func (im IBCMiddleware) OnRecvPacket(ctx sdk.Context, channelVersion string, packet channeltypes.Packet, relayer sdk.AccAddress) ibcexported.Acknowledgement {
	var data ibctransfertypes.FungibleTokenPacketData
	if err := ibctransfertypes.ModuleCdc.UnmarshalJSON(packet.GetData(), &data); err != nil {
		// not an ICS-20 transfer packet: let the application decide what to do with it
		return im.app.OnRecvPacket(ctx, channelVersion, packet, relayer)
	}

	// Tokens that originally left this chain are unescrowed on the way back, not
	// minted, so returning vouchers must always be accepted - blocking them would
	// strand BZE-origin funds on the counterparty chain.
	//nolint:staticcheck // the plain string-prefix check is kept on purpose: same answer as before ibc-go v10
	if ibctransfertypes.ReceiverChainIsSource(packet.GetSourcePort(), packet.GetSourceChannel(), data.Denom) {
		return im.app.OnRecvPacket(ctx, channelVersion, packet, relayer)
	}

	if !im.keeper.GetParams(ctx).IsInboundBlocked(packet.GetDestChannel(), data.Denom) {
		return im.app.OnRecvPacket(ctx, channelVersion, packet, relayer)
	}

	err := ctx.EventManager().EmitTypedEvent(&types.BlockedIbcInboundEvent{
		Channel:  packet.GetDestChannel(),
		Denom:    data.Denom,
		Amount:   data.Amount,
		Sender:   data.Sender,
		Receiver: data.Receiver,
	})
	if err != nil {
		// the packet is refused either way: an event we could not emit must not turn
		// into a failed acknowledgement, which is a different outcome for the sender
		ctx.Logger().Error("could not emit blocked inbound ibc transfer event", "error", err)
	}

	ctx.Logger().Info(
		"refused blocked inbound ibc transfer",
		"channel", packet.GetDestChannel(),
		"denom", data.Denom,
		"amount", data.Amount,
		"sequence", packet.GetSequence(),
	)

	return channeltypes.NewErrorAcknowledgement(types.ErrBlockedIbcInbound)
}

// OnAcknowledgementPacket passes through: it carries the counterparty's answer to a
// packet this chain sent, including the refunds of blocked transfers we bounced back.
func (im IBCMiddleware) OnAcknowledgementPacket(ctx sdk.Context, channelVersion string, packet channeltypes.Packet, acknowledgement []byte, relayer sdk.AccAddress) error {
	return im.app.OnAcknowledgementPacket(ctx, channelVersion, packet, acknowledgement, relayer)
}

// OnTimeoutPacket passes through so outgoing transfers are refunded as usual.
func (im IBCMiddleware) OnTimeoutPacket(ctx sdk.Context, channelVersion string, packet channeltypes.Packet, relayer sdk.AccAddress) error {
	return im.app.OnTimeoutPacket(ctx, channelVersion, packet, relayer)
}

func (im IBCMiddleware) OnChanOpenInit(
	ctx sdk.Context,
	order channeltypes.Order,
	connectionHops []string,
	portID string,
	channelID string,
	counterparty channeltypes.Counterparty,
	version string,
) (string, error) {
	return im.app.OnChanOpenInit(ctx, order, connectionHops, portID, channelID, counterparty, version)
}

func (im IBCMiddleware) OnChanOpenTry(
	ctx sdk.Context,
	order channeltypes.Order,
	connectionHops []string,
	portID string,
	channelID string,
	counterparty channeltypes.Counterparty,
	counterpartyVersion string,
) (string, error) {
	return im.app.OnChanOpenTry(ctx, order, connectionHops, portID, channelID, counterparty, counterpartyVersion)
}

func (im IBCMiddleware) OnChanOpenAck(ctx sdk.Context, portID, channelID, counterpartyChannelID, counterpartyVersion string) error {
	return im.app.OnChanOpenAck(ctx, portID, channelID, counterpartyChannelID, counterpartyVersion)
}

func (im IBCMiddleware) OnChanOpenConfirm(ctx sdk.Context, portID, channelID string) error {
	return im.app.OnChanOpenConfirm(ctx, portID, channelID)
}

func (im IBCMiddleware) OnChanCloseInit(ctx sdk.Context, portID, channelID string) error {
	return im.app.OnChanCloseInit(ctx, portID, channelID)
}

func (im IBCMiddleware) OnChanCloseConfirm(ctx sdk.Context, portID, channelID string) error {
	return im.app.OnChanCloseConfirm(ctx, portID, channelID)
}

// SendPacket implements the ICS4Wrapper interface: pass through, the filter never
// touches outgoing packets.
func (im IBCMiddleware) SendPacket(
	ctx sdk.Context,
	sourcePort string,
	sourceChannel string,
	timeoutHeight clienttypes.Height,
	timeoutTimestamp uint64,
	data []byte,
) (sequence uint64, err error) {
	return im.ics4.SendPacket(ctx, sourcePort, sourceChannel, timeoutHeight, timeoutTimestamp, data)
}

// WriteAcknowledgement implements the ICS4Wrapper interface: pass through.
func (im IBCMiddleware) WriteAcknowledgement(
	ctx sdk.Context,
	packet ibcexported.PacketI,
	ack ibcexported.Acknowledgement,
) error {
	return im.ics4.WriteAcknowledgement(ctx, packet, ack)
}

// GetAppVersion implements the ICS4Wrapper interface: pass through.
func (im IBCMiddleware) GetAppVersion(ctx sdk.Context, portID, channelID string) (string, bool) {
	return im.ics4.GetAppVersion(ctx, portID, channelID)
}

// The transfer application implements the optional PacketDataUnmarshaler interface
// (ADR 008), which callers reach through a type assertion on the module routed for
// the port. Sitting on top of the transfer application, the filter must forward it,
// otherwise packet unmarshalling would stop working for the transfer stack. (ibc-go
// v10 removed channel upgradability, so UpgradableModule no longer exists.)
var _ porttypes.PacketDataUnmarshaler = (*IBCMiddleware)(nil)

// UnmarshalPacketData forwards to the wrapped application, which owns the packet
// format. This implements the optional PacketDataUnmarshaler interface (ADR 008).
func (im IBCMiddleware) UnmarshalPacketData(ctx sdk.Context, portID, channelID string, bz []byte) (any, string, error) {
	unmarshaler, ok := im.app.(porttypes.PacketDataUnmarshaler)
	if !ok {
		return nil, "", errorsmod.Wrapf(types.ErrUnsupportedPacketUnmarshal, "underlying app does not implement %T", (*porttypes.PacketDataUnmarshaler)(nil))
	}

	return unmarshaler.UnmarshalPacketData(ctx, portID, channelID, bz)
}
