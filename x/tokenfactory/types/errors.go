package types

// DONTCOVER

import (
	"fmt"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// x/tokenfactory module sentinel errors
var (
	ErrDenomExists           = sdkerrors.Register(ModuleName, 2, "attempting to create a denom that already exists (has bank metadata)")
	ErrUnauthorized          = sdkerrors.Register(ModuleName, 3, "unauthorized account")
	ErrInvalidDenom          = sdkerrors.Register(ModuleName, 4, "invalid denom")
	ErrInvalidAmount         = sdkerrors.Register(ModuleName, 5, "invalid amount")
	ErrInvalidCreator        = sdkerrors.Register(ModuleName, 6, "invalid creator")
	ErrSubdenomTooLong       = sdkerrors.Register(ModuleName, 7, fmt.Sprintf("subdenom too long, max length is %d bytes", MaxSubdenomLength))
	ErrCreatorTooLong        = sdkerrors.Register(ModuleName, 8, fmt.Sprintf("creator too long, max length is %d bytes", MaxCreatorLength))
	ErrDenomDoesNotExist     = sdkerrors.Register(ModuleName, 9, "denom does not exist")
	ErrBurnFromModuleAccount = sdkerrors.Register(ModuleName, 10, "burning from Module Account is not allowed")
	ErrInvalidSubdenom       = sdkerrors.Register(ModuleName, 11, "invalid subdenom")
)
