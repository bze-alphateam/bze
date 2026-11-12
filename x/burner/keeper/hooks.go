package keeper

import (
	"github.com/bze-alphateam/bze/x/burner/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	hookName             = "burner_raffle_cleanup"
	periodicBurnHookName = "periodic_burner"
)

// GetBurnerPeriodicBurnHook marks a periodic burn as pending at the end of every PeriodicBurningWeeks-th week. It only
// writes the queue flag and moves no coin, so it also runs while the protocol automation is paused: the burn itself
// (ProcessPeriodicBurnQueue) waits, and an epoch that ended during the pause is burned right after the resume instead
// of at the next burning epoch.
func (k Keeper) GetBurnerPeriodicBurnHook() types.EpochHook {
	return types.NewAfterEpochHook(periodicBurnHookName, func(ctx sdk.Context, epochIdentifier string, epochNumber int64) error {
		if epochIdentifier != periodicBurnEpochIdentifier {
			return nil
		}

		params := k.GetParams(ctx)
		if epochNumber%params.PeriodicBurningWeeks != 0 {
			return nil
		}

		k.Logger().
			With("epoch", epochIdentifier, "epoch_number", epochNumber, "hook_name", periodicBurnHookName).
			Debug("preparing to execute hook")

		k.EnqueuePeriodicBurn(ctx)

		return nil
	})
}

func (k Keeper) GetBurnerRaffleCleanupHook() types.EpochHook {
	return types.NewAfterEpochHook(hookName, func(ctx sdk.Context, epochIdentifier string, epochNumber int64) error {
		if epochIdentifier != raffleEpochIdentifier {
			return nil
		}

		k.Logger().
			With("epoch", epochIdentifier, "epoch_number", epochNumber, "hook_name", hookName).
			Debug("preparing to execute hook")

		k.EnqueueRaffleCleanup(ctx, uint64(epochNumber))

		return nil
	})
}
