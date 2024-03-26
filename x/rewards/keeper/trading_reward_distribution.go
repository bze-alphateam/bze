package keeper

import (
	"github.com/bze-alphateam/bze/x/rewards/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	oneWeekInHours = 7 * 24
)

func (k Keeper) distributeTradingRewards(ctx sdk.Context, epochNumber int64) {
	logger := k.Logger(ctx).With("epoch_number", epochNumber)
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

			continue
		}

		rewardPerSlot, err := k.getAmountToCapture("", tr.PrizeDenom, tr.PrizeAmount, 1)
		if err != nil {
			logger.Error("could not get reward per slot")
			continue
		}

		leaderboard, found := k.GetTradingRewardLeaderboard(ctx, tr.RewardId)
		if !found {
			logger.Error("trading reward leaderboard not found")
			continue
		}

		for _, winner := range leaderboard.List {
			acc, err := sdk.AccAddressFromBech32(winner.Address)
			if err != nil {
				panic(err)
			}

			_ = k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, acc, rewardPerSlot)
		}

		//the trading reward is finished. remove the market id
		k.RemoveMarketIdRewardId(ctx, tr.MarketId)
		//extend the expiration of this paid trading reward for another period just to display the winners
		exp.ExpireAt += oneWeekInHours
		k.SetActiveTradingRewardExpiration(ctx, exp)
	}
}
