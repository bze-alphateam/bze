package keeper

import (
	"encoding/binary"

	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	"github.com/bze-alphateam/bze/x/rewards/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// SetDenomRewardSchedule sets a DenomRewardSchedule in the store.
func (k Keeper) SetDenomRewardSchedule(ctx sdk.Context, schedule types.DenomRewardSchedule) {
	store := k.getPrefixedStore(ctx, types.KeyPrefix(types.DenomRewardScheduleKeyPrefix))
	b := k.cdc.MustMarshal(&schedule)
	store.Set(types.DenomRewardScheduleKey(schedule.StakingDenom, schedule.ScheduleId), b)
}

// GetDenomRewardSchedule returns a DenomRewardSchedule from its (staking denom, schedule id).
func (k Keeper) GetDenomRewardSchedule(ctx sdk.Context, stakingDenom, scheduleId string) (val types.DenomRewardSchedule, found bool) {
	store := k.getPrefixedStore(ctx, types.KeyPrefix(types.DenomRewardScheduleKeyPrefix))

	b := store.Get(types.DenomRewardScheduleKey(stakingDenom, scheduleId))
	if b == nil {
		return val, false
	}

	k.cdc.MustUnmarshal(b, &val)
	return val, true
}

// RemoveDenomRewardSchedule removes a DenomRewardSchedule from the store.
func (k Keeper) RemoveDenomRewardSchedule(ctx sdk.Context, stakingDenom, scheduleId string) {
	store := k.getPrefixedStore(ctx, types.KeyPrefix(types.DenomRewardScheduleKeyPrefix))
	store.Delete(types.DenomRewardScheduleKey(stakingDenom, scheduleId))
}

// IterateDenomSchedules iterates every schedule of a staking denom.
func (k Keeper) IterateDenomSchedules(ctx sdk.Context, stakingDenom string, msgHandler func(ctx sdk.Context, schedule types.DenomRewardSchedule) (stop bool)) {
	store := k.getPrefixedStore(ctx, types.DenomRewardSchedulePrefix(stakingDenom))
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var schedule types.DenomRewardSchedule
		k.cdc.MustUnmarshal(iterator.Value(), &schedule)
		if msgHandler(ctx, schedule) {
			break
		}
	}
}

// GetBatchDenomRewardSchedules returns up to limit DenomRewardSchedule entries across every
// staking denom, in composite-key order, starting strictly after the given cursor. An empty
// cursor starts from the beginning. Mirrors GetBatchStakingRewards; the cursor is the store key
// (suffix) of the last processed schedule, i.e. string(DenomRewardScheduleKey(denom, id)).
func (k Keeper) GetBatchDenomRewardSchedules(ctx sdk.Context, cursor string, limit int) []types.DenomRewardSchedule {
	store := k.getPrefixedStore(ctx, types.KeyPrefix(types.DenomRewardScheduleKeyPrefix))

	var startKey []byte
	if cursor != "" {
		// start right after the cursor key by appending 0x00
		startKey = append([]byte(cursor), 0x00)
	}

	iterator := store.Iterator(startKey, nil)
	defer iterator.Close()

	var list []types.DenomRewardSchedule
	for ; iterator.Valid() && len(list) < limit; iterator.Next() {
		var schedule types.DenomRewardSchedule
		k.cdc.MustUnmarshal(iterator.Value(), &schedule)
		list = append(list, schedule)
	}

	return list
}

// GetAllDenomRewardSchedule returns every DenomRewardSchedule (genesis export).
func (k Keeper) GetAllDenomRewardSchedule(ctx sdk.Context) (list []types.DenomRewardSchedule) {
	store := k.getPrefixedStore(ctx, types.KeyPrefix(types.DenomRewardScheduleKeyPrefix))
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.DenomRewardSchedule
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}

// getDenomRewardCounterStore returns the DR-private counter store. It is separate from the
// SR/trading counter store (getCounterStore) so the DR schedule counter never collides with them.
func (k Keeper) getDenomRewardCounterStore(ctx sdk.Context) prefix.Store {
	return k.getPrefixedStore(ctx, types.KeyPrefix(types.DenomRewardCounterKey))
}

// GetDenomRewardScheduleCounter returns the current DR schedule counter. Mirrors store_counter.go.
func (k Keeper) GetDenomRewardScheduleCounter(ctx sdk.Context) uint64 {
	store := k.getDenomRewardCounterStore(ctx)
	counter := store.Get(types.DenomRewardScheduleCounterKey())
	if counter == nil {
		return 0
	}

	return binary.BigEndian.Uint64(counter)
}

// SetDenomRewardScheduleCounter stores the DR schedule counter. Mirrors store_counter.go.
func (k Keeper) SetDenomRewardScheduleCounter(ctx sdk.Context, counter uint64) {
	store := k.getDenomRewardCounterStore(ctx)
	record := make([]byte, 8)
	binary.BigEndian.PutUint64(record, counter)

	store.Set(types.DenomRewardScheduleCounterKey(), record)
}

// incrementDenomRewardScheduleCounter increments and returns the DR schedule counter.
func (k Keeper) incrementDenomRewardScheduleCounter(ctx sdk.Context) uint64 {
	counter := k.GetDenomRewardScheduleCounter(ctx)
	counter++
	k.SetDenomRewardScheduleCounter(ctx, counter)

	return counter
}

// SetDenomRewardsDistributionQueue stores the singleton DR distribution queue.
func (k Keeper) SetDenomRewardsDistributionQueue(ctx sdk.Context, q types.DenomRewardsDistributionQueue) {
	store := k.getPrefixedStore(ctx, types.KeyPrefix(types.DenomRewardsDistributionQueueKey))
	b := k.cdc.MustMarshal(&q)
	store.Set([]byte{1}, b)
}

// GetDenomRewardsDistributionQueue returns the singleton DR distribution queue.
func (k Keeper) GetDenomRewardsDistributionQueue(ctx sdk.Context) (val types.DenomRewardsDistributionQueue, found bool) {
	store := k.getPrefixedStore(ctx, types.KeyPrefix(types.DenomRewardsDistributionQueueKey))

	b := store.Get([]byte{1})
	if b == nil {
		return val, false
	}

	k.cdc.MustUnmarshal(b, &val)
	return val, true
}

// RemoveDenomRewardsDistributionQueue removes the singleton DR distribution queue.
func (k Keeper) RemoveDenomRewardsDistributionQueue(ctx sdk.Context) {
	store := k.getPrefixedStore(ctx, types.KeyPrefix(types.DenomRewardsDistributionQueueKey))
	store.Delete([]byte{1})
}
