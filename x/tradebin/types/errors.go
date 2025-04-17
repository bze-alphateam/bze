package types

// DONTCOVER

import (
	"cosmossdk.io/errors"
)

// x/tradebin module sentinel errors
var (
	ErrMarketAlreadyExists    = errors.Register(ModuleName, 4000, "market already exists")
	ErrDenomHasNoSupply       = errors.Register(ModuleName, 4001, "no supply found for the provided denom")
	ErrInvalidOrderType       = errors.Register(ModuleName, 4002, "invalid order type")
	ErrInvalidOrderAmount     = errors.Register(ModuleName, 4003, "invalid order amount")
	ErrInvalidOrderPrice      = errors.Register(ModuleName, 4004, "invalid order price")
	ErrInvalidOrderMarketId   = errors.Register(ModuleName, 4005, "invalid order market id")
	ErrMarketNotFound         = errors.Register(ModuleName, 4006, "market not found")
	ErrInvalidOrderId         = errors.Register(ModuleName, 4007, "invalid order id")
	ErrOrderNotFound          = errors.Register(ModuleName, 4008, "order not found")
	ErrUnauthorizedOrder      = errors.Register(ModuleName, 4009, "not authorized")
	ErrInvalidDenom           = errors.Register(ModuleName, 4010, "invalid denom provided")
	ErrInvalidOrdersToFill    = errors.Register(ModuleName, 4011, "invalid orders to fill")
	ErrInvalidFeeDestination  = errors.Register(ModuleName, 4012, "invalid fee destination")
	ErrNegativeFeeDestination = errors.Register(ModuleName, 4013, "negative fee destination")
	ErrResultedAmountTooLow   = errors.Register(ModuleName, 4014, "the resulted amount is too low")
	ErrInvalidRoutes          = errors.Register(ModuleName, 4015, "invalid routes")
	ErrInvalidPoolSwap        = errors.Register(ModuleName, 4016, "invalid pool swap")
)
