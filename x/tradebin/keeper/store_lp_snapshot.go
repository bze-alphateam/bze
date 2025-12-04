package keeper

import (
	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	"github.com/bze-alphateam/bze/x/tradebin/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k Keeper) getLpSnapshotStore(ctx sdk.Context) prefix.Store {
	return k.getPrefixedStore(ctx, types.LpSnapshotPrefix())
}

// SetLiquidityPoolSnapshot stores a liquidity pool snapshot in the KV store, associating it with the pool's unique ID.
func (k Keeper) SetLiquidityPoolSnapshot(ctx sdk.Context, pool types.LiquidityPool) {
	store := k.getLpSnapshotStore(ctx)
	b := k.cdc.MustMarshal(&pool)
	store.Set(types.PoolKey(pool.Id), b)
}

// GetLiquidityPoolSnapshot retrieves a liquidity pool by id from the store and returns it along with a found flag.
func (k Keeper) GetLiquidityPoolSnapshot(ctx sdk.Context, id string) (val types.LiquidityPool, found bool) {
	store := k.getLpSnapshotStore(ctx)
	b := store.Get(types.PoolKey(id))
	if b == nil {

		return val, false
	}

	k.cdc.MustUnmarshal(b, &val)

	return val, true
}

// GetAllLiquidityPoolSnapshots retrieves all liquidity pool snapshots from the store and returns them as a list.
func (k Keeper) GetAllLiquidityPoolSnapshots(ctx sdk.Context) (list []types.LiquidityPool) {
	store := k.getLpSnapshotStore(ctx)
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.LiquidityPool
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}
