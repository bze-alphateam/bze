package types

// DONTCOVER

import (
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// x/tradebin module sentinel errors
var (
	ErrMarketAlreadyExists  = sdkerrors.Register(ModuleName, 4000, "market already exists")
	ErrDenomHasNoSupply     = sdkerrors.Register(ModuleName, 4001, "no supply found for the provided denom")
	ErrInvalidOrderType     = sdkerrors.Register(ModuleName, 4002, "invalid order type")
	ErrInvalidOrderAmount   = sdkerrors.Register(ModuleName, 4003, "invalid order amount")
	ErrInvalidOrderPrice    = sdkerrors.Register(ModuleName, 4004, "invalid order price")
	ErrInvalidOrderMarketId = sdkerrors.Register(ModuleName, 4005, "invalid order market id")
	ErrMarketNotFound       = sdkerrors.Register(ModuleName, 4006, "market not found")
	ErrInvalidOrderId       = sdkerrors.Register(ModuleName, 4007, "invalid order id")
	ErrOrderNotFound        = sdkerrors.Register(ModuleName, 4008, "order not found")
	ErrUnauthorizedOrder    = sdkerrors.Register(ModuleName, 4009, "not authorized")
	ErrInvalidDenom         = sdkerrors.Register(ModuleName, 4010, "invalid denom provided")
)
