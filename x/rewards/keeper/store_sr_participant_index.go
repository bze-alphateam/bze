package keeper

import (
	"strings"

	"github.com/bze-alphateam/bze/x/rewards/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// SetStakingRewardParticipantIndexEntry marks the address as a participant of
// the reward in the reverse index. Set semantics: writing an existing entry
// overwrites it in place, so top-up joins are idempotent.
func (k Keeper) SetStakingRewardParticipantIndexEntry(ctx sdk.Context, rewardId, address string) {
	store := k.getPrefixedStore(ctx, types.KeyPrefix(types.StakingRewardParticipantIndexKeyPrefix))
	store.Set(types.StakingRewardParticipantIndexKey(rewardId, address), types.StakingRewardParticipantIndexValue)
}

// RemoveStakingRewardParticipantIndexEntry removes the address' index entry
// for the reward
func (k Keeper) RemoveStakingRewardParticipantIndexEntry(ctx sdk.Context, rewardId, address string) {
	store := k.getPrefixedStore(ctx, types.KeyPrefix(types.StakingRewardParticipantIndexKeyPrefix))
	store.Delete(types.StakingRewardParticipantIndexKey(rewardId, address))
}

// HasStakingRewardParticipantIndexEntry reports whether the address is in the
// reward's index
func (k Keeper) HasStakingRewardParticipantIndexEntry(ctx sdk.Context, rewardId, address string) bool {
	store := k.getPrefixedStore(ctx, types.KeyPrefix(types.StakingRewardParticipantIndexKeyPrefix))
	return store.Has(types.StakingRewardParticipantIndexKey(rewardId, address))
}

// GetStakingRewardParticipantIndexAddresses returns up to limit participant
// addresses of one reward, in ascending key order, starting after afterAddress
// ("" = from the beginning; limit 0 = nothing). The exclusive-start cursor and
// the bound let the cleanup sweep resume from a persisted cursor with
// predictable work per call.
func (k Keeper) GetStakingRewardParticipantIndexAddresses(ctx sdk.Context, rewardId, afterAddress string, limit uint32) (list []string) {
	store := k.getPrefixedStore(ctx, types.StakingRewardParticipantIndexRewardPrefix(rewardId))

	iterator := store.Iterator(participantIndexExclusiveStart(afterAddress), nil)
	defer iterator.Close()

	for ; iterator.Valid() && uint32(len(list)) < limit; iterator.Next() {
		list = append(list, strings.TrimSuffix(string(iterator.Key()), "/"))
	}

	return
}

// CountStakingRewardParticipantIndexEntries returns how many of the reward's
// index entries sort strictly after afterAddress ("" = all of them) — the
// cleanup sweep's remaining work, computed node-side for the status query.
func (k Keeper) CountStakingRewardParticipantIndexEntries(ctx sdk.Context, rewardId, afterAddress string) (count uint64) {
	store := k.getPrefixedStore(ctx, types.StakingRewardParticipantIndexRewardPrefix(rewardId))

	iterator := store.Iterator(participantIndexExclusiveStart(afterAddress), nil)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		count++
	}

	return
}

func participantIndexExclusiveStart(afterAddress string) []byte {
	if afterAddress == "" {
		return nil
	}

	// index keys end in "/": a zero byte appended to the cursor's own key
	// is the smallest key sorting after it, making the start exclusive
	return append([]byte(afterAddress+"/"), 0)
}
