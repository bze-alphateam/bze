package keeper

import (
	"github.com/bze-alphateam/bze/x/rewards/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// HasDenomRewardParticipantMarker reports whether the address-first participant marker
// (drp/a/) exists. Test-only accessor to the unexported marker store, used to verify that
// SetDenomRewardParticipant/RemoveDenomRewardParticipant keep the marker in sync.
func (k Keeper) HasDenomRewardParticipantMarker(ctx sdk.Context, address, stakingDenom string) bool {
	store := k.getPrefixedStore(ctx, types.KeyPrefix(types.DenomRewardParticipantMarkerKeyPrefix))
	return store.Has(types.DenomRewardParticipantMarkerKey(address, stakingDenom))
}

// IncrementDenomRewardScheduleCounter exposes the unexported counter-advance path for tests.
func (k Keeper) IncrementDenomRewardScheduleCounter(ctx sdk.Context) uint64 {
	return k.incrementDenomRewardScheduleCounter(ctx)
}
