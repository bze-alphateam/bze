package keeper

import (
	storetypes "cosmossdk.io/store/types"
	"github.com/bze-alphateam/bze/x/rewards/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// SetDenomRewardParticipantIndex sets a participant's settlement index for a prize denom.
func (k Keeper) SetDenomRewardParticipantIndex(ctx sdk.Context, index types.DenomRewardParticipantIndex) {
	store := k.getPrefixedStore(ctx, types.KeyPrefix(types.DenomRewardParticipantIndexKeyPrefix))
	b := k.cdc.MustMarshal(&index)
	store.Set(types.DenomRewardParticipantIndexKey(index.Address, index.StakingDenom, index.PrizeDenom), b)
}

// GetDenomRewardParticipantIndex returns a participant's settlement index for a prize denom.
func (k Keeper) GetDenomRewardParticipantIndex(ctx sdk.Context, address, stakingDenom, prizeDenom string) (val types.DenomRewardParticipantIndex, found bool) {
	store := k.getPrefixedStore(ctx, types.KeyPrefix(types.DenomRewardParticipantIndexKeyPrefix))

	b := store.Get(types.DenomRewardParticipantIndexKey(address, stakingDenom, prizeDenom))
	if b == nil {
		return val, false
	}

	k.cdc.MustUnmarshal(b, &val)
	return val, true
}

// RemoveDenomRewardParticipantIndex removes a single participant index.
func (k Keeper) RemoveDenomRewardParticipantIndex(ctx sdk.Context, address, stakingDenom, prizeDenom string) {
	store := k.getPrefixedStore(ctx, types.KeyPrefix(types.DenomRewardParticipantIndexKeyPrefix))
	store.Delete(types.DenomRewardParticipantIndexKey(address, stakingDenom, prizeDenom))
}

// IterateParticipantIndexes iterates every prize-denom index of a single (address, staking denom)
// pair — i.e. one participant's indexes within one DR.
func (k Keeper) IterateParticipantIndexes(ctx sdk.Context, address, stakingDenom string, msgHandler func(ctx sdk.Context, index types.DenomRewardParticipantIndex) (stop bool)) {
	store := k.getPrefixedStore(ctx, types.DenomRewardParticipantIndexPrefix(address, stakingDenom))
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var index types.DenomRewardParticipantIndex
		k.cdc.MustUnmarshal(iterator.Value(), &index)
		if msgHandler(ctx, index) {
			break
		}
	}
}

// RemoveAllParticipantIndexes removes exactly the index slice of one (address, staking denom)
// pair. Keys are collected first and deleted after the iterator is closed (audited SR pattern:
// never delete while iterating).
func (k Keeper) RemoveAllParticipantIndexes(ctx sdk.Context, address, stakingDenom string) {
	store := k.getPrefixedStore(ctx, types.DenomRewardParticipantIndexPrefix(address, stakingDenom))
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})

	var keys [][]byte
	for ; iterator.Valid(); iterator.Next() {
		key := iterator.Key()
		keys = append(keys, append([]byte(nil), key...))
	}
	iterator.Close()

	for _, key := range keys {
		store.Delete(key)
	}
}

// GetAllDenomRewardParticipantIndex returns all participant index records (genesis export).
func (k Keeper) GetAllDenomRewardParticipantIndex(ctx sdk.Context) (list []types.DenomRewardParticipantIndex) {
	store := k.getPrefixedStore(ctx, types.KeyPrefix(types.DenomRewardParticipantIndexKeyPrefix))
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.DenomRewardParticipantIndex
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}
