package keeper

import (
	"github.com/bze-alphateam/bze/x/rewards/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"sort"
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
		if epochIdentifier != expirationEpoch {
			return nil
		}

		k.Logger(ctx).
			With("epoch", epochIdentifier, "epoch_number", epochNumber, "hook_name", hookName).
			Debug("preparing to execute hook")

		k.UnlockAllPendingUnlockParticipantsByEpoch(ctx, epochNumber)

		return nil
	})
}

func (k Keeper) GetRemoveExpiredPendingTradingRewardsHook() types.EpochHook {
	hookName := "remove_expired_trading_rewards"
	return types.NewAfterEpochHook(hookName, func(ctx sdk.Context, epochIdentifier string, epochNumber int64) error {
		if epochIdentifier != expirationEpoch {
			return nil
		}

		k.Logger(ctx).
			With("epoch", epochIdentifier, "epoch_number", epochNumber, "hook_name", hookName).
			Debug("preparing to execute hook")

		k.removeExpiredPendingTradingRewards(ctx, epochNumber)

		return nil
	})
}

func (k Keeper) GetTradingRewardsDistributionHook() types.EpochHook {
	hookName := "trading_rewards_distribution"
	return types.NewAfterEpochHook(hookName, func(ctx sdk.Context, epochIdentifier string, epochNumber int64) error {
		if epochIdentifier != expirationEpoch {
			return nil
		}

		k.Logger(ctx).
			With("epoch", epochIdentifier, "epoch_number", epochNumber, "hook_name", hookName).
			Debug("preparing to execute hook")

		k.distributeTradingRewards(ctx, epochNumber)

		return nil
	})
}

func (k Keeper) GetOnOrderFillHook() func(ctx sdk.Context, marketId, amountTraded, userAddress string) {
	return func(ctx sdk.Context, marketId, amountTraded, userAddress string) {
		logger := ctx.Logger().With("market_id", marketId)
		existingReward, found := k.GetMarketIdRewardId(ctx, marketId)
		if !found {
			logger.Debug("no rewards found for this market id")
			return
		}

		reward, found := k.GetActiveTradingReward(ctx, existingReward.RewardId)
		if !found {
			logger.With("reward_id", existingReward).
				Error("the reward id found for this market does not exist in active trading reward store")
			return
		}
		logger = logger.With("reward", reward)

		candidate, found := k.GetTradingRewardCandidate(ctx, reward.RewardId, userAddress)
		if !found {
			logger.Debug("candidate not found. creating a new one")
			candidate = types.TradingRewardCandidate{
				RewardId: reward.RewardId,
				Amount:   "0",
				Address:  userAddress,
			}
		}
		candidateAmount, ok := sdk.NewIntFromString(candidate.Amount)
		if !ok {
			logger.Error("could not parse candidate amount")

			return
		}

		tradedAmount, ok := sdk.NewIntFromString(amountTraded)
		if !ok {
			logger.Error("could not parse traded amount")

			return
		}

		candidateAmount = candidateAmount.Add(tradedAmount)
		candidate.Amount = candidateAmount.String()
		k.SetTradingRewardCandidate(ctx, candidate)
		logger.Debug("trading reward candidate saved")

		//try to add to leaderboard
		leaderboard, found := k.GetTradingRewardLeaderboard(ctx, reward.RewardId)
		if !found {
			logger.Debug("leaderboard does not exist. creating new one")
			leaderboard = types.TradingRewardLeaderboard{
				RewardId: reward.RewardId,
				List:     []types.TradingRewardLeaderboardEntry{},
			}
		}

		addedToList := false
		for i, entry := range leaderboard.List {
			if candidate.Address != entry.Address {
				continue
			}

			entry.Amount = candidateAmount.String()
			leaderboard.List[i] = entry
			addedToList = true

			logger.Debug("candidate already exists in leaderboard")

			break
		}

		//not found in leaderboard, let's add it
		if !addedToList {
			logger.Debug("candidate does not exists in leaderboard. creating new entry")
			newEntry := types.TradingRewardLeaderboardEntry{
				Amount:    candidate.Amount,
				Address:   candidate.Address,
				CreatedAt: ctx.BlockTime().Unix(),
			}
			leaderboard.List = append(leaderboard.List, newEntry)
		}

		//sort the slice
		sort.SliceStable(leaderboard.List[:], func(i, j int) bool {
			iAmt, _ := sdk.NewIntFromString(amountTraded)
			jAmt, _ := sdk.NewIntFromString(amountTraded)
			if iAmt.GT(jAmt) {
				return true
			}
			if iAmt.LT(jAmt) {
				return false
			}

			return leaderboard.List[i].CreatedAt < leaderboard.List[j].CreatedAt
		})

		//trim slice if it's longer than the rewarded slots
		if reward.Slots < uint32(len(leaderboard.List)) {
			logger.Debug("trimming leaderboard list")
			leaderboard.List = leaderboard.List[:reward.Slots]
		}

		k.SetTradingRewardLeaderboard(ctx, leaderboard)
		logger.Debug("leaderboard set into storage")
	}
}
