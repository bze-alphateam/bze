package types

// DONTCOVER

import (
	sdkerrors "cosmossdk.io/errors"
)

// x/burner module sentinel errors
var (
	ErrInvalidSigner     = sdkerrors.Register(ModuleName, 1100, "expected gov account as only signer for proposal message")
	ErrInvalidBurnAmount = sdkerrors.Register(ModuleName, 1101, "invalid burn amount")
	ErrInvalidRequest    = sdkerrors.Register(ModuleName, 1102, "invalid request")
	ErrIbcMoveCooldown   = sdkerrors.Register(ModuleName, 1103, "locked coins of this denom were moved too recently")
	ErrIbcMoveNoCapacity = sdkerrors.Register(ModuleName, 1104, "locked coins of this denom can not be moved to a pool")
)
