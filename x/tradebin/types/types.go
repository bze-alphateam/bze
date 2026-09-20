package types

import (
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// RefundReasonMarketHalted is the QueueMessageRefundedEvent reason set by the EndBlock engine when
// a queued buy/sell/fill message is refunded because its market's base or quote denom is halted.
const RefundReasonMarketHalted = "market_halted"

type MsgCreator interface {
	GetCreatorAcc() sdk.AccAddress
}

type OrderCoins struct {
	Coin     sdk.Coin
	Dust     math.LegacyDec
	UserDust *UserDust
}

type OrderCoinsArguments struct {
	OrderType    string
	OrderPrice   string
	OrderAmount  math.Int
	Market       *Market
	UserAddress  string
	UserReceives bool
}

func OrderTypeToMessageTypeFill(orderType string) string {
	if orderType == OrderTypeBuy {
		return MessageTypeFillBuy
	}

	return MessageTypeFillSell
}
