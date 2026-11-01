package keeper

import (
	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	"github.com/bze-alphateam/bze/x/tradebin/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// The owner-price index (BZE-249) points at the order an owner rests at a market, side and price, so a
// new order of that owner at that price is merged into it. It is derived state: SaveOrder and
// RemoveOrder keep it, genesis init and the v5 migration rebuild it from the orders.
//
// Orders placed before v8.3.0 may hold several orders of one owner at one price. The index points at
// the lowest id of them (the first in FIFO order) and is deleted only by the order it points at; the
// other ones keep resting as separate orders.

func (k Keeper) getOwnerPriceOrderStore(ctx sdk.Context) prefix.Store {
	return k.getPrefixedStore(ctx, types.KeyPrefix(types.OwnerPriceKeyPrefix))
}

// GetOwnerPriceOrder returns the reference of the order the owner rests at the market, side and price.
func (k Keeper) GetOwnerPriceOrder(ctx sdk.Context, owner, marketId, orderType, price string) (ref types.OrderReference, found bool) {
	store := k.getOwnerPriceOrderStore(ctx)
	b := store.Get(types.OwnerPriceOrderKey(owner, marketId, orderType, price))
	if b == nil {
		return ref, false
	}

	k.cdc.MustUnmarshal(b, &ref)

	return ref, true
}

// GetAllOwnerPriceOrder returns every owner-price index entry in key order.
func (k Keeper) GetAllOwnerPriceOrder(ctx sdk.Context) (list []types.OrderReference) {
	store := k.getOwnerPriceOrderStore(ctx)
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.OrderReference
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}

// IndexOwnerPriceOrder points the owner-price index at the order, unless it already points at the
// same or a lower id.
func (k Keeper) IndexOwnerPriceOrder(ctx sdk.Context, order types.Order) {
	existing, found := k.GetOwnerPriceOrder(ctx, order.Owner, order.MarketId, order.OrderType, order.Price)
	if found && !isLowerOrderId(order.Id, existing.Id) {
		return
	}

	ref := types.OrderReference{
		Id:        order.Id,
		MarketId:  order.MarketId,
		OrderType: order.OrderType,
	}
	store := k.getOwnerPriceOrderStore(ctx)
	store.Set(types.OwnerPriceOrderKey(order.Owner, order.MarketId, order.OrderType, order.Price), k.cdc.MustMarshal(&ref))
}

// unindexOwnerPriceOrder deletes the owner-price index entry of the order, only if it points at it.
func (k Keeper) unindexOwnerPriceOrder(ctx sdk.Context, order types.Order) {
	existing, found := k.GetOwnerPriceOrder(ctx, order.Owner, order.MarketId, order.OrderType, order.Price)
	if !found || existing.Id != order.Id {
		return
	}

	store := k.getOwnerPriceOrderStore(ctx)
	store.Delete(types.OwnerPriceOrderKey(order.Owner, order.MarketId, order.OrderType, order.Price))
}

// isLowerOrderId compares two zero-filled order ids numerically.
func isLowerOrderId(id, other string) bool {
	if len(id) != len(other) {
		return len(id) < len(other)
	}

	return id < other
}
