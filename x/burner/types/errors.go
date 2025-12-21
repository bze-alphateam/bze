package types

// DONTCOVER

import (
	sdkerrors "cosmossdk.io/errors"
)

// x/burner module sentinel errors
var (
	ErrInvalidSigner     = sdkerrors.Register(ModuleName, 1100, "expected gov account as only signer for proposal message")
	ErrInvalidBurnAmount = sdkerrors.Register(ModuleName, 1101, "invalid burn amount")
)
