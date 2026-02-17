package keeper

import (
	storetypes "cosmossdk.io/store/types"
	"github.com/bze-alphateam/bze/x/rewards/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// SetPendingUnlockParticipant set a specific types.PendingUnlockParticipant in the store on its index
func (k Keeper) SetPendingUnlockParticipant(ctx sdk.Context, p types.PendingUnlockParticipant) {
	store := k.getPrefixedStore(ctx, types.KeyPrefix(types.PendingUnlockParticipantKeyPrefix))
	b := k.cdc.MustMarshal(&p)
	store.Set(types.PendingUnlockParticipantKey(p.Index), b)
}

// GetPendingUnlockParticipant returns a types.PendingUnlockParticipant from its index
func (k Keeper) GetPendingUnlockParticipant(ctx sdk.Context, index string) (val types.PendingUnlockParticipant, found bool) {
	store := k.getPrefixedStore(ctx, types.KeyPrefix(types.PendingUnlockParticipantKeyPrefix))

	b := store.Get(types.PendingUnlockParticipantKey(index))
	if b == nil {
		return val, false
	}

	k.cdc.MustUnmarshal(b, &val)
	return val, true
}

// RemovePendingUnlockParticipant removes a types.PendingUnlockParticipant from the store
func (k Keeper) RemovePendingUnlockParticipant(ctx sdk.Context, p types.PendingUnlockParticipant) {
	store := k.getPrefixedStore(ctx, types.KeyPrefix(types.PendingUnlockParticipantKeyPrefix))
	store.Delete(types.PendingUnlockParticipantKey(p.Index))
}

// GetAllEpochPendingUnlockParticipant returns all types.PendingUnlockParticipant for a certain epoch
func (k Keeper) GetAllEpochPendingUnlockParticipant(ctx sdk.Context, epoch int64) (list []types.PendingUnlockParticipant) {
	store := k.getPrefixedStore(ctx, types.KeyPrefix(types.PendingUnlockParticipantPrefix(epoch)))
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.PendingUnlockParticipant
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}

func (k Keeper) IterateAllEpochPendingUnlockParticipant(ctx sdk.Context, epoch int64, msgHandler func(ctx sdk.Context, sr types.PendingUnlockParticipant) (stop bool)) {
	store := k.getPrefixedStore(ctx, types.KeyPrefix(types.PendingUnlockParticipantPrefix(epoch)))
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var sr types.PendingUnlockParticipant
		k.cdc.MustUnmarshal(iterator.Value(), &sr)
		s := msgHandler(ctx, sr)
		if s {
			break
		}
	}
}

// GetBatchEpochPendingUnlockParticipant returns up to limit PendingUnlockParticipant entries for the given epoch
func (k Keeper) GetBatchEpochPendingUnlockParticipant(ctx sdk.Context, epoch int64, limit int) []types.PendingUnlockParticipant {
	store := k.getPrefixedStore(ctx, types.KeyPrefix(types.PendingUnlockParticipantPrefix(epoch)))
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})
	defer iterator.Close()

	var list []types.PendingUnlockParticipant
	for ; iterator.Valid() && len(list) < limit; iterator.Next() {
		var val types.PendingUnlockParticipant
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return list
}

// GetAllPendingUnlockParticipant returns all types.PendingUnlockParticipant
func (k Keeper) GetAllPendingUnlockParticipant(ctx sdk.Context) (list []types.PendingUnlockParticipant) {
	store := k.getPrefixedStore(ctx, types.KeyPrefix(types.PendingUnlockParticipantKeyPrefix))
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.PendingUnlockParticipant
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}

func (k Keeper) SetUnlockParticipantsQueue(ctx sdk.Context, p types.UnlockParticipantsQueue) {
	store := k.getPrefixedStore(ctx, types.KeyPrefix(types.PendingUnlockQueueKey))
	b := k.cdc.MustMarshal(&p)
	store.Set([]byte{1}, b)
}

func (k Keeper) GetUnlockParticipantsQueue(ctx sdk.Context) (val types.UnlockParticipantsQueue, found bool) {
	store := k.getPrefixedStore(ctx, types.KeyPrefix(types.PendingUnlockQueueKey))

	b := store.Get([]byte{1})
	if b == nil {
		return val, false
	}

	k.cdc.MustUnmarshal(b, &val)
	return val, true
}
