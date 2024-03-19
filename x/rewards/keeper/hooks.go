package keeper

import (
	"github.com/bze-alphateam/bze/x/rewards/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k Keeper) GetDistributeAllStakingRewardsHook() types.EpochHook {
	hookName := "distribution_hook"
	return types.NewAfterEpochHook(hookName, func(ctx sdk.Context, epochIdentifier string, epochNumber int64) error {
		if epochIdentifier != distributionEpoch {
			return nil
		}

		k.Logger(ctx).
			With("epoch", epochIdentifier, "epoch_number", epochNumber, "hook_name", hookName).
			Debug("preparing to execute hook")

		k.DistributeAllStakingRewards(ctx)

		return nil
	})
}

func (k Keeper) GetUnlockPendingUnlockParticipantsHook() types.EpochHook {
	hookName := "pending_unlock_hook"
	return types.NewAfterEpochHook(hookName, func(ctx sdk.Context, epochIdentifier string, epochNumber int64) error {
		if epochIdentifier != distributionEpoch {
			return nil
		}

		k.Logger(ctx).
			With("epoch", epochIdentifier, "epoch_number", epochNumber, "hook_name", hookName).
			Debug("preparing to execute hook")

		k.UnlockAllPendingUnlockParticipants(ctx, epochNumber)

		return nil
	})
}
