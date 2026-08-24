package keeper

import (
	"github.com/bze-alphateam/bze/x/rewards/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// GetDenomRewardsDistributionHook returns the day-epoch hook driving the daily denom reward
// schedule payouts. It mirrors GetDistributeAllStakingRewardsHook: the hook only enqueues — the
// payouts themselves run cursor-batched in EndBlock (ProcessDenomRewardsDistributionQueue), so
// the day tick's work stays bounded no matter how many schedules exist.
func (k Keeper) GetDenomRewardsDistributionHook() types.EpochHook {
	hookName := "denom_rewards_distribution_hook"
	return types.NewAfterEpochHook(hookName, func(ctx sdk.Context, epochIdentifier string, epochNumber int64) error {
		if epochIdentifier != distributionEpoch {
			return nil
		}

		k.Logger().
			With("epoch", epochIdentifier, "epoch_number", epochNumber, "hook_name", hookName).
			Debug("preparing to execute hook")

		k.EnqueueDenomRewardsDistribution(ctx)

		return nil
	})
}
