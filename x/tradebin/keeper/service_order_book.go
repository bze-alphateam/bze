package keeper

import (
	"fmt"

	"cosmossdk.io/errors"
	"cosmossdk.io/math"
	"github.com/bze-alphateam/bze/x/tradebin/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// CalculateMinAmount - Deprecated: use CalculateMinAmountFromPriceDec
func CalculateMinAmount(price string) (math.Int, error) {
	// Convert the price string to a Dec
	priceDec, err := math.LegacyNewDecFromStr(price)
	if err != nil {
		return math.ZeroInt(), fmt.Errorf("error converting price to Dec: %w", err)
	}
	if priceDec.IsZero() {
		return math.ZeroInt(), fmt.Errorf("price cannot be zero")
	}

	// The denominator for our operation, represented as a Dec
	oneDec := math.LegacyOneDec()

	// Perform the division (1 / price), ensuring high precision
	amtDec := oneDec.Quo(priceDec)
	// Ceil the result to ensure we avoid dust effectively
	amtDec = amtDec.Ceil()

	// Multiply by 2 to adjust for potential dust and lower loss,
	// as described in your comment.
	amtDec = amtDec.MulInt64(2)

	return amtDec.TruncateInt(), nil
}

func CalculateMinAmountFromPriceDec(price math.LegacyDec) (math.Int, error) {
	// The denominator for our operation, represented as a Dec
	oneDec := math.LegacyOneDec()

	// Perform the division (1 / price), ensuring high precision
	amtDec := oneDec.Quo(price)
	// Ceil the result to ensure we avoid dust effectively
	amtDec = amtDec.Ceil()

	// Multiply by 2 to adjust for potential dust and lower loss,
	// as described in your comment.
	amtDec = amtDec.MulInt64(2)

	return amtDec.TruncateInt(), nil
}

// GetOrderSdkCoin - returns the needed coins for an order
// When the user submits an order we have to capture the coins needed for that order to be placed.
// This function returns the sdk.Coins that we have to send from user's account to module and back
func (k Keeper) GetOrderSdkCoin(orderType, orderPrice string, orderAmount math.Int, market *types.Market) (coin sdk.Coin, dust math.LegacyDec, err error) {
	var amount math.Int
	var denom string
	switch orderType {
	case types.OrderTypeBuy:
		denom = market.Quote
		oAmount := math.LegacyNewDecFromInt(orderAmount)
		oPrice, err := math.LegacyNewDecFromStr(orderPrice)
		if err != nil {
			return coin, dust, errors.Wrapf(types.ErrInvalidOrderPrice, "error when transforming order price: %v", err)
		}
		oAmount = oAmount.Mul(oPrice)
		dust = oAmount
		oAmount = oAmount.TruncateDec()
		dust = dust.Sub(oAmount)
		amount = oAmount.TruncateInt()
	case types.OrderTypeSell:
		denom = market.Base
		amount = orderAmount
		dust = math.LegacyZeroDec()
	default:
		return coin, dust, types.ErrInvalidOrderType
	}

	if amount.LT(math.ZeroInt()) {
		return coin, dust, errors.Wrapf(types.ErrInvalidOrderAmount, "order amount is too low for this price")
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
