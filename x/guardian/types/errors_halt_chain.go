package types

// DONTCOVER

import (
	sdkerrors "cosmossdk.io/errors"
)

// x/guardian chain halt errors. They use their own code range so the other
// guardian switches can add theirs without clashing.
var (
	ErrInvalidHaltReason   = sdkerrors.Register(ModuleName, 1150, "invalid halt reason")
	ErrUpgradeKeeperNotSet = sdkerrors.Register(ModuleName, 1151, "upgrade keeper is not set")
)
