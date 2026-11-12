package types

// DONTCOVER

import (
	sdkerrors "cosmossdk.io/errors"
)

// x/guardian module sentinel errors
var (
	ErrInvalidSigner         = sdkerrors.Register(ModuleName, 1100, "expected gov account as only signer for proposal message")
	ErrUnauthorized          = sdkerrors.Register(ModuleName, 1101, "signer is neither the guardian nor the governance authority")
	ErrRateLimitNotFound     = sdkerrors.Register(ModuleName, 1102, "rate limit not found")
	ErrQuotaRaise            = sdkerrors.Register(ModuleName, 1103, "a quota can only be lowered")
	ErrInvalidQuota          = sdkerrors.Register(ModuleName, 1104, "invalid quota")
	ErrRateLimitKeeperNotSet = sdkerrors.Register(ModuleName, 1105, "rate limit keeper is not set")
	ErrAlreadyBlocked        = sdkerrors.Register(ModuleName, 1106, "denom is already blocked")
	ErrNotBlocked            = sdkerrors.Register(ModuleName, 1107, "denom is not blocked")
	ErrInvalidDenom          = sdkerrors.Register(ModuleName, 1108, "invalid denom")
	ErrDenomNotFound         = sdkerrors.Register(ModuleName, 1109, "denom does not exist on chain")

	// channel switch: codes 1140-1149
	ErrInvalidChannel       = sdkerrors.Register(ModuleName, 1140, "invalid channel id")
	ErrChannelNotFound      = sdkerrors.Register(ModuleName, 1141, "transfer channel not found")
	ErrChannelAlreadyClosed = sdkerrors.Register(ModuleName, 1142, "channel is already closed")
	ErrChannelNotClosed     = sdkerrors.Register(ModuleName, 1143, "channel is not closed")
	ErrChannelKeeperNotSet  = sdkerrors.Register(ModuleName, 1144, "channel keeper is not set")
)
