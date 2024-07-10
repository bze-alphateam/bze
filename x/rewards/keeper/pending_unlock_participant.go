package keeper

import (
	"github.com/bze-alphateam/bze/x/rewards/types"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// SetPendingUnlockParticipant set a specific types.PendingUnlockParticipant in the store on its index
func (k Keeper) SetPendingUnlockParticipant(ctx sdk.Context, p types.PendingUnlockParticipant) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.PendingUnlockParticipantKeyPrefix))
	b := k.cdc.MustMarshal(&p)
	store.Set(types.PendingUnlockParticipantKey(p.Index), b)
}

// GetPendingUnlockParticipant returns a types.PendingUnlockParticipant from its index
func (k Keeper) GetPendingUnlockParticipant(ctx sdk.Context, index string) (val types.PendingUnlockParticipant, found bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.PendingUnlockParticipantKeyPrefix))

	b := store.Get(types.PendingUnlockParticipantKey(index))
	if b == nil {
		return val, false
	}

	k.cdc.MustUnmarshal(b, &val)
	return val, true
}

// RemovePendingUnlockParticipant removes a types.PendingUnlockParticipant from the store
func (k Keeper) RemovePendingUnlockParticipant(ctx sdk.Context, p types.PendingUnlockParticipant) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.PendingUnlockParticipantKeyPrefix))
	store.Delete(types.PendingUnlockParticipantKey(p.Index))
}

// GetAllEpochPendingUnlockParticipant returns all types.PendingUnlockParticipant for a certain epoch
func (k Keeper) GetAllEpochPendingUnlockParticipant(ctx sdk.Context, epoch int64) (list []types.PendingUnlockParticipant) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.PendingUnlockParticipantPrefix(epoch)))
	iterator := sdk.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.PendingUnlockParticipant
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}

// GetAllPendingUnlockParticipant returns all types.PendingUnlockParticipant
func (k Keeper) GetAllPendingUnlockParticipant(ctx sdk.Context) (list []types.PendingUnlockParticipant) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.PendingUnlockParticipantKeyPrefix))
	iterator := sdk.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.PendingUnlockParticipant
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}
