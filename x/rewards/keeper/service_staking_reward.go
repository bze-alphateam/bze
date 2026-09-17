package keeper

import (
	"fmt"

	"cosmossdk.io/math"
	"github.com/bze-alphateam/bze/bzeutils"
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

	// Each reward is paid inside its own recovering cache context (the iterator above is already
	// closed, as ApplyFuncIfNoError requires): a panic in one reward's pass is logged, its writes
	// are discarded and the batch carries on, so a single bad record can neither halt the chain
	// nor wedge the queue.
	lastProcessedId := queue.Cursor
	for _, sr := range rewards {
		lastProcessedId = sr.RewardId
		k.safeDistributeStakingReward(ctx, sr)
	}

	if finished {
		k.RemoveStakingRewardsDistributionQueue(ctx)
	} else {
		queue.Cursor = lastProcessedId
		k.SetStakingRewardsDistributionQueue(ctx, queue)
	}
}

// safeDistributeStakingReward runs distributeStakingReward in a cache context with panic
// recovery. On a panic nothing the pass wrote survives and the reward is skipped for this day
// only: the next day tick enqueues it again, exactly as after a logged error.
func (k Keeper) safeDistributeStakingReward(ctx sdk.Context, sr types.StakingReward) {
	err := bzeutils.ApplyFuncIfNoError(ctx, func(c sdk.Context) error {
		k.distributeStakingReward(c, sr)

		return nil
	})
	if err != nil {
		k.Logger().Error(
			"staking reward distribution skipped after panic",
			"reward_id", sr.RewardId,
			"err", err,
		)
	}
}

func (k Keeper) distributeStakingReward(ctx sdk.Context, sr types.StakingReward) {
	logger := k.Logger().With("staking_reward", sr)

	logger.Debug("preparing to distribute staking reward")

	if sr.StakedAmount == "0" {
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
			Amount:   sr.PrizeAmount,
		},
	)

	if err != nil {
		k.Logger().Error(err.Error())
	}
}

func (k Keeper) distributeStakingRewards(sr *types.StakingReward, rewardAmount string) error {
	stakedAmount, err := math.LegacyNewDecFromStr(sr.StakedAmount)
	if err != nil {
		return fmt.Errorf("could not transform staked amount from storage into int: %w", err)
	}

	if !stakedAmount.IsPositive() {
		return fmt.Errorf("no stakers found")
	}

	reward, err := math.LegacyNewDecFromStr(rewardAmount)
	if err != nil {
		return fmt.Errorf("could not transform reward amount to int: %w", err)
	}

	if !reward.IsPositive() {
		return fmt.Errorf("reward amount should be positive")
	}

	sFloat, err := math.LegacyNewDecFromStr(sr.DistributedStake)
	if err != nil {
		return err
	}

	//S = S + r / T;
	// QuoTruncate (round down), never round-to-nearest. Participants claim floor(deposited·ΔS);
	// truncating the accumulator keeps Σ deposited·ΔS ≤ T·ΔS ≤ r, so summed claims can never exceed
	// the funded reward. The sub-unit remainder stays as dust in the pool (BZE-104).
	sFloat = sFloat.Add(reward.QuoTruncate(stakedAmount))
	sr.DistributedStake = sFloat.String()

	return nil
}
