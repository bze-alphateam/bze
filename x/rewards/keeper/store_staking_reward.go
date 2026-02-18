package keeper

import (
	storetypes "cosmossdk.io/store/types"
	"github.com/bze-alphateam/bze/x/rewards/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// SetStakingReward set a specific stakingReward in the store from its index
func (k Keeper) SetStakingReward(ctx sdk.Context, stakingReward types.StakingReward) {
	store := k.getPrefixedStore(ctx, types.KeyPrefix(types.StakingRewardKeyPrefix))
	b := k.cdc.MustMarshal(&stakingReward)
	store.Set(types.StakingRewardKey(stakingReward.RewardId), b)
}

// GetStakingReward returns a stakingReward from its index
func (k Keeper) GetStakingReward(ctx sdk.Context, rewardId string) (val types.StakingReward, found bool) {
	store := k.getPrefixedStore(ctx, types.KeyPrefix(types.StakingRewardKeyPrefix))

	b := store.Get(types.StakingRewardKey(rewardId))
	if b == nil {
		return val, false
	}

	k.cdc.MustUnmarshal(b, &val)
	return val, true
}

// RemoveStakingReward removes a stakingReward from the store
func (k Keeper) RemoveStakingReward(ctx sdk.Context, rewardId string) {
	store := k.getPrefixedStore(ctx, types.KeyPrefix(types.StakingRewardKeyPrefix))
	store.Delete(types.StakingRewardKey(rewardId))
}

// GetAllStakingReward returns all stakingReward
func (k Keeper) GetAllStakingReward(ctx sdk.Context) (list []types.StakingReward) {
	store := k.getPrefixedStore(ctx, types.KeyPrefix(types.StakingRewardKeyPrefix))
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.StakingReward
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}

func (k Keeper) IterateAllStakingRewards(ctx sdk.Context, msgHandler func(ctx sdk.Context, sr types.StakingReward) (stop bool)) {
	store := k.getPrefixedStore(ctx, types.KeyPrefix(types.StakingRewardKeyPrefix))
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var sr types.StakingReward
		k.cdc.MustUnmarshal(iterator.Value(), &sr)
		s := msgHandler(ctx, sr)
		if s {
			break
		}
	}
}

// GetBatchStakingRewards returns up to limit StakingReward entries starting after the given cursor.
// If cursor is empty, it starts from the beginning.
func (k Keeper) GetBatchStakingRewards(ctx sdk.Context, startAtRewardId string, limit int) []types.StakingReward {
	store := k.getPrefixedStore(ctx, types.KeyPrefix(types.StakingRewardKeyPrefix))

	var startKey []byte
	if startAtRewardId != "" {
		// Start right after the startAtRewardId key by appending 0x00
		startKey = append(types.StakingRewardKey(startAtRewardId), 0x00)
	}

	iterator := store.Iterator(startKey, nil)
	defer iterator.Close()

	var list []types.StakingReward
	for ; iterator.Valid() && len(list) < limit; iterator.Next() {
		var sr types.StakingReward
		k.cdc.MustUnmarshal(iterator.Value(), &sr)
		list = append(list, sr)
	}

	return list
}

func (k Keeper) SetStakingRewardsDistributionQueue(ctx sdk.Context, q types.StakingRewardsDistributionQueue) {
	store := k.getPrefixedStore(ctx, types.KeyPrefix(types.StakingRewardDistributionQueueKey))
	b := k.cdc.MustMarshal(&q)
	store.Set([]byte{1}, b)
}

func (k Keeper) GetStakingRewardsDistributionQueue(ctx sdk.Context) (val types.StakingRewardsDistributionQueue, found bool) {
	store := k.getPrefixedStore(ctx, types.KeyPrefix(types.StakingRewardDistributionQueueKey))

	b := store.Get([]byte{1})
	if b == nil {
		return val, false
	}

	k.cdc.MustUnmarshal(b, &val)
	return val, true
}

func (k Keeper) RemoveStakingRewardsDistributionQueue(ctx sdk.Context) {
	store := k.getPrefixedStore(ctx, types.KeyPrefix(types.StakingRewardDistributionQueueKey))
	store.Delete([]byte{1})
}
