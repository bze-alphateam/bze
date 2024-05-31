package keeper

import (
	"github.com/bze-alphateam/bze/x/tradebin/types"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k Keeper) getHistoryOrderStore(ctx sdk.Context) sdk.KVStore {
	return prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.HistoryOrderKeyPrefix))
}

func (k Keeper) getHistoryOrderByMarketStore(ctx sdk.Context, marketId string) sdk.KVStore {
	return prefix.NewStore(ctx.KVStore(k.storeKey), types.HistoryOrderByMarketPrefix(marketId))
}

func (k Keeper) GetAllHistoryOrder(ctx sdk.Context) (list []types.HistoryOrder) {
	store := k.getHistoryOrderStore(ctx)
	iterator := sdk.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.HistoryOrder
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}

func (k Keeper) SetHistoryOrder(ctx sdk.Context, order types.HistoryOrder, index string) {
	store := k.getHistoryOrderStore(ctx)
	b := k.cdc.MustMarshal(&order)
	key := types.HistoryOrderKey(order.MarketId, k.smallZeroFillId(uint64(order.ExecutedAt)), index)
	store.Set(key, b)
}
