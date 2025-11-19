package types

// DONTCOVER

import (
	sdkerrors "cosmossdk.io/errors"
)

// x/tradebin module sentinel errors
var (
	ErrInvalidSigner          = sdkerrors.Register(ModuleName, 1100, "expected gov account as only signer for proposal message")
	ErrMarketAlreadyExists    = sdkerrors.Register(ModuleName, 4000, "market already exists")
	ErrDenomHasNoSupply       = sdkerrors.Register(ModuleName, 4001, "no supply found for the provided denom")
	ErrInvalidOrderType       = sdkerrors.Register(ModuleName, 4002, "invalid order type")
	ErrInvalidOrderAmount     = sdkerrors.Register(ModuleName, 4003, "invalid order amount")
	ErrInvalidOrderPrice      = sdkerrors.Register(ModuleName, 4004, "invalid order price")
	ErrInvalidOrderMarketId   = sdkerrors.Register(ModuleName, 4005, "invalid order market id")
	ErrMarketNotFound         = sdkerrors.Register(ModuleName, 4006, "market not found")
	ErrInvalidOrderId         = sdkerrors.Register(ModuleName, 4007, "invalid order id")
	ErrOrderNotFound          = sdkerrors.Register(ModuleName, 4008, "order not found")
	ErrUnauthorizedOrder      = sdkerrors.Register(ModuleName, 4009, "not authorized")
	ErrInvalidDenom           = sdkerrors.Register(ModuleName, 4010, "invalid denom provided")
	ErrInvalidOrdersToFill    = sdkerrors.Register(ModuleName, 4011, "invalid orders to fill")
	ErrInvalidFeeDestination  = sdkerrors.Register(ModuleName, 4012, "invalid fee destination")
	ErrNegativeFeeDestination = sdkerrors.Register(ModuleName, 4013, "negative fee destination")
	ErrResultedAmountTooLow   = sdkerrors.Register(ModuleName, 4014, "the resulted amount is too low")
	ErrInvalidRoutes          = sdkerrors.Register(ModuleName, 4015, "invalid routes")
	ErrInvalidPoolSwap        = sdkerrors.Register(ModuleName, 4016, "invalid pool swap")
)
