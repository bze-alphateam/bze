package keeper

import sdk "github.com/cosmos/cosmos-sdk/types"

func (k Keeper) SnapshotModifiedLiquidityPools(ctx sdk.Context) {
	modified := k.GetModifiedLpIds(ctx)
	for _, id := range modified {
		lp, found := k.GetLiquidityPool(ctx, id)
		if !found {
			k.Logger().Error("could not snapshot pool: id existed in modified queue but the pool was not found", "id", id)
			continue
		}
		k.SetLiquidityPoolSnapshot(ctx, lp)
	}

	k.ClearLpModificationQueue(ctx, modified)
}
