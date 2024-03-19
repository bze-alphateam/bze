package keeper

import (
	"github.com/bze-alphateam/bze/x/rewards/types"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// SetStakingReward set a specific stakingReward in the store from its index
func (k Keeper) SetStakingReward(ctx sdk.Context, stakingReward types.StakingReward) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.StakingRewardKeyPrefix))
	b := k.cdc.MustMarshal(&stakingReward)
	store.Set(types.StakingRewardKey(stakingReward.RewardId), b)

	k.incrementStakingRewardsCounter(ctx)
}

// GetStakingReward returns a stakingReward from its index
func (k Keeper) GetStakingReward(ctx sdk.Context, rewardId string) (val types.StakingReward, found bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.StakingRewardKeyPrefix))

	b := store.Get(types.StakingRewardKey(rewardId))
	if b == nil {
		return val, false
	}

	k.cdc.MustUnmarshal(b, &val)
	return val, true
}

// RemoveStakingReward removes a stakingReward from the store
func (k Keeper) RemoveStakingReward(ctx sdk.Context, rewardId string) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.StakingRewardKeyPrefix))
	store.Delete(types.StakingRewardKey(rewardId))
}

// GetAllStakingReward returns all stakingReward
func (k Keeper) GetAllStakingReward(ctx sdk.Context) (list []types.StakingReward) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.StakingRewardKeyPrefix))
	iterator := sdk.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.StakingReward
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}

func (k Keeper) IterateAllStakingRewards(ctx sdk.Context, msgHandler func(ctx sdk.Context, sr types.StakingReward) (stop bool)) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.StakingRewardKeyPrefix))
	iterator := sdk.KVStorePrefixIterator(store, []byte{})
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
