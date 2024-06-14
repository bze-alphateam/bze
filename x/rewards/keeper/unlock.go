package keeper

import (
	"github.com/bze-alphateam/bze/x/rewards/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k Keeper) UnlockAllPendingUnlockParticipantsByEpoch(ctx sdk.Context, epochNumber int64) {
	pending := k.GetAllEpochPendingUnlockParticipant(ctx, epochNumber)
	for _, p := range pending {
		logger := k.Logger(ctx).With("pending_unlock", p)

		err := k.performUnlock(ctx, &p)
		if err != nil {
			logger.Error(err.Error())
			continue
		}

		k.RemovePendingUnlockParticipant(ctx, p)
	}
}

func (k Keeper) performUnlock(ctx sdk.Context, p *types.PendingUnlockParticipant) error {
	partCoins, err := k.getAmountToCapture("", p.Denom, p.Amount, int64(1))
	if err != nil {
		return err
	}

	acc, err := sdk.AccAddressFromBech32(p.Address)
	if err != nil {
		return err
	}

	err = k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, acc, partCoins)
	if err != nil {
		return err
	}

	return nil
}
