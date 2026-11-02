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
)
