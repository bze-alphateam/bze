package app

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	ibctransfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
)

// transferChannelLister lists the channels bound to a port. Implemented by the IBC channel keeper.
type transferChannelLister interface {
	GetAllChannelsWithPortPrefix(ctx sdk.Context, portPrefix string) []channeltypes.IdentifiedChannel
}

// transferEscrowChecker recognises the ICS-20 escrow accounts, which ibc-go v10 keeps as plain accounts. txfeecollector
// uses it so the transfer fee never charges an escrow.
//
// It derives the escrow address of every channel on the transfer port, on each fee-bearing factory-token send. That is
// cheap only because the chain has a handful of transfer channels and the check runs after the "fee > 0" early
// return: don't reuse this pattern on a hot path or on a chain with many channels.
type transferEscrowChecker struct {
	channels transferChannelLister
}

func newTransferEscrowChecker(channels transferChannelLister) transferEscrowChecker {
	return transferEscrowChecker{channels: channels}
}

// IsEscrowAddress reports whether addr is the escrow account of a channel on the transfer port.
func (c transferEscrowChecker) IsEscrowAddress(ctx sdk.Context, addr sdk.AccAddress) bool {
	for _, ch := range c.channels.GetAllChannelsWithPortPrefix(ctx, ibctransfertypes.PortID) {
		if ch.PortId != ibctransfertypes.PortID {
			continue
		}

		if ibctransfertypes.GetEscrowAddress(ch.PortId, ch.ChannelId).Equals(addr) {
			return true
		}
	}

	return false
}
