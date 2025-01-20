package keeper

import sdk "github.com/cosmos/cosmos-sdk/types"

const (
	raffleEpochIdentifier = "hour"

	periodicBurnEpochIdentifier = "week"
)

func (k Keeper) GetRaffleCurrentEpoch(ctx sdk.Context) uint64 {
	return uint64(k.epochKeeper.GetEpochCountByIdentifier(ctx, raffleEpochIdentifier))
}
