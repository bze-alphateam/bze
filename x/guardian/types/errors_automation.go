package types

// DONTCOVER

import (
	sdkerrors "cosmossdk.io/errors"
)

// x/guardian protocol automation errors
var (
	ErrAlreadyPaused = sdkerrors.Register(ModuleName, 1130, "protocol automation is already paused")
	ErrNotPaused     = sdkerrors.Register(ModuleName, 1131, "protocol automation is not paused")
)
