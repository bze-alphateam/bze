package keeper

import (
	"github.com/bze-alphateam/bze/x/tradebin/types"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"math"
	"strconv"
)

func (k Keeper) getOrderStore(ctx sdk.Context) sdk.KVStore {
	return prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.OrderKeyPrefix))
}

func (k Keeper) getUserOrderStore(ctx sdk.Context) sdk.KVStore {
	return prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.UserOrderKeyPrefix))
}

func (k Keeper) getPriceOrderStore(ctx sdk.Context) sdk.KVStore {
	return prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.PriceOrderKeyPrefix))
}

func (k Keeper) getAggregatedOrderStore(ctx sdk.Context) sdk.KVStore {
	return prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.AggOrderKeyPrefix))
}

func (k Keeper) getHistoryOrderStore(ctx sdk.Context) sdk.KVStore {
	return prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.HistoryOrderKeyPrefix))
}

// getCreateOrderNeededCoins - returns the needed coins for an order
// When the user submits an order we have to capture the coins needed for that order to be placed.
// This function returns the sdk.Coins that we have to send from user's account to module
func (k Keeper) getOrderNeededCoins(msg *types.MsgCreateOrder, market *types.Market) (sdk.Coin, error) {
	var amount int64
	var denom string
	var coin sdk.Coin
	switch msg.OrderType {
	case types.OrderTypeBuy:
		denom = market.Quote
		priceFloat, err := strconv.ParseFloat(msg.Price, 64)
		if err != nil {
			return coin, sdkerrors.Wrapf(types.ErrInvalidOrderPrice, "order price float error: %s", err)
		}
		floatAmount := priceFloat * float64(msg.Amount)
		amount = int64(math.Floor(floatAmount))
	case types.OrderTypeSell:
		denom = market.Base
		amount = msg.Amount
	default:
		return coin, types.ErrInvalidOrderType
	}

	if amount <= 0 {
		return coin, sdkerrors.Wrapf(types.ErrInvalidOrderAmount, "order amount is too low for this price")
	}

	coin = sdk.NewInt64Coin(denom, amount)

	return coin, nil
}

func (k Keeper) GetOrder(ctx sdk.Context, marketId, orderType, orderId string) (order types.Order, found bool) {
	store := k.getOrderStore(ctx)

	key := types.OrderKey(marketId, orderType, orderId)
	b := store.Get(key)
	if b == nil {
		return order, false
	}

	k.cdc.MustUnmarshal(b, &order)
	return order, true
}
