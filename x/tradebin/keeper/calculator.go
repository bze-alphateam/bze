package keeper

import (
	"fmt"
	"github.com/bze-alphateam/bze/x/tradebin/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

func CalculateMinAmount(price string) sdk.Int {
	// Convert the price string to a Dec
	priceDec, err := sdk.NewDecFromStr(price)
	if err != nil {
		fmt.Println("Error converting price to Dec:", err)
		return sdk.NewInt(0)
	}
	if priceDec.IsZero() {
		return sdk.NewInt(0)
	}

	// The denominator for our operation, represented as a Dec
	oneDec := sdk.NewDec(1)

	// Perform the division (1 / price), ensuring high precision
	amtDec := oneDec.Quo(priceDec)
	// Ceil the result to ensure we avoid dust effectively
	amtDec = amtDec.Ceil()

	// Multiply by 2 to adjust for potential dust and lower loss,
	// as described in your comment.
	amtDec = amtDec.MulInt64(2)

	return amtDec.TruncateInt()
}

// GetOrderSdkCoin - returns the needed coins for an order
// When the user submits an order we have to capture the coins needed for that order to be placed.
// This function returns the sdk.Coins that we have to send from user's account to module and back

// GetOrderSdkCoin - returns the needed coins for an order
// When the user submits an order we have to capture the coins needed for that order to be placed.
// This function returns the sdk.Coins that we have to send from user's account to module and back
func (k Keeper) GetOrderSdkCoin(orderType, orderPrice string, orderAmount sdk.Int, market *types.Market) (coin sdk.Coin, dust sdk.Dec, err error) {
	var amount sdk.Int
	var denom string
	switch orderType {
	case types.OrderTypeBuy:
		denom = market.Quote
		oAmount := orderAmount.ToDec()
		oPrice, err := sdk.NewDecFromStr(orderPrice)
		if err != nil {
			return coin, dust, sdkerrors.Wrapf(types.ErrInvalidOrderPrice, "error when transforming order price: %v", err)
		}
		oAmount = oAmount.Mul(oPrice)
		dust = oAmount
		oAmount = oAmount.TruncateDec()
		dust = dust.Sub(oAmount)
		amount = oAmount.TruncateInt()
	case types.OrderTypeSell:
		denom = market.Base
		amount = orderAmount
		dust = sdk.ZeroDec()
	default:
		return coin, dust, types.ErrInvalidOrderType
	}

	if amount.LT(sdk.ZeroInt()) {
		return coin, dust, sdkerrors.Wrapf(types.ErrInvalidOrderAmount, "order amount is too low for this price")
	}

	coin = sdk.NewCoin(denom, amount)

	return coin, dust, nil
}

func (k Keeper) GetOrderCoinsWithDust(ctx sdk.Context, orderCoinsArgs types.OrderCoinsArguments) (types.OrderCoins, error) {
	coin, dust, err := k.GetOrderSdkCoin(orderCoinsArgs.OrderType, orderCoinsArgs.OrderPrice, orderCoinsArgs.OrderAmount, orderCoinsArgs.Market)
	if err != nil {
		return types.OrderCoins{}, err
	}

	coin, storedUserDust, dust, err := k.CollectUserDust(ctx, orderCoinsArgs.UserAddress, coin, dust, orderCoinsArgs.UserReceives)
	if err != nil {
		return types.OrderCoins{}, err
	}

	return types.OrderCoins{
		Coin:     coin,
		Dust:     dust,
		UserDust: storedUserDust,
	}, nil
}
