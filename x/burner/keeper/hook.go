package keeper

import (
	"github.com/bze-alphateam/bze/x/burner/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	hookName = "burner_raffle_cleanup"
)

func (k Keeper) GetBurnerRaffleCleanupHook() types.EpochHook {
	return types.NewAfterEpochHook(hookName, func(ctx sdk.Context, epochIdentifier string, epochNumber int64) error {
		if epochIdentifier != raffleEpochIdentifier {
			return nil
		}

		k.Logger(ctx).
			With("epoch", epochIdentifier, "epoch_number", epochNumber, "hook_name", hookName).
			Debug("preparing to execute hook")

		k.burnerRaffleCleanup(ctx, epochNumber)

		return nil
	})
}

func (k Keeper) burnerRaffleCleanup(ctx sdk.Context, epochNumber int64) {
	toDelete := k.GetRaffleDeleteHookByEndAtPrefix(ctx, uint64(epochNumber))
	if len(toDelete) == 0 {
		return
	}

	for _, item := range toDelete {
		logger := k.Logger(ctx).With("epoch", epochNumber, "hook", hookName, "to_delete", item)
		k.RemoveRaffleDeleteHook(ctx, item)
		k.RemoveRaffle(ctx, item.Denom)
		winners := k.GetRaffleWinners(ctx, item.Denom)
		for _, w := range winners {
			k.RemoveRaffleWinner(ctx, w)
		}

		//get raffle module account
		raffleAcc := k.accKeeper.GetModuleAccount(ctx, types.RaffleModuleName)
		if raffleAcc == nil {
			logger.Error("could not find module account")
			continue
		}

		currentPot := k.bankKeeper.GetBalance(ctx, raffleAcc.GetAddress(), item.Denom)
		if !currentPot.IsPositive() {
			logger.Info("no coins to burn for this raffle that we delete")
			continue
		}

		err := k.bankKeeper.BurnCoins(ctx, types.RaffleModuleName, sdk.NewCoins(currentPot))
		if err != nil {
			logger.Error("failed to burn raffle remaining coins", "error", err)
			continue
		}

		logger.Debug("burned raffle coins", "burned_current_pot", currentPot)

		err = ctx.EventManager().EmitTypedEvent(&types.RaffleFinishedEvent{Denom: item.Denom})
		if err != nil {
			logger.Error("failed to emit raffle finished event", "error", err)
		}
	}
}
