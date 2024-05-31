package keeper

import (
	"github.com/bze-alphateam/bze/x/rewards/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k Keeper) UnlockAllPendingUnlockParticipantsByEpoch(ctx sdk.Context, epochNumber int64) {
	pending := k.GetAllEpochPendingUnlockParticipant(ctx, epochNumber)
	for _, p := range pending {
		logger := k.Logger(ctx).With("pending_unlock", p)
		partCoins, err := k.getAmountToCapture("", p.Denom, p.Amount, int64(1))
		if err != nil {
			logger.Error(err.Error())
			continue
		}

		acc, err := sdk.AccAddressFromBech32(p.Address)
		if err != nil {
			logger.Error(err.Error())
			continue
		}

		err = k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, acc, partCoins)
		if err != nil {
			logger.Error(err.Error())
			continue
		}

		k.RemovePendingUnlockParticipant(ctx, p)
	}
}
