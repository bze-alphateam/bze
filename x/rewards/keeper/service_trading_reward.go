package keeper

import (
	"github.com/bze-alphateam/bze/x/rewards/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	oneWeekInHours = 7 * 24
)

func (k Keeper) removeExpiredPendingTradingRewards(ctx sdk.Context, epochNumber int64) {
	logger := k.Logger().With("epoch_number", epochNumber)
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
		toBurn, err := k.getAmountToCapture(tr.PrizeDenom, tr.PrizeAmount, int64(tr.Slots))
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
			k.Logger().Error(err.Error())
		}
	}
}

func (k Keeper) distributeTradingRewards(ctx sdk.Context, epochNumber int64) {
	logger := k.Logger().With("epoch_number", epochNumber)
	toPay := k.GetAllActiveTradingRewardExpirationByExpireAt(ctx, uint32(epochNumber))
	for _, exp := range toPay {
		tr, found := k.GetActiveTradingReward(ctx, exp.RewardId)
		if !found {
			logger.With("trading_reward_expiration", exp).Error("trading reward not found for this trading reward expiration")
			continue
		}
		logger = logger.With("trading_reward", tr)

		//delete those already paid in a previous epoch that were extended just so we can display them for a few more days
		if int64(tr.ExpireAt) != epochNumber {
			k.RemoveActiveTradingRewardExpiration(ctx, exp.ExpireAt, exp.RewardId)
			k.RemoveActiveTradingReward(ctx, exp.RewardId)
			k.RemoveTradingRewardLeaderboard(ctx, exp.RewardId)
			k.cleanupTradingRewardCandidates(ctx, exp.RewardId)

			continue
		}

		rewardPerSlot, err := k.getAmountToCapture(tr.PrizeDenom, tr.PrizeAmount, 1)
		if err != nil {
			logger.Error("could not get reward per slot")
			continue
		}

		leaderboard, found := k.GetTradingRewardLeaderboard(ctx, tr.RewardId)
		if !found {
			logger.Error("trading reward leaderboard not found")
			continue
		}

		var eventWinners []string
		for _, winner := range leaderboard.List {
			acc, err := sdk.AccAddressFromBech32(winner.Address)
			if err != nil {
				panic(err)
			}

			_ = k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, acc, rewardPerSlot)
			eventWinners = append(eventWinners, winner.Address)
		}

		//the trading reward is finished. remove the market id
		k.RemoveMarketIdRewardId(ctx, tr.MarketId)
		//extend the expiration of this paid trading reward for another period just to display the winners
		exp.ExpireAt += oneWeekInHours
		k.SetActiveTradingRewardExpiration(ctx, exp)

		err = ctx.EventManager().EmitTypedEvent(
			&types.TradingRewardDistributionEvent{
				RewardId:    exp.RewardId,
				PrizeAmount: rewardPerSlot.String(),
				PrizeDenom:  tr.PrizeDenom,
				Winners:     eventWinners,
			},
		)

		if err != nil {
			k.Logger().Error(err.Error())
		}
	}
}

func (k Keeper) cleanupTradingRewardCandidates(ctx sdk.Context, rewardId string) {
	toRemove := k.GetTradingRewardCandidateByRewardId(ctx, rewardId)
	for _, trc := range toRemove {
		k.RemoveTradingRewardCandidate(ctx, trc.RewardId, trc.Address)
	}
}
