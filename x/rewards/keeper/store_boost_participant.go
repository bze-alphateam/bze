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
