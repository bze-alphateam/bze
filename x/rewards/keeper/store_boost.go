package keeper

import (
	storetypes "cosmossdk.io/store/types"
	"github.com/bze-alphateam/bze/x/rewards/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// SetBoost set a specific boost in the store from its index
func (k Keeper) SetBoost(ctx sdk.Context, boost types.Boost) {
	store := k.getPrefixedStore(ctx, types.KeyPrefix(types.BoostKeyPrefix))
	b := k.cdc.MustMarshal(&boost)
	store.Set(types.BoostKey(boost.RewardId, boost.Id), b)
}

// GetBoost returns a boost from its index
func (k Keeper) GetBoost(ctx sdk.Context, rewardId, boostId string) (val types.Boost, found bool) {
	store := k.getPrefixedStore(ctx, types.KeyPrefix(types.BoostKeyPrefix))

	b := store.Get(types.BoostKey(rewardId, boostId))
	if b == nil {
		return val, false
	}

	k.cdc.MustUnmarshal(b, &val)
	return val, true
}

// RemoveBoost removes a boost from the store
func (k Keeper) RemoveBoost(ctx sdk.Context, rewardId, boostId string) {
	store := k.getPrefixedStore(ctx, types.KeyPrefix(types.BoostKeyPrefix))
	store.Delete(types.BoostKey(rewardId, boostId))
}

// GetRewardBoosts returns all boosts attached to a reward (bounded by the
// max_boosts_per_reward cap)
func (k Keeper) GetRewardBoosts(ctx sdk.Context, rewardId string) (list []types.Boost) {
	store := k.getPrefixedStore(ctx, types.KeyPrefix(types.BoostKeyPrefix))
	iterator := storetypes.KVStorePrefixIterator(store, types.BoostByRewardPrefix(rewardId))

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.Boost
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}

// GetAllBoosts returns all boosts
func (k Keeper) GetAllBoosts(ctx sdk.Context) (list []types.Boost) {
	store := k.getPrefixedStore(ctx, types.KeyPrefix(types.BoostKeyPrefix))
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.Boost
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}

// ReserveBoostId increments the boost counter and returns the reserved id in
// its fixed-width store form. Ids are globally unique and never reused.
func (k Keeper) ReserveBoostId(ctx sdk.Context) string {
	return k.smallZeroFillId(k.incrementBoostsCounter(ctx))
}

func (k Keeper) GetBoostsCounter(ctx sdk.Context) uint64 {
	return k.GetCounter(ctx, types.BoostCounterKey())
}

func (k Keeper) SetBoostsCounter(ctx sdk.Context, counter uint64) {
	k.SetCounter(ctx, types.BoostCounterKey(), counter)
}

func (k Keeper) incrementBoostsCounter(ctx sdk.Context) uint64 {
	return k.incrementCounter(ctx, types.BoostCounterKey())
}
