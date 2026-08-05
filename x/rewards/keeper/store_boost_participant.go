package keeper

import (
	storetypes "cosmossdk.io/store/types"
	"github.com/bze-alphateam/bze/x/rewards/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// SetBoostParticipant set a specific boostParticipant in the store from its index
func (k Keeper) SetBoostParticipant(ctx sdk.Context, boostParticipant types.BoostParticipant) {
	store := k.getPrefixedStore(ctx, types.KeyPrefix(types.BoostParticipantKeyPrefix))
	b := k.cdc.MustMarshal(&boostParticipant)
	store.Set(types.BoostParticipantKey(boostParticipant.Address, boostParticipant.RewardId, boostParticipant.BoostId), b)
}

// GetBoostParticipant returns a boostParticipant from its index. An absent
// record means the address staked since before the boost was created (S0 = 0).
func (k Keeper) GetBoostParticipant(ctx sdk.Context, address, rewardId, boostId string) (val types.BoostParticipant, found bool) {
	store := k.getPrefixedStore(ctx, types.KeyPrefix(types.BoostParticipantKeyPrefix))

	b := store.Get(types.BoostParticipantKey(address, rewardId, boostId))
	if b == nil {
		return val, false
	}

	k.cdc.MustUnmarshal(b, &val)
	return val, true
}

// RemoveRewardBoostParticipants removes all of an address' boost participant
// entries for one reward (prefix delete, bounded by the boosts-per-reward cap)
func (k Keeper) RemoveRewardBoostParticipants(ctx sdk.Context, address, rewardId string) {
	store := k.getPrefixedStore(ctx, types.KeyPrefix(types.BoostParticipantKeyPrefix))
	iterator := storetypes.KVStorePrefixIterator(store, types.BoostParticipantByRewardPrefix(address, rewardId))

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		store.Delete(iterator.Key())
	}
}

// removeOrphanedBoostParticipants deletes the address' participant entries
// for one reward that reference boosts which no longer exist (cleaned up).
// Bounded by the boosts-per-reward cap, like the exit prefix delete.
func (k Keeper) removeOrphanedBoostParticipants(ctx sdk.Context, address, rewardId string) {
	store := k.getPrefixedStore(ctx, types.KeyPrefix(types.BoostParticipantKeyPrefix))
	iterator := storetypes.KVStorePrefixIterator(store, types.BoostParticipantByRewardPrefix(address, rewardId))

	// collect first: the store must not be mutated while the iterator is open
	var orphaned [][]byte
	for ; iterator.Valid(); iterator.Next() {
		var val types.BoostParticipant
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		if _, found := k.GetBoost(ctx, val.RewardId, val.BoostId); !found {
			orphaned = append(orphaned, append([]byte(nil), iterator.Key()...))
		}
	}
	iterator.Close()

	for _, key := range orphaned {
		store.Delete(key)
	}
}

// GetAllLiveBoostParticipant returns all boostParticipant entries whose boost
// still exists. Orphaned entries awaiting lazy removal are excluded — genesis
// validation requires every exported entry's boost to be present.
func (k Keeper) GetAllLiveBoostParticipant(ctx sdk.Context) (list []types.BoostParticipant) {
	for _, entry := range k.GetAllBoostParticipant(ctx) {
		if _, found := k.GetBoost(ctx, entry.RewardId, entry.BoostId); found {
			list = append(list, entry)
		}
	}

	return
}

// GetAllBoostParticipant returns all boostParticipant
func (k Keeper) GetAllBoostParticipant(ctx sdk.Context) (list []types.BoostParticipant) {
	store := k.getPrefixedStore(ctx, types.KeyPrefix(types.BoostParticipantKeyPrefix))
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.BoostParticipant
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}
