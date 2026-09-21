package types

import (
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

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
