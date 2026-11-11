package keeper

import (
	"cosmossdk.io/store/prefix"
	"github.com/bze-alphateam/bze/x/tradebin/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k Keeper) getPoolPriceAverageStore(ctx sdk.Context) prefix.Store {
	return k.getPrefixedStore(ctx, types.PoolPriceAveragePrefix())
}

// SetPoolPriceAverage stores the price average record of a pool.
func (k Keeper) SetPoolPriceAverage(ctx sdk.Context, record types.PoolPriceAverage) {
	b := k.cdc.MustMarshal(&record)
	k.getPoolPriceAverageStore(ctx).Set(types.PoolPriceAverageKey(record.PoolId), b)
}

// GetPoolPriceAverageRecord returns the stored price average record of a pool, as it was last written.
func (k Keeper) GetPoolPriceAverageRecord(ctx sdk.Context, poolId string) (val types.PoolPriceAverage, found bool) {
	b := k.getPoolPriceAverageStore(ctx).Get(types.PoolPriceAverageKey(poolId))
	if b == nil {
		return val, false
	}

	k.cdc.MustUnmarshal(b, &val)

	return val, true
}
