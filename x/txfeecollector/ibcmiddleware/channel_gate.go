package ibcmiddleware

import (
	"context"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	porttypes "github.com/cosmos/ibc-go/v10/modules/core/05-port/types"
	ibcexported "github.com/cosmos/ibc-go/v10/modules/core/exported"

	"github.com/bze-alphateam/bze/x/txfeecollector/types"
)

// ChannelGuard reports whether every ICS-20 transfer on a channel of this chain is
// stopped. The app wires the txfeecollector keeper here, which reads the x/guardian
// closed channels through its GuardianKeeper; this package never imports x/guardian.
type ChannelGuard interface {
	IsChannelClosed(ctx context.Context, channelID string) bool
}

// SendGate is the ICS-4 wrapper the transfer keeper sends its packets through. It
// refuses a send on a closed channel before the packet reaches the next wrapper (the
// rate-limiting keeper in the app), so the transfer transaction fails, nothing is
// escrowed and no quota is counted. Every other ICS-4 call passes straight through.
type SendGate struct {
	ics4  porttypes.ICS4Wrapper
	guard ChannelGuard
}

var _ porttypes.ICS4Wrapper = SendGate{}

// NewSendGate wraps ics4 with the closed-channel check.
func NewSendGate(ics4 porttypes.ICS4Wrapper, guard ChannelGuard) SendGate {
	return SendGate{ics4: ics4, guard: guard}
}

// SendPacket fails on a closed source channel and forwards anything else.
func (g SendGate) SendPacket(
	ctx sdk.Context,
	sourcePort string,
	sourceChannel string,
	timeoutHeight clienttypes.Height,
	timeoutTimestamp uint64,
	data []byte,
) (uint64, error) {
	if g.guard.IsChannelClosed(ctx, sourceChannel) {
		return 0, errorsmod.Wrapf(types.ErrChannelClosed, "channel %s", sourceChannel)
	}

	return g.ics4.SendPacket(ctx, sourcePort, sourceChannel, timeoutHeight, timeoutTimestamp, data)
}

// WriteAcknowledgement passes through: it is the answer to a packet already received.
func (g SendGate) WriteAcknowledgement(ctx sdk.Context, packet ibcexported.PacketI, ack ibcexported.Acknowledgement) error {
	return g.ics4.WriteAcknowledgement(ctx, packet, ack)
}

// GetAppVersion passes through.
func (g SendGate) GetAppVersion(ctx sdk.Context, portID, channelID string) (string, bool) {
	return g.ics4.GetAppVersion(ctx, portID, channelID)
}
