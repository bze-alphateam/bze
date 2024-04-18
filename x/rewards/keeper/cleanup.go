package keeper

import (
	"github.com/bze-alphateam/bze/x/rewards/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k Keeper) removeExpiredTradingRewards(ctx sdk.Context, epochNumber int64) {
	logger := k.Logger(ctx).With("epoch_number", epochNumber)
	toRemove := k.GetAllPendingTradingRewardExpirationByExpireAt(ctx, uint32(epochNumber))
	for _, exp := range toRemove {
		k.RemovePendingTradingRewardExpiration(ctx, exp.ExpireAt, exp.RewardId)
		tr, found := k.GetPendingTradingReward(ctx, exp.RewardId)
		if !found {
			logger.With("trading_reward_expiration", exp).Error("trading reward not found for this trading reward expiration")
			continue
		}
		k.RemovePendingTradingReward(ctx, exp.RewardId)

		//burn coins that were captured
		toBurn, err := k.getAmountToCapture("", tr.PrizeDenom, tr.PrizeAmount, int64(tr.Slots))
		if err != nil {
			logger.With("trading_reward", tr).Error("could not create amount to burn from trading reward")
			continue
		}

		err = k.bankKeeper.BurnCoins(ctx, types.ModuleName, toBurn)
		if err != nil {
			logger.With("trading_reward", tr, "to_burn", toBurn).Error("could not burn coins for trading reward")
			continue
		}

		logger.With("trading_reward", tr, "to_burn", toBurn).
			Debug("removed expired trading reward and burnt the tokens associated with it")

		err = ctx.EventManager().EmitTypedEvent(
			&types.TradingRewardExpireEvent{
				RewardId: exp.RewardId,
			},
		)

		if err != nil {
			k.Logger(ctx).Error(err.Error())
		}
	}
}
