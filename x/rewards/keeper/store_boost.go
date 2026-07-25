package keeper

import (
	storetypes "cosmossdk.io/store/types"
	"github.com/bze-alphateam/bze/x/rewards/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// SetBoost sets a specific boost in the store from its (reward_id, denom) index.
func (k Keeper) SetBoost(ctx sdk.Context, boost types.Boost) {
	store := k.getPrefixedStore(ctx, types.KeyPrefix(types.BoostKeyPrefix))
	b := k.cdc.MustMarshal(&boost)
	store.Set(types.BoostKey(boost.RewardId, boost.Denom), b)
}

// GetBoost returns a boost from its (reward_id, denom) index.
func (k Keeper) GetBoost(ctx sdk.Context, rewardId, denom string) (val types.Boost, found bool) {
	store := k.getPrefixedStore(ctx, types.KeyPrefix(types.BoostKeyPrefix))

	b := store.Get(types.BoostKey(rewardId, denom))
	if b == nil {
		return val, false
	}

	k.cdc.MustUnmarshal(b, &val)
	return val, true
}

// RemoveBoost removes a boost from the store.
func (k Keeper) RemoveBoost(ctx sdk.Context, rewardId, denom string) {
	store := k.getPrefixedStore(ctx, types.KeyPrefix(types.BoostKeyPrefix))
	store.Delete(types.BoostKey(rewardId, denom))
}

// GetRewardBoosts returns all boosts of a single reward via a bounded prefix scan.
func (k Keeper) GetRewardBoosts(ctx sdk.Context, rewardId string) (list []types.Boost) {
	store := k.getPrefixedStore(ctx, types.KeyPrefix(types.BoostKeyPrefix))
	iterator := storetypes.KVStorePrefixIterator(store, types.BoostRewardPrefix(rewardId))

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.Boost
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}

// GetAllBoosts returns every boost in the store.
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

// GetBoostCounter returns the current value of the global boost uid counter.
func (k Keeper) GetBoostCounter(ctx sdk.Context) uint64 {
	return k.GetCounter(ctx, types.BoostCounterKey())
}

// SetBoostCounter sets the global boost uid counter.
func (k Keeper) SetBoostCounter(ctx sdk.Context, counter uint64) {
	k.SetCounter(ctx, types.BoostCounterKey(), counter)
}

// ReserveBoostUid increments the global boost uid counter and returns the new,
// unique uid to assign to a freshly created boost run.
func (k Keeper) ReserveBoostUid(ctx sdk.Context) uint64 {
	return k.incrementCounter(ctx, types.BoostCounterKey())
}
