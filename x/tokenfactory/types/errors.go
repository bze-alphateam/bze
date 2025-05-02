package types

// DONTCOVER

import (
	sdkerrors "cosmossdk.io/errors"
	"fmt"
)

// x/tokenfactory module sentinel errors
var (
	ErrInvalidSigner         = sdkerrors.Register(ModuleName, 1100, "expected gov account as only signer for proposal message")
	ErrDenomExists           = sdkerrors.Register(ModuleName, 1101, "attempting to create a denom that already exists (has bank metadata)")
	ErrUnauthorized          = sdkerrors.Register(ModuleName, 1102, "unauthorized account")
	ErrInvalidDenom          = sdkerrors.Register(ModuleName, 1103, "invalid denom")
	ErrInvalidAmount         = sdkerrors.Register(ModuleName, 1104, "invalid amount")
	ErrInvalidCreator        = sdkerrors.Register(ModuleName, 1105, "invalid creator")
	ErrSubdenomTooLong       = sdkerrors.Register(ModuleName, 1106, fmt.Sprintf("subdenom too long, max length is %d bytes", MaxSubdenomLength))
	ErrCreatorTooLong        = sdkerrors.Register(ModuleName, 1107, fmt.Sprintf("creator too long, max length is %d bytes", MaxCreatorLength))
	ErrDenomDoesNotExist     = sdkerrors.Register(ModuleName, 1108, "denom does not exist")
	ErrBurnFromModuleAccount = sdkerrors.Register(ModuleName, 1109, "burning from Module Account is not allowed")
	ErrInvalidSubdenom       = sdkerrors.Register(ModuleName, 1110, "invalid subdenom")
)
