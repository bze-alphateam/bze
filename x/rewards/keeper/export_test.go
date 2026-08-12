package keeper

import (
	"cosmossdk.io/math"
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

// SettleDenomParticipant exposes the unexported denom-reward settle engine for tests (no message
// handlers exist yet — they arrive in later cascade stories — so the service is driven directly).
func (k Keeper) SettleDenomParticipant(ctx sdk.Context, dr types.DenomReward, participant types.DenomRewardParticipant) (sdk.Coins, error) {
	return k.settleDenomParticipant(ctx, dr, participant)
}

// StampParticipantIndexes exposes the unexported fresh-join index stamping for tests.
func (k Keeper) StampParticipantIndexes(ctx sdk.Context, address, stakingDenom string) {
	k.stampParticipantIndexes(ctx, address, stakingDenom)
}

// DistributeToDenomPrize exposes the unexported accumulator-bump primitive for tests.
func (k Keeper) DistributeToDenomPrize(ctx sdk.Context, prize types.DenomRewardPrize, amount, stakedTotal math.Int) error {
	return k.distributeToDenomPrize(ctx, prize, amount, stakedTotal)
}
