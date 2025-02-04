package types

import sdk "github.com/cosmos/cosmos-sdk/types"

const (
	FillOrdersExtraGas = 5000
)

type OrderCoins struct {
	Coin     sdk.Coin
	Dust     sdk.Dec
	UserDust *UserDust
}

type OrderCoinsArguments struct {
	OrderType    string
	OrderPrice   string
	OrderAmount  sdk.Int
	Market       *Market
	UserAddress  string
	UserReceives bool
}
