package types

// Events emitted by the module's IBC inbound filter.
const (
	// EventTypeBlockedIbcInbound is emitted when an inbound ICS-20 packet is answered
	// with an error acknowledgement because of the BlockedIbcInbound param.
	EventTypeBlockedIbcInbound = "blocked_ibc_inbound"

	AttributeKeyChannel  = "channel"
	AttributeKeyDenom    = "denom"
	AttributeKeyAmount   = "amount"
	AttributeKeySender   = "sender"
	AttributeKeyReceiver = "receiver"
	AttributeKeyReason   = "reason"
)
