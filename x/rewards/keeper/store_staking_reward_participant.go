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

// IterateStakingRewardParticipantsByReward walks every StakingRewardParticipant
// that belongs to `rewardId` and invokes `cb` for each one. Returning true
// from `cb` stops iteration immediately.
//
// Implementation note: the existing participant key layout is
// `<address>/<reward_id>/` (address-major), so there is no efficient index
// to scan participants of a single reward without a full prefix scan. We
// iterate the whole participant store and filter in-process. At BZE's
// scale (small number of staking programs, bounded participants per
// program) the linear cost is acceptable; if it ever stops being acceptable
// the right fix is a secondary `<reward_id>/<address>/` index maintained
// in Set/Remove, plus a migration to backfill existing rows.
//
// This is consumed by x/daodao's REWARD_STAKED snapshot path
// (votingBackend.SnapshotAll). Snapshots happen at most once per proposal
// creation, never per block, so the cost is bounded by user behavior.
func (k Keeper) IterateStakingRewardParticipantsByReward(
	ctx sdk.Context,
	rewardId string,
	cb func(types.StakingRewardParticipant) (stop bool),
) {
	store := k.getPrefixedStore(ctx, types.KeyPrefix(types.StakingRewardParticipantKeyPrefix))
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var p types.StakingRewardParticipant
		k.cdc.MustUnmarshal(iterator.Value(), &p)
		if p.RewardId != rewardId {
			continue
		}
		if cb(p) {
			return
		}
	}
}
