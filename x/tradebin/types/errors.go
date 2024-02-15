package types

// DONTCOVER

import (
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// x/tradebin module sentinel errors
var (
	ErrMarketAlreadyExists = sdkerrors.Register(ModuleName, 4000, "market already exists")
	ErrDenomHasNoSupply    = sdkerrors.Register(ModuleName, 4001, "bank module has no supply for the provided denom")
)
