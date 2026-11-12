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

	// 1120-1129: DEX denom halt
	ErrTradebinKeeperNotSet = sdkerrors.Register(ModuleName, 1120, "tradebin keeper is not set")
)
