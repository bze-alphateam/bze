package keeper

import (
	"strings"

	storetypes "cosmossdk.io/store/types"
	"github.com/bze-alphateam/bze/x/rewards/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// SetDenomRewardParticipant sets a DenomRewardParticipant and, in the same call, writes the
// address-first marker used by the user-positions query. The two writes must stay in sync.
func (k Keeper) SetDenomRewardParticipant(ctx sdk.Context, participant types.DenomRewardParticipant) {
	store := k.getPrefixedStore(ctx, types.KeyPrefix(types.DenomRewardParticipantKeyPrefix))
	b := k.cdc.MustMarshal(&participant)
	store.Set(types.DenomRewardParticipantKey(participant.StakingDenom, participant.Address), b)

	markerStore := k.getPrefixedStore(ctx, types.KeyPrefix(types.DenomRewardParticipantMarkerKeyPrefix))
	markerStore.Set(types.DenomRewardParticipantMarkerKey(participant.Address, participant.StakingDenom), []byte{})
}

// GetDenomRewardParticipant returns a DenomRewardParticipant from its (staking denom, address).
func (k Keeper) GetDenomRewardParticipant(ctx sdk.Context, stakingDenom, address string) (val types.DenomRewardParticipant, found bool) {
	store := k.getPrefixedStore(ctx, types.KeyPrefix(types.DenomRewardParticipantKeyPrefix))

	b := store.Get(types.DenomRewardParticipantKey(stakingDenom, address))
	if b == nil {
		return val, false
	}

	k.cdc.MustUnmarshal(b, &val)
	return val, true
}

// RemoveDenomRewardParticipant removes a DenomRewardParticipant and its address-first marker
// in the same call, keeping the two stores in sync.
func (k Keeper) RemoveDenomRewardParticipant(ctx sdk.Context, stakingDenom, address string) {
	store := k.getPrefixedStore(ctx, types.KeyPrefix(types.DenomRewardParticipantKeyPrefix))
	store.Delete(types.DenomRewardParticipantKey(stakingDenom, address))

	markerStore := k.getPrefixedStore(ctx, types.KeyPrefix(types.DenomRewardParticipantMarkerKeyPrefix))
	markerStore.Delete(types.DenomRewardParticipantMarkerKey(address, stakingDenom))
}

// IterateDenomRewardParticipants iterates every participant of a staking denom.
func (k Keeper) IterateDenomRewardParticipants(ctx sdk.Context, stakingDenom string, msgHandler func(ctx sdk.Context, participant types.DenomRewardParticipant) (stop bool)) {
	store := k.getPrefixedStore(ctx, types.DenomRewardParticipantPrefix(stakingDenom))
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var participant types.DenomRewardParticipant
		k.cdc.MustUnmarshal(iterator.Value(), &participant)
		if msgHandler(ctx, participant) {
			break
		}
	}
}

// IterateUserDenomRewards iterates every participation of an address, resolving each denom
// through the address-first marker index.
func (k Keeper) IterateUserDenomRewards(ctx sdk.Context, address string, msgHandler func(ctx sdk.Context, participant types.DenomRewardParticipant) (stop bool)) {
	markerStore := k.getPrefixedStore(ctx, types.DenomRewardParticipantMarkerPrefix(address))
	iterator := storetypes.KVStorePrefixIterator(markerStore, []byte{})
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		// the marker key (prefix stripped) is the staking denom followed by the separator,
		// a byte that can never appear inside a denom (unlike "/", legal in ibc/factory denoms)
		stakingDenom := strings.TrimSuffix(string(iterator.Key()), types.DenomRewardKeySeparator)
		participant, found := k.GetDenomRewardParticipant(ctx, stakingDenom, address)
		if !found {
			continue
		}
		if msgHandler(ctx, participant) {
			break
		}
	}
}

// GetAllDenomRewardParticipant returns all DenomRewardParticipant records across every staking
// denom (genesis export).
func (k Keeper) GetAllDenomRewardParticipant(ctx sdk.Context) (list []types.DenomRewardParticipant) {
	store := k.getPrefixedStore(ctx, types.KeyPrefix(types.DenomRewardParticipantKeyPrefix))
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.DenomRewardParticipant
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}
