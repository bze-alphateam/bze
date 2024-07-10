package keeper

import (
	"github.com/bze-alphateam/bze/x/tradebin/types"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k Keeper) getOrderStore(ctx sdk.Context) sdk.KVStore {
	return prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.OrderKeyPrefix))
}

func (k Keeper) getUserOrderStore(ctx sdk.Context) sdk.KVStore {
	return prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.UserOrderKeyPrefix))
}

func (k Keeper) getUserOrderByAddressStore(ctx sdk.Context, address string) sdk.KVStore {
	return prefix.NewStore(ctx.KVStore(k.storeKey), types.UserOrderByUserPrefix(address))
}

func (k Keeper) getUserOrderByAddressAndMarketStore(ctx sdk.Context, address, marketId string) sdk.KVStore {
	return prefix.NewStore(ctx.KVStore(k.storeKey), types.UserOrderByUserAndMarketPrefix(address, marketId))
}

func (k Keeper) getPriceOrderStore(ctx sdk.Context) sdk.KVStore {
	return prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.PriceOrderKeyPrefix))
}

func (k Keeper) getAggregatedOrderStore(ctx sdk.Context) sdk.KVStore {
	return prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.AggOrderKeyPrefix))
}

func (k Keeper) getAggregatedOrderByMarketAndTypeStore(ctx sdk.Context, marketId, orderType string) sdk.KVStore {
	return prefix.NewStore(ctx.KVStore(k.storeKey), types.AggOrderByMarketAndTypePrefix(marketId, orderType))
}

func (k Keeper) getOrderCounterStore(ctx sdk.Context) sdk.KVStore {
	return prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.OrderCounterPrefix))
}

func (k Keeper) GetAllOrder(ctx sdk.Context) (list []types.Order) {
	store := k.getOrderStore(ctx)
	iterator := sdk.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.Order
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
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

func (k Keeper) NewOrder(ctx sdk.Context, order types.Order) types.Order {
	defer k.incrementOrderCounter(ctx)

	counter := k.GetOrderCounter(ctx)
	order.Id = k.largeZeroFillId(counter)
	order.CreatedAt = ctx.BlockHeader().Time.Unix()

	k.SaveOrder(ctx, order)

	return order
}

func (k Keeper) SaveOrder(ctx sdk.Context, order types.Order) types.Order {
	store := k.getOrderStore(ctx)
	b := k.cdc.MustMarshal(&order)
	key := types.OrderKey(order.MarketId, order.OrderType, order.Id)
	store.Set(key, b)

	orderRef := types.OrderReference{
		Id:        order.Id,
		MarketId:  order.MarketId,
		OrderType: order.OrderType,
	}

	k.SetPriceOrder(ctx, orderRef, order.Price)
	k.SetUserOrder(ctx, orderRef, order.Owner)

	return order
}

func (k Keeper) RemoveOrder(ctx sdk.Context, order types.Order) {
	store := k.getOrderStore(ctx)
	key := types.OrderKey(order.MarketId, order.OrderType, order.Id)
	store.Delete(key)

	k.RemovePriceOrder(ctx, order.MarketId, order.OrderType, order.Price, order.Id)
	k.RemoveUserOrder(ctx, order.Owner, order.MarketId, order.OrderType, order.Id)
}

func (k Keeper) GetAllPriceOrder(ctx sdk.Context) (list []types.OrderReference) {
	store := k.getPriceOrderStore(ctx)
	iterator := sdk.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.OrderReference
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}

func (k Keeper) SetPriceOrder(ctx sdk.Context, order types.OrderReference, price string) {
	store := k.getPriceOrderStore(ctx)
	b := k.cdc.MustMarshal(&order)
	key := types.PriceOrderKey(order.MarketId, order.OrderType, price, order.Id)
	store.Set(key, b)
}

func (k Keeper) RemovePriceOrder(ctx sdk.Context, marketId, orderType, price, orderId string) {
	store := k.getPriceOrderStore(ctx)
	key := types.PriceOrderKey(marketId, orderType, price, orderId)
	store.Delete(key)
}

func (k Keeper) GetPriceOrderByPrice(ctx sdk.Context, marketId, orderType, price string) (list []types.OrderReference) {
	store := k.getPriceOrderStore(ctx)
	iterator := sdk.KVStorePrefixIterator(store, types.PriceOrderPrefixKey(marketId, orderType, price))

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.OrderReference
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}

func (k Keeper) GetAllUserOrder(ctx sdk.Context) (list []types.OrderReference) {
	store := k.getUserOrderStore(ctx)
	iterator := sdk.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.OrderReference
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}

func (k Keeper) SetUserOrder(ctx sdk.Context, order types.OrderReference, userAddress string) {
	store := k.getUserOrderStore(ctx)
	b := k.cdc.MustMarshal(&order)
	key := types.UserOrderKey(userAddress, order.MarketId, order.OrderType, order.Id)
	store.Set(key, b)
}

func (k Keeper) RemoveUserOrder(ctx sdk.Context, userAddress, marketId, orderType, orderId string) {
	store := k.getUserOrderStore(ctx)
	key := types.UserOrderKey(userAddress, marketId, orderType, orderId)
	store.Delete(key)
}

func (k Keeper) GetAllAggregatedOrder(ctx sdk.Context) (list []types.AggregatedOrder) {
	store := k.getAggregatedOrderStore(ctx)
	iterator := sdk.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.AggregatedOrder
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}

func (k Keeper) SetAggregatedOrder(ctx sdk.Context, order types.AggregatedOrder) {
	store := k.getAggregatedOrderStore(ctx)
	b := k.cdc.MustMarshal(&order)
	key := types.AggOrderKey(order.MarketId, order.OrderType, order.Price)
	store.Set(key, b)
}

func (k Keeper) GetAggregatedOrder(ctx sdk.Context, marketId, orderType, price string) (order types.AggregatedOrder, found bool) {
	store := k.getAggregatedOrderStore(ctx)
	key := types.AggOrderKey(marketId, orderType, price)
	b := store.Get(key)
	if b == nil {
		return order, false
	}

	k.cdc.MustUnmarshal(b, &order)

	return order, true
}

func (k Keeper) RemoveAggregatedOrder(ctx sdk.Context, order types.AggregatedOrder) {
	key := types.AggOrderKey(order.MarketId, order.OrderType, order.Price)
	store := k.getAggregatedOrderStore(ctx)
	store.Delete(key)
}
