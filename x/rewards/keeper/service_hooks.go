package keeper

import (
	"slices"

	"cosmossdk.io/math"
	"github.com/bze-alphateam/bze/x/rewards/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k Keeper) GetDistributeAllStakingRewardsHook() types.EpochHook {
	hookName := "distribution_hook"
	return types.NewAfterEpochHook(hookName, func(ctx sdk.Context, epochIdentifier string, epochNumber int64) error {
		if epochIdentifier != distributionEpoch {
			return nil
		}

		k.Logger().
			With("epoch", epochIdentifier, "epoch_number", epochNumber, "hook_name", hookName).
			Debug("preparing to execute hook")

		k.EnqueueStakingRewardsDistribution(ctx)

		return nil
	})
}

func (k Keeper) GetUnlockPendingUnlockParticipantsHook() types.EpochHook {
	hookName := "pending_unlock_hook"
	return types.NewAfterEpochHook(hookName, func(ctx sdk.Context, epochIdentifier string, epochNumber int64) error {
		if epochIdentifier != expirationEpoch {
			return nil
		}

		k.Logger().
			With("epoch", epochIdentifier, "epoch_number", epochNumber, "hook_name", hookName).
			Debug("preparing to execute hook")

		k.EnqueueUnlockParticipants(ctx, epochNumber)

		return nil
	})
}

func (k Keeper) GetRemoveExpiredPendingTradingRewardsHook() types.EpochHook {
	hookName := "remove_expired_trading_rewards"
	return types.NewAfterEpochHook(hookName, func(ctx sdk.Context, epochIdentifier string, epochNumber int64) error {
		if epochIdentifier != expirationEpoch {
			return nil
		}

		k.Logger().
			With("epoch", epochIdentifier, "epoch_number", epochNumber, "hook_name", hookName).
			Debug("preparing to execute hook")

		k.EnqueueExpiredTradingRewardRemoval(ctx, epochNumber)

		return nil
	})
}

func (k Keeper) GetTradingRewardsDistributionHook() types.EpochHook {
	hookName := "trading_rewards_distribution"
	return types.NewAfterEpochHook(hookName, func(ctx sdk.Context, epochIdentifier string, epochNumber int64) error {
		if epochIdentifier != expirationEpoch {
			return nil
		}

		k.Logger().
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
		candidateAmount, ok := math.NewIntFromString(candidate.Amount)
		if !ok {
			logger.Error("could not parse candidate amount")

			return
		}

		tradedAmount, ok := math.NewIntFromString(amountTraded)
		if !ok {
			logger.Error("could not parse traded amount")

			return
		}

		if !tradedAmount.IsPositive() {
			logger.Error("traded amount is not positive")
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
		//safe to iterate over the slice since it's limited to 10 entries
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
		slices.SortStableFunc(leaderboard.List, func(a, b types.TradingRewardLeaderboardEntry) int {
			aAmt, aOk := math.NewIntFromString(a.Amount)
			if !aOk {
				logger.Error("could not parse amount from leaderboard entry", "entry", a)
				aAmt = math.ZeroInt()
			}
			bAmt, bOk := math.NewIntFromString(b.Amount)
			if !bOk {
				logger.Error("could not parse amount from leaderboard entry", "entry", b)
				bAmt = math.ZeroInt()
			}

			//if the amounts are equal, use CreatedAt to sort
			if aAmt.Equal(bAmt) {
				if a.CreatedAt == b.CreatedAt {
					//keep the original order -> ensures deterministic leaderboard
					return 0
				}

				if a.CreatedAt < b.CreatedAt {
					return -1
				}

				return 1
			}

			if aAmt.GT(bAmt) {
				return -1
			}

			return 1
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
