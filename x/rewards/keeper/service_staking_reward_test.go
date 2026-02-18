package keeper_test

import (
	"fmt"
	"github.com/bze-alphateam/bze/x/rewards/types"
)

// --- EnqueueStakingRewardsDistribution tests ---

func (suite *IntegrationTestSuite) TestEnqueueStakingRewardsDistribution_Empty() {
	// No staking rewards - should not create queue
	suite.k.EnqueueStakingRewardsDistribution(suite.ctx)

	_, found := suite.k.GetStakingRewardsDistributionQueue(suite.ctx)
	suite.Require().False(found)
}

func (suite *IntegrationTestSuite) TestEnqueueStakingRewardsDistribution_WithRewards() {
	suite.k.SetStakingReward(suite.ctx, types.StakingReward{
		RewardId:         "reward-1",
		PrizeAmount:      "1000",
		PrizeDenom:       "ubze",
		StakingDenom:     "ubze",
		Duration:         5,
		Payouts:          0,
		MinStake:         100,
		Lock:             7,
		StakedAmount:     "5000",
		DistributedStake: "0",
	})

	suite.k.EnqueueStakingRewardsDistribution(suite.ctx)

	queue, found := suite.k.GetStakingRewardsDistributionQueue(suite.ctx)
	suite.Require().True(found)
	suite.Require().True(queue.Pending)
	suite.Require().Equal("", queue.Cursor)
}

func (suite *IntegrationTestSuite) TestEnqueueStakingRewardsDistribution_AlreadyPending() {
	suite.k.SetStakingReward(suite.ctx, types.StakingReward{
		RewardId:         "reward-1",
		PrizeAmount:      "1000",
		PrizeDenom:       "ubze",
		StakingDenom:     "ubze",
		Duration:         5,
		Payouts:          0,
		MinStake:         100,
		Lock:             7,
		StakedAmount:     "5000",
		DistributedStake: "0",
	})

	// Enqueue first time
	suite.k.EnqueueStakingRewardsDistribution(suite.ctx)

	// Set cursor to simulate in-progress distribution
	queue, found := suite.k.GetStakingRewardsDistributionQueue(suite.ctx)
	suite.Require().True(found)
	queue.Cursor = "reward-1"
	suite.k.SetStakingRewardsDistributionQueue(suite.ctx, queue)

	// Enqueue again - should not reset cursor
	suite.k.EnqueueStakingRewardsDistribution(suite.ctx)

	queue, found = suite.k.GetStakingRewardsDistributionQueue(suite.ctx)
	suite.Require().True(found)
	suite.Require().True(queue.Pending)
	suite.Require().Equal("reward-1", queue.Cursor) // cursor should be preserved
}

// --- ProcessStakingRewardsDistributionQueue tests ---

func (suite *IntegrationTestSuite) TestProcessStakingDistributionQueue_EmptyQueue() {
	// No queue at all - should not panic
	suite.Require().NotPanics(func() {
		suite.k.ProcessStakingRewardsDistributionQueue(suite.ctx)
	})
}

func (suite *IntegrationTestSuite) TestProcessStakingDistributionQueue_NotPending() {
	suite.k.SetStakingRewardsDistributionQueue(suite.ctx, types.StakingRewardsDistributionQueue{
		Pending: false,
		Cursor:  "",
	})

	suite.Require().NotPanics(func() {
		suite.k.ProcessStakingRewardsDistributionQueue(suite.ctx)
	})
}

func (suite *IntegrationTestSuite) TestProcessStakingDistributionQueue_SingleReward() {
	suite.k.SetStakingReward(suite.ctx, types.StakingReward{
		RewardId:         "reward-1",
		PrizeAmount:      "1000",
		PrizeDenom:       "ubze",
		StakingDenom:     "ubze",
		Duration:         5,
		Payouts:          2,
		MinStake:         100,
		Lock:             7,
		StakedAmount:     "5000",
		DistributedStake: "100",
	})

	suite.k.EnqueueStakingRewardsDistribution(suite.ctx)
	suite.k.ProcessStakingRewardsDistributionQueue(suite.ctx)

	// Verify reward was processed (payouts incremented)
	reward, found := suite.k.GetStakingReward(suite.ctx, "reward-1")
	suite.Require().True(found)
	suite.Require().Equal(uint32(3), reward.Payouts)

	// Verify queue was removed (distribution complete)
	_, found = suite.k.GetStakingRewardsDistributionQueue(suite.ctx)
	suite.Require().False(found)
}

func (suite *IntegrationTestSuite) TestProcessStakingDistributionQueue_MultipleRewards() {
	for i := 1; i <= 3; i++ {
		suite.k.SetStakingReward(suite.ctx, types.StakingReward{
			RewardId:         fmt.Sprintf("reward-%d", i),
			PrizeAmount:      "1000",
			PrizeDenom:       "ubze",
			StakingDenom:     "ubze",
			Duration:         5,
			Payouts:          0,
			MinStake:         100,
			Lock:             7,
			StakedAmount:     "5000",
			DistributedStake: "0",
		})
	}

	suite.k.EnqueueStakingRewardsDistribution(suite.ctx)
	suite.k.ProcessStakingRewardsDistributionQueue(suite.ctx)

	// Verify all rewards were processed
	for i := 1; i <= 3; i++ {
		reward, found := suite.k.GetStakingReward(suite.ctx, fmt.Sprintf("reward-%d", i))
		suite.Require().True(found)
		suite.Require().Equal(uint32(1), reward.Payouts)
	}

	// Verify queue was removed
	_, found := suite.k.GetStakingRewardsDistributionQueue(suite.ctx)
	suite.Require().False(found)
}

func (suite *IntegrationTestSuite) TestProcessStakingDistributionQueue_SkipsZeroStaked() {
	suite.k.SetStakingReward(suite.ctx, types.StakingReward{
		RewardId:         "reward-1",
		PrizeAmount:      "1000",
		PrizeDenom:       "ubze",
		StakingDenom:     "ubze",
		Duration:         5,
		Payouts:          0,
		MinStake:         100,
		Lock:             7,
		StakedAmount:     "0", // no stakers
		DistributedStake: "0",
	})

	suite.k.EnqueueStakingRewardsDistribution(suite.ctx)
	suite.k.ProcessStakingRewardsDistributionQueue(suite.ctx)

	// Verify reward was not distributed (payouts stays 0)
	reward, found := suite.k.GetStakingReward(suite.ctx, "reward-1")
	suite.Require().True(found)
	suite.Require().Equal(uint32(0), reward.Payouts)

	// Queue should be removed
	_, found = suite.k.GetStakingRewardsDistributionQueue(suite.ctx)
	suite.Require().False(found)
}

func (suite *IntegrationTestSuite) TestProcessStakingDistributionQueue_SkipsFinishedReward() {
	suite.k.SetStakingReward(suite.ctx, types.StakingReward{
		RewardId:         "reward-1",
		PrizeAmount:      "1000",
		PrizeDenom:       "ubze",
		StakingDenom:     "ubze",
		Duration:         5,
		Payouts:          5, // already finished
		MinStake:         100,
		Lock:             7,
		StakedAmount:     "5000",
		DistributedStake: "100",
	})

	suite.k.EnqueueStakingRewardsDistribution(suite.ctx)
	suite.k.ProcessStakingRewardsDistributionQueue(suite.ctx)

	// Verify reward was not distributed (payouts stays 5)
	reward, found := suite.k.GetStakingReward(suite.ctx, "reward-1")
	suite.Require().True(found)
	suite.Require().Equal(uint32(5), reward.Payouts)

	// Queue should be removed
	_, found = suite.k.GetStakingRewardsDistributionQueue(suite.ctx)
	suite.Require().False(found)
}

func (suite *IntegrationTestSuite) TestProcessStakingDistributionQueue_MultiBlockDrain() {
	totalRewards := types.MaxStakingDistributionsPerBlock + 50 // 150 rewards

	for i := 1; i <= totalRewards; i++ {
		suite.k.SetStakingReward(suite.ctx, types.StakingReward{
			RewardId:         fmt.Sprintf("drain-reward-%03d", i), // zero-padded for deterministic ordering
			PrizeAmount:      "1000",
			PrizeDenom:       "ubze",
			StakingDenom:     "ubze",
			Duration:         5,
			Payouts:          0,
			MinStake:         100,
			Lock:             7,
			StakedAmount:     "5000",
			DistributedStake: "0",
		})
	}

	suite.k.EnqueueStakingRewardsDistribution(suite.ctx)

	// First block: should process MaxStakingDistributionsPerBlock entries
	suite.k.ProcessStakingRewardsDistributionQueue(suite.ctx)

	// Queue should still be pending with a cursor set
	queue, found := suite.k.GetStakingRewardsDistributionQueue(suite.ctx)
	suite.Require().True(found)
	suite.Require().True(queue.Pending)
	suite.Require().NotEmpty(queue.Cursor)

	// Count processed (payouts == 1) vs unprocessed (payouts == 0)
	processedCount := 0
	unprocessedCount := 0
	for i := 1; i <= totalRewards; i++ {
		reward, found := suite.k.GetStakingReward(suite.ctx, fmt.Sprintf("drain-reward-%03d", i))
		suite.Require().True(found)
		if reward.Payouts == 1 {
			processedCount++
		} else {
			unprocessedCount++
		}
	}
	suite.Require().Equal(types.MaxStakingDistributionsPerBlock, processedCount)
	suite.Require().Equal(50, unprocessedCount)

	// Second block: should process remaining 50 and remove queue
	suite.k.ProcessStakingRewardsDistributionQueue(suite.ctx)

	_, found = suite.k.GetStakingRewardsDistributionQueue(suite.ctx)
	suite.Require().False(found)

	// All rewards should now be processed
	for i := 1; i <= totalRewards; i++ {
		reward, found := suite.k.GetStakingReward(suite.ctx, fmt.Sprintf("drain-reward-%03d", i))
		suite.Require().True(found)
		suite.Require().Equal(uint32(1), reward.Payouts)
	}
}

func (suite *IntegrationTestSuite) TestProcessStakingDistributionQueue_ExactlyAtBatchLimit() {
	totalRewards := types.MaxStakingDistributionsPerBlock // exactly 100

	for i := 1; i <= totalRewards; i++ {
		suite.k.SetStakingReward(suite.ctx, types.StakingReward{
			RewardId:         fmt.Sprintf("exact-reward-%03d", i),
			PrizeAmount:      "1000",
			PrizeDenom:       "ubze",
			StakingDenom:     "ubze",
			Duration:         5,
			Payouts:          0,
			MinStake:         100,
			Lock:             7,
			StakedAmount:     "5000",
			DistributedStake: "0",
		})
	}

	suite.k.EnqueueStakingRewardsDistribution(suite.ctx)

	// First block: processes exactly MaxStakingDistributionsPerBlock entries
	// len(rewards) == limit, so finished = false, cursor is set
	suite.k.ProcessStakingRewardsDistributionQueue(suite.ctx)

	// Queue should still exist since len(rewards) was NOT less than limit
	queue, found := suite.k.GetStakingRewardsDistributionQueue(suite.ctx)
	suite.Require().True(found)
	suite.Require().True(queue.Pending)

	// All rewards should be processed
	for i := 1; i <= totalRewards; i++ {
		reward, found := suite.k.GetStakingReward(suite.ctx, fmt.Sprintf("exact-reward-%03d", i))
		suite.Require().True(found)
		suite.Require().Equal(uint32(1), reward.Payouts)
	}

	// Second block: GetBatchStakingRewards returns empty, queue is removed
	suite.k.ProcessStakingRewardsDistributionQueue(suite.ctx)

	_, found = suite.k.GetStakingRewardsDistributionQueue(suite.ctx)
	suite.Require().False(found)
}

func (suite *IntegrationTestSuite) TestProcessStakingDistributionQueue_CursorResumesFromLastProcessed() {
	// Set a cursor pointing to reward-1, meaning reward-1 was already processed
	suite.k.SetStakingRewardsDistributionQueue(suite.ctx, types.StakingRewardsDistributionQueue{
		Pending: true,
		Cursor:  "reward-1",
	})

	suite.k.SetStakingReward(suite.ctx, types.StakingReward{
		RewardId:         "reward-1",
		PrizeAmount:      "1000",
		PrizeDenom:       "ubze",
		StakingDenom:     "ubze",
		Duration:         5,
		Payouts:          1, // already distributed once
		MinStake:         100,
		Lock:             7,
		StakedAmount:     "5000",
		DistributedStake: "100",
	})

	suite.k.SetStakingReward(suite.ctx, types.StakingReward{
		RewardId:         "reward-2",
		PrizeAmount:      "500",
		PrizeDenom:       "ubze",
		StakingDenom:     "ubze",
		Duration:         5,
		Payouts:          0,
		MinStake:         100,
		Lock:             7,
		StakedAmount:     "3000",
		DistributedStake: "0",
	})

	suite.k.ProcessStakingRewardsDistributionQueue(suite.ctx)

	// reward-1 should NOT be processed again (cursor was past it)
	reward1, found := suite.k.GetStakingReward(suite.ctx, "reward-1")
	suite.Require().True(found)
	suite.Require().Equal(uint32(1), reward1.Payouts) // unchanged

	// reward-2 should be processed
	reward2, found := suite.k.GetStakingReward(suite.ctx, "reward-2")
	suite.Require().True(found)
	suite.Require().Equal(uint32(1), reward2.Payouts) // incremented

	// Queue should be removed (all processed)
	_, found = suite.k.GetStakingRewardsDistributionQueue(suite.ctx)
	suite.Require().False(found)
}
