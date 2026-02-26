package keeper

import (
	"fmt"

	"cosmossdk.io/math"
	"github.com/bze-alphateam/bze/x/rewards/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// EnqueueStakingRewardsDistribution checks if any staking rewards exist and enqueues
// a distribution request. The module will process the queue in the following blocks.
func (k Keeper) EnqueueStakingRewardsDistribution(ctx sdk.Context) {
	var hasStakingRewards bool
	k.IterateAllStakingRewards(ctx, func(ctx sdk.Context, sr types.StakingReward) (stop bool) {
		hasStakingRewards = true
		return true
	})

	if !hasStakingRewards {
		return
	}

	queue, found := k.GetStakingRewardsDistributionQueue(ctx)
	if found && queue.Pending {
		// distribution already pending, skip
		return
	}

	queue = types.StakingRewardsDistributionQueue{
		Pending: true,
		Cursor:  "",
	}
	k.SetStakingRewardsDistributionQueue(ctx, queue)
}

// ProcessStakingRewardsDistributionQueue processes staking reward distributions in bounded batches.
// It collects entries to distribute in batches to avoid unbounded iteration within a single block.
func (k Keeper) ProcessStakingRewardsDistributionQueue(ctx sdk.Context) {
	queue, found := k.GetStakingRewardsDistributionQueue(ctx)
	if !found || !queue.Pending {
		return
	}

	rewards := k.GetBatchStakingRewards(ctx, queue.Cursor, types.MaxStakingDistributionsPerBlock)
	if len(rewards) == 0 {
		// no more rewards to process, distribution is complete
		k.RemoveStakingRewardsDistributionQueue(ctx)
		return
	}

	finished := len(rewards) < types.MaxStakingDistributionsPerBlock

	// Process collected entries in a safe context
	lastProcessedId := queue.Cursor
	for _, sr := range rewards {
		lastProcessedId = sr.RewardId
		k.distributeStakingReward(ctx, sr)
	}

	if finished {
		k.RemoveStakingRewardsDistributionQueue(ctx)
	} else {
		queue.Cursor = lastProcessedId
		k.SetStakingRewardsDistributionQueue(ctx, queue)
	}
}

func (k Keeper) distributeStakingReward(ctx sdk.Context, sr types.StakingReward) {
	logger := k.Logger().With("staking_reward", sr)

	logger.Debug("preparing to distribute staking reward")

	if sr.StakedAmount.IsZero() {
		logger.Debug("staking reward has no staked coins. skipping distribution")
		return
	}

	if sr.Payouts >= sr.Duration {
		logger.Debug("staking reward finished. skipping distribution")
		return
	}

	err := k.distributeStakingRewards(&sr, sr.PrizeAmount)
	if err != nil {
		logger.Error(err.Error())
		return
	}

	//increment payouts to know when the reward finished (a.k.a. all payouts calculated)
	sr.Payouts++
	k.SetStakingReward(ctx, sr)

	err = ctx.EventManager().EmitTypedEvent(
		&types.StakingRewardDistributionEvent{
			RewardId: sr.RewardId,
			Amount:   sr.PrizeAmount.String(),
		},
	)

	if err != nil {
		k.Logger().Error(err.Error())
	}
}

func (k Keeper) distributeStakingRewards(sr *types.StakingReward, rewardAmount math.Int) error {
	stakedAmount := math.LegacyNewDecFromInt(sr.StakedAmount)

	if !stakedAmount.IsPositive() {
		return fmt.Errorf("no stakers found")
	}

	reward := math.LegacyNewDecFromInt(rewardAmount)

	if !reward.IsPositive() {
		return fmt.Errorf("reward amount should be positive")
	}

	sFloat := sr.DistributedStake

	//S = S + r / T;
	sFloat = sFloat.Add(reward.Quo(stakedAmount))
	sr.DistributedStake = sFloat

	return nil
}
