package keeper

import (
	storetypes "cosmossdk.io/store/types"
	"github.com/bze-alphateam/bze/x/rewards/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// SetStakingRewardParticipant set a specific stakingRewardParticipant in the store from its index
func (k Keeper) SetStakingRewardParticipant(ctx sdk.Context, stakingRewardParticipant types.StakingRewardParticipant) {
	store := k.getPrefixedStore(ctx, types.KeyPrefix(types.StakingRewardParticipantKeyPrefix))
	b := k.cdc.MustMarshal(&stakingRewardParticipant)
	store.Set(types.StakingRewardParticipantKey(stakingRewardParticipant.Address, stakingRewardParticipant.RewardId), b)
}

// GetStakingRewardParticipant returns a stakingRewardParticipant from its index
func (k Keeper) GetStakingRewardParticipant(ctx sdk.Context, address, rewardId string) (val types.StakingRewardParticipant, found bool) {
	store := k.getPrefixedStore(ctx, types.KeyPrefix(types.StakingRewardParticipantKeyPrefix))

	b := store.Get(types.StakingRewardParticipantKey(address, rewardId))
	if b == nil {
		return val, false
	}

	k.cdc.MustUnmarshal(b, &val)
	return val, true
}

// RemoveStakingRewardParticipant removes a stakingRewardParticipant from the store
func (k Keeper) RemoveStakingRewardParticipant(ctx sdk.Context, address, rewardId string) {
	store := k.getPrefixedStore(ctx, types.KeyPrefix(types.StakingRewardParticipantKeyPrefix))
	store.Delete(types.StakingRewardParticipantKey(address, rewardId))
}

// GetAllStakingRewardParticipant returns all stakingRewardParticipant
func (k Keeper) GetAllStakingRewardParticipant(ctx sdk.Context) (list []types.StakingRewardParticipant) {
	store := k.getPrefixedStore(ctx, types.KeyPrefix(types.StakingRewardParticipantKeyPrefix))
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.StakingRewardParticipant
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}
