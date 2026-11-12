package types

// DONTCOVER

import (
	sdkerrors "cosmossdk.io/errors"
)

// x/txfeecollector module sentinel errors
var (
	ErrInvalidSigner = sdkerrors.Register(ModuleName, 1100, "expected gov account as only signer for proposal message")
	// ErrBlockedIbcInbound is returned to the counterparty chain in an error
	// acknowledgement when an inbound ICS-20 packet matches the BlockedIbcInbound
	// param. The sending chain refunds the escrowed tokens when it relays this ack.
	ErrBlockedIbcInbound = sdkerrors.Register(ModuleName, 1101, "inbound ibc transfer of this denom on this channel is blocked by chain parameters")
	// ErrUnsupportedPacketUnmarshal is returned when the application wrapped by the
	// inbound filter cannot unmarshal packet data itself (ADR 008).
	ErrUnsupportedPacketUnmarshal = sdkerrors.Register(ModuleName, 1102, "underlying application does not support packet data unmarshalling")
	// ErrChannelClosed refuses every ICS-20 transfer on a channel closed through
	// x/guardian: a send fails the transaction, a receive gets an error
	// acknowledgement so the sending chain refunds the sender.
	ErrChannelClosed = sdkerrors.Register(ModuleName, 1140, "ibc transfers on this channel are stopped")
)
