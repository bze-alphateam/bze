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

func (k Keeper) SetLiquidityPool(ctx sdk.Context, pool types.LiquidityPool) {
	store := k.getLpStore(ctx)
	b := k.cdc.MustMarshal(&pool)
	store.Set(types.PoolKey(pool.Id), b)
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
