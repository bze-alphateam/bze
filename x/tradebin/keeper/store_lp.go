package keeper

import (
	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	"github.com/bze-alphateam/bze/x/tradebin/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k Keeper) getLpStore(ctx sdk.Context) prefix.Store {
	return k.getPrefixedStore(ctx, types.LpPrefix())
}

func (k Keeper) getLpModificationQueueStore(ctx sdk.Context) prefix.Store {
	return k.getPrefixedStore(ctx, types.KeyPrefix(types.LpModificationQueuePrefix))
}

func (k Keeper) SetLiquidityPool(ctx sdk.Context, pool types.LiquidityPool) {
	store := k.getLpStore(ctx)
	b := k.cdc.MustMarshal(&pool)
	store.Set(types.PoolKey(pool.Id), b)

	// Add pool ID to modification queue
	k.AddLpToModificationQueue(ctx, pool.Id)
}

// AddLpToModificationQueue adds a liquidity pool ID to the modification queue
func (k Keeper) AddLpToModificationQueue(ctx sdk.Context, poolId string) {
	store := k.getLpModificationQueueStore(ctx)
	// Store with empty value - we only need the key to exist
	store.Set(types.LpModificationQueueKey(poolId), []byte{})
}

// GetModifiedLpIds returns all liquidity pool IDs that have been modified
func (k Keeper) GetModifiedLpIds(ctx sdk.Context) []string {
	store := k.getLpModificationQueueStore(ctx)
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})
	defer iterator.Close()

	var poolIds []string
	for ; iterator.Valid(); iterator.Next() {
		// The key is the pool ID
		poolIds = append(poolIds, string(iterator.Key()))
	}

	return poolIds
}

// ClearLpModificationQueue removes all entries from the modification queue
func (k Keeper) ClearLpModificationQueue(ctx sdk.Context, poolIds []string) {
	store := k.getLpModificationQueueStore(ctx)
	// Delete all keys
	for _, lpId := range poolIds {
		store.Delete(types.LpModificationQueueKey(lpId))
	}
}

// GetLiquidityPool returns a LiquidityPool from its id
func (k Keeper) GetLiquidityPool(ctx sdk.Context, id string) (val types.LiquidityPool, found bool) {
	store := k.getLpStore(ctx)
	b := store.Get(types.PoolKey(id))
	if b == nil {

		return val, false
	}

	k.cdc.MustUnmarshal(b, &val)

	return val, true
}

// GetAllLiquidityPool returns all LiquidityPools
func (k Keeper) GetAllLiquidityPool(ctx sdk.Context) (list []types.LiquidityPool) {
	store := k.getLpStore(ctx)
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.LiquidityPool
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}
