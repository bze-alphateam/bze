package keeper

import sdk "github.com/cosmos/cosmos-sdk/types"

const (
	raffleEpochIdentifier = "hour"

	periodicBurnEpochIdentifier = "week"
)

func (k Keeper) GetRaffleCurrentEpoch(ctx sdk.Context) (uint64, error) {
	no, err := k.epochKeeper.SafeGetEpochCountByIdentifier(ctx, raffleEpochIdentifier)
	if err != nil {
		return 0, err
	}

	return uint64(no), nil
}
