package keeper_test

import (
	"cosmossdk.io/math"
	"fmt"
	"github.com/bze-alphateam/bze/x/rewards/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (suite *IntegrationTestSuite) TestStoreStakingReward_SetAndGet() {
	stakingReward := types.StakingReward{
		RewardId:         "reward-1",
		PrizeAmount:      math.NewInt(1000),
		PrizeDenom:       "ubze",
		StakingDenom:     "ubze",
		Duration:         30,
		Payouts:          10,
		MinStake:         100,
		Lock:             7,
		StakedAmount:     math.ZeroInt(),
		DistributedStake: math.LegacyZeroDec(),
	}

	// Test SetStakingReward
	suite.k.SetStakingReward(suite.ctx, stakingReward)

	// Test GetStakingReward
	retrievedReward, found := suite.k.GetStakingReward(suite.ctx, "reward-1")
	suite.Require().True(found)
	suite.Require().Equal(stakingReward.RewardId, retrievedReward.RewardId)
	suite.Require().Equal(stakingReward.PrizeAmount, retrievedReward.PrizeAmount)
	suite.Require().Equal(stakingReward.PrizeDenom, retrievedReward.PrizeDenom)
	suite.Require().Equal(stakingReward.StakingDenom, retrievedReward.StakingDenom)
	suite.Require().Equal(stakingReward.Duration, retrievedReward.Duration)
	suite.Require().Equal(stakingReward.Payouts, retrievedReward.Payouts)
	suite.Require().Equal(stakingReward.MinStake, retrievedReward.MinStake)
	suite.Require().Equal(stakingReward.Lock, retrievedReward.Lock)
	suite.Require().Equal(stakingReward.StakedAmount, retrievedReward.StakedAmount)
	suite.Require().Equal(stakingReward.DistributedStake, retrievedReward.DistributedStake)
}

func (suite *IntegrationTestSuite) TestStoreStakingReward_GetNonExistent() {
	// Test getting non-existent reward
	_, found := suite.k.GetStakingReward(suite.ctx, "non-existent-id")
	suite.Require().False(found)
}

func (suite *IntegrationTestSuite) TestStoreStakingReward_SetMultiple() {
	reward1 := types.StakingReward{
		RewardId:         "reward-1",
		PrizeAmount:      math.NewInt(1000),
		PrizeDenom:       "ubze",
		StakingDenom:     "ubze",
		Duration:         30,
		Payouts:          10,
		MinStake:         100,
		Lock:             7,
		StakedAmount:     math.NewInt(500),
		DistributedStake: math.LegacyMustNewDecFromStr("250"),
	}

	reward2 := types.StakingReward{
		RewardId:         "reward-2",
		PrizeAmount:      math.NewInt(2000),
		PrizeDenom:       "utoken",
		StakingDenom:     "ustake",
		Duration:         60,
		Payouts:          20,
		MinStake:         200,
		Lock:             14,
		StakedAmount:     math.NewInt(1000),
		DistributedStake: math.LegacyMustNewDecFromStr("500"),
	}

	// Set both rewards
	suite.k.SetStakingReward(suite.ctx, reward1)
	suite.k.SetStakingReward(suite.ctx, reward2)

	// Verify both can be retrieved independently
	retrievedReward1, found1 := suite.k.GetStakingReward(suite.ctx, "reward-1")
	suite.Require().True(found1)
	suite.Require().Equal(reward1.RewardId, retrievedReward1.RewardId)
	suite.Require().Equal(reward1.PrizeAmount, retrievedReward1.PrizeAmount)
	suite.Require().Equal(reward1.PrizeDenom, retrievedReward1.PrizeDenom)
	suite.Require().Equal(reward1.Duration, retrievedReward1.Duration)

	retrievedReward2, found2 := suite.k.GetStakingReward(suite.ctx, "reward-2")
	suite.Require().True(found2)
	suite.Require().Equal(reward2.RewardId, retrievedReward2.RewardId)
	suite.Require().Equal(reward2.PrizeAmount, retrievedReward2.PrizeAmount)
	suite.Require().Equal(reward2.StakingDenom, retrievedReward2.StakingDenom)
	suite.Require().Equal(reward2.MinStake, retrievedReward2.MinStake)
}

func (suite *IntegrationTestSuite) TestStoreStakingReward_UpdateExisting() {
	originalReward := types.StakingReward{
		RewardId:         "reward-update",
		PrizeAmount:      math.NewInt(500),
		PrizeDenom:       "ubze",
		StakingDenom:     "ubze",
		Duration:         30,
		Payouts:          5,
		MinStake:         50,
		Lock:             3,
		StakedAmount:     math.NewInt(100),
		DistributedStake: math.LegacyMustNewDecFromStr("50"),
	}

	updatedReward := types.StakingReward{
		RewardId:         "reward-update",
		PrizeAmount:      math.NewInt(1500),
		PrizeDenom:       "utoken",
		StakingDenom:     "ustake",
		Duration:         90,
		Payouts:          15,
		MinStake:         150,
		Lock:             10,
		StakedAmount:     math.NewInt(300),
		DistributedStake: math.LegacyMustNewDecFromStr("150"),
	}

	// Set original reward
	suite.k.SetStakingReward(suite.ctx, originalReward)

	// Update the reward
	suite.k.SetStakingReward(suite.ctx, updatedReward)

	// Verify the reward was updated
	retrievedReward, found := suite.k.GetStakingReward(suite.ctx, "reward-update")
	suite.Require().True(found)
	suite.Require().Equal(updatedReward.PrizeAmount, retrievedReward.PrizeAmount)
	suite.Require().Equal(updatedReward.PrizeDenom, retrievedReward.PrizeDenom)
	suite.Require().Equal(updatedReward.StakingDenom, retrievedReward.StakingDenom)
	suite.Require().Equal(updatedReward.Duration, retrievedReward.Duration)
	suite.Require().Equal(updatedReward.Payouts, retrievedReward.Payouts)
	suite.Require().Equal(updatedReward.MinStake, retrievedReward.MinStake)
	suite.Require().Equal(updatedReward.Lock, retrievedReward.Lock)
	suite.Require().Equal(updatedReward.StakedAmount, retrievedReward.StakedAmount)
	suite.Require().Equal(updatedReward.DistributedStake, retrievedReward.DistributedStake)
}

func (suite *IntegrationTestSuite) TestStoreStakingReward_Remove() {
	stakingReward := types.StakingReward{
		RewardId:         "reward-to-remove",
		PrizeAmount:      math.NewInt(750),
		PrizeDenom:       "ubze",
		StakingDenom:     "ubze",
		Duration:         45,
		Payouts:          8,
		MinStake:         75,
		Lock:             5,
		StakedAmount:     math.NewInt(200),
		DistributedStake: math.LegacyMustNewDecFromStr("100"),
	}

	// Set the reward
	suite.k.SetStakingReward(suite.ctx, stakingReward)

	// Verify it exists
	_, found := suite.k.GetStakingReward(suite.ctx, "reward-to-remove")
	suite.Require().True(found)

	// Remove the reward
	suite.k.RemoveStakingReward(suite.ctx, "reward-to-remove")

	// Verify it no longer exists
	_, found = suite.k.GetStakingReward(suite.ctx, "reward-to-remove")
	suite.Require().False(found)
}

func (suite *IntegrationTestSuite) TestStoreStakingReward_RemoveNonExistent() {
	// Removing non-existent reward should not cause issues
	suite.Require().NotPanics(func() {
		suite.k.RemoveStakingReward(suite.ctx, "non-existent-reward")
	})
}

func (suite *IntegrationTestSuite) TestStoreStakingReward_GetAllEmpty() {
	// Test GetAllStakingReward when no rewards exist
	allRewards := suite.k.GetAllStakingReward(suite.ctx)
	suite.Require().Empty(allRewards)
}

func (suite *IntegrationTestSuite) TestStoreStakingReward_GetAllMultiple() {
	rewards := []types.StakingReward{
		{
			RewardId:         "reward-1",
			PrizeAmount:      math.NewInt(100),
			PrizeDenom:       "ubze",
			StakingDenom:     "ubze",
			Duration:         30,
			Payouts:          5,
			MinStake:         10,
			Lock:             1,
			StakedAmount:     math.NewInt(50),
			DistributedStake: math.LegacyMustNewDecFromStr("25"),
		},
		{
			RewardId:         "reward-2",
			PrizeAmount:      math.NewInt(200),
			PrizeDenom:       "utoken",
			StakingDenom:     "ustake",
			Duration:         60,
			Payouts:          10,
			MinStake:         20,
			Lock:             2,
			StakedAmount:     math.NewInt(100),
			DistributedStake: math.LegacyMustNewDecFromStr("50"),
		},
		{
			RewardId:         "reward-3",
			PrizeAmount:      math.NewInt(300),
			PrizeDenom:       "ucoin",
			StakingDenom:     "ucoin",
			Duration:         90,
			Payouts:          15,
			MinStake:         30,
			Lock:             3,
			StakedAmount:     math.NewInt(150),
			DistributedStake: math.LegacyMustNewDecFromStr("75"),
		},
	}

	// Set all rewards
	for _, reward := range rewards {
		suite.k.SetStakingReward(suite.ctx, reward)
	}

	// Get all rewards
	allRewards := suite.k.GetAllStakingReward(suite.ctx)
	suite.Require().Len(allRewards, 3)

	// Verify all rewards are present (order might vary)
	rewardIds := make(map[string]bool)
	for _, reward := range allRewards {
		rewardIds[reward.RewardId] = true
	}

	suite.Require().True(rewardIds["reward-1"])
	suite.Require().True(rewardIds["reward-2"])
	suite.Require().True(rewardIds["reward-3"])
}

func (suite *IntegrationTestSuite) TestStoreStakingReward_IterateAllEmpty() {
	callCount := 0
	suite.k.IterateAllStakingRewards(suite.ctx, func(ctx sdk.Context, sr types.StakingReward) bool {
		callCount++
		return false
	})

	suite.Require().Equal(0, callCount)
}

func (suite *IntegrationTestSuite) TestStoreStakingReward_IterateAllMultiple() {
	rewards := []types.StakingReward{
		{
			RewardId:         "iter-reward-1",
			PrizeAmount:      math.NewInt(100),
			PrizeDenom:       "ubze",
			StakingDenom:     "ubze",
			Duration:         30,
			Payouts:          5,
			MinStake:         10,
			Lock:             1,
			StakedAmount:     math.NewInt(50),
			DistributedStake: math.LegacyMustNewDecFromStr("25"),
		},
		{
			RewardId:         "iter-reward-2",
			PrizeAmount:      math.NewInt(200),
			PrizeDenom:       "utoken",
			StakingDenom:     "ustake",
			Duration:         60,
			Payouts:          10,
			MinStake:         20,
			Lock:             2,
			StakedAmount:     math.NewInt(100),
			DistributedStake: math.LegacyMustNewDecFromStr("50"),
		},
	}

	// Set rewards
	for _, reward := range rewards {
		suite.k.SetStakingReward(suite.ctx, reward)
	}

	// Iterate and collect all rewards
	var iteratedRewards []types.StakingReward
	suite.k.IterateAllStakingRewards(suite.ctx, func(ctx sdk.Context, sr types.StakingReward) bool {
		iteratedRewards = append(iteratedRewards, sr)
		return false // Don't stop iteration
	})

	suite.Require().Len(iteratedRewards, 2)

	// Verify rewards are present
	rewardIds := make(map[string]bool)
	for _, reward := range iteratedRewards {
		rewardIds[reward.RewardId] = true
	}

	suite.Require().True(rewardIds["iter-reward-1"])
	suite.Require().True(rewardIds["iter-reward-2"])
}

// --- GetBatchStakingRewards tests ---

func (suite *IntegrationTestSuite) TestStoreStakingReward_GetBatchFromBeginning() {
	for i := 1; i <= 5; i++ {
		suite.k.SetStakingReward(suite.ctx, types.StakingReward{
			RewardId:         fmt.Sprintf("batch-reward-%d", i),
			PrizeAmount:      math.NewInt(1000),
			PrizeDenom:       "ubze",
			StakingDenom:     "ubze",
			Duration:         5,
			Payouts:          0,
			MinStake:         100,
			Lock:             7,
			StakedAmount:     math.NewInt(5000),
			DistributedStake: math.LegacyZeroDec(),
		})
	}

	// Get first 3 from the beginning (empty cursor)
	batch := suite.k.GetBatchStakingRewards(suite.ctx, "", 3)
	suite.Require().Len(batch, 3)
}

func (suite *IntegrationTestSuite) TestStoreStakingReward_GetBatchWithCursor() {
	for i := 1; i <= 5; i++ {
		suite.k.SetStakingReward(suite.ctx, types.StakingReward{
			RewardId:         fmt.Sprintf("cursor-reward-%d", i),
			PrizeAmount:      math.NewInt(1000),
			PrizeDenom:       "ubze",
			StakingDenom:     "ubze",
			Duration:         5,
			Payouts:          0,
			MinStake:         100,
			Lock:             7,
			StakedAmount:     math.NewInt(5000),
			DistributedStake: math.LegacyZeroDec(),
		})
	}

	// Get first batch
	batch1 := suite.k.GetBatchStakingRewards(suite.ctx, "", 2)
	suite.Require().Len(batch1, 2)

	// Get second batch using cursor from first batch
	cursor := batch1[len(batch1)-1].RewardId
	batch2 := suite.k.GetBatchStakingRewards(suite.ctx, cursor, 2)
	suite.Require().Len(batch2, 2)

	// Verify no overlap between batches
	for _, r1 := range batch1 {
		for _, r2 := range batch2 {
			suite.Require().NotEqual(r1.RewardId, r2.RewardId)
		}
	}

	// Get third batch - should return remaining 1
	cursor2 := batch2[len(batch2)-1].RewardId
	batch3 := suite.k.GetBatchStakingRewards(suite.ctx, cursor2, 10)
	suite.Require().Len(batch3, 1)

	// All 5 rewards should be covered across batches
	allIds := make(map[string]bool)
	for _, r := range batch1 {
		allIds[r.RewardId] = true
	}
	for _, r := range batch2 {
		allIds[r.RewardId] = true
	}
	for _, r := range batch3 {
		allIds[r.RewardId] = true
	}
	suite.Require().Len(allIds, 5)
}

func (suite *IntegrationTestSuite) TestStoreStakingReward_GetBatchCursorAtEnd() {
	suite.k.SetStakingReward(suite.ctx, types.StakingReward{
		RewardId:         "last-reward",
		PrizeAmount:      math.NewInt(1000),
		PrizeDenom:       "ubze",
		StakingDenom:     "ubze",
		Duration:         5,
		Payouts:          0,
		MinStake:         100,
		Lock:             7,
		StakedAmount:     math.NewInt(5000),
		DistributedStake: math.LegacyZeroDec(),
	})

	// Cursor points to the last (and only) reward - nothing after it
	batch := suite.k.GetBatchStakingRewards(suite.ctx, "last-reward", 10)
	suite.Require().Empty(batch)
}

func (suite *IntegrationTestSuite) TestStoreStakingReward_GetBatchEmptyStore() {
	batch := suite.k.GetBatchStakingRewards(suite.ctx, "", 10)
	suite.Require().Empty(batch)
}

func (suite *IntegrationTestSuite) TestStoreStakingReward_GetBatchLimitHigherThanEntries() {
	for i := 1; i <= 3; i++ {
		suite.k.SetStakingReward(suite.ctx, types.StakingReward{
			RewardId:         fmt.Sprintf("limit-reward-%d", i),
			PrizeAmount:      math.NewInt(1000),
			PrizeDenom:       "ubze",
			StakingDenom:     "ubze",
			Duration:         5,
			Payouts:          0,
			MinStake:         100,
			Lock:             7,
			StakedAmount:     math.NewInt(5000),
			DistributedStake: math.LegacyZeroDec(),
		})
	}

	// Limit is higher than total entries
	batch := suite.k.GetBatchStakingRewards(suite.ctx, "", 100)
	suite.Require().Len(batch, 3)
}

func (suite *IntegrationTestSuite) TestStoreStakingReward_GetBatchCursorNotFound() {
	suite.k.SetStakingReward(suite.ctx, types.StakingReward{
		RewardId:         "existing-reward",
		PrizeAmount:      math.NewInt(1000),
		PrizeDenom:       "ubze",
		StakingDenom:     "ubze",
		Duration:         5,
		Payouts:          0,
		MinStake:         100,
		Lock:             7,
		StakedAmount:     math.NewInt(5000),
		DistributedStake: math.LegacyZeroDec(),
	})

	// Cursor points to a non-existent reward - should return nothing since cursor is never matched
	batch := suite.k.GetBatchStakingRewards(suite.ctx, "non-existent-cursor", 10)
	suite.Require().Empty(batch)
}

// --- StakingRewardsDistributionQueue store tests ---

func (suite *IntegrationTestSuite) TestStoreStakingReward_DistributionQueue_SetAndGet() {
	queue := types.StakingRewardsDistributionQueue{
		Pending: true,
		Cursor:  "reward-5",
	}
	suite.k.SetStakingRewardsDistributionQueue(suite.ctx, queue)

	result, found := suite.k.GetStakingRewardsDistributionQueue(suite.ctx)
	suite.Require().True(found)
	suite.Require().True(result.Pending)
	suite.Require().Equal("reward-5", result.Cursor)
}

func (suite *IntegrationTestSuite) TestStoreStakingReward_DistributionQueue_NotFound() {
	_, found := suite.k.GetStakingRewardsDistributionQueue(suite.ctx)
	suite.Require().False(found)
}

func (suite *IntegrationTestSuite) TestStoreStakingReward_DistributionQueue_Remove() {
	suite.k.SetStakingRewardsDistributionQueue(suite.ctx, types.StakingRewardsDistributionQueue{
		Pending: true,
		Cursor:  "some-cursor",
	})

	_, found := suite.k.GetStakingRewardsDistributionQueue(suite.ctx)
	suite.Require().True(found)

	suite.k.RemoveStakingRewardsDistributionQueue(suite.ctx)

	_, found = suite.k.GetStakingRewardsDistributionQueue(suite.ctx)
	suite.Require().False(found)
}

func (suite *IntegrationTestSuite) TestStoreStakingReward_DistributionQueue_Update() {
	suite.k.SetStakingRewardsDistributionQueue(suite.ctx, types.StakingRewardsDistributionQueue{
		Pending: true,
		Cursor:  "",
	})

	// Update the queue
	suite.k.SetStakingRewardsDistributionQueue(suite.ctx, types.StakingRewardsDistributionQueue{
		Pending: true,
		Cursor:  "reward-10",
	})

	result, found := suite.k.GetStakingRewardsDistributionQueue(suite.ctx)
	suite.Require().True(found)
	suite.Require().True(result.Pending)
	suite.Require().Equal("reward-10", result.Cursor)
}

func (suite *IntegrationTestSuite) TestStoreStakingReward_DistributionQueue_EmptyCursor() {
	suite.k.SetStakingRewardsDistributionQueue(suite.ctx, types.StakingRewardsDistributionQueue{
		Pending: true,
		Cursor:  "",
	})

	result, found := suite.k.GetStakingRewardsDistributionQueue(suite.ctx)
	suite.Require().True(found)
	suite.Require().True(result.Pending)
	suite.Require().Equal("", result.Cursor)
}

func (suite *IntegrationTestSuite) TestStoreStakingReward_DistributionQueue_NotPending() {
	suite.k.SetStakingRewardsDistributionQueue(suite.ctx, types.StakingRewardsDistributionQueue{
		Pending: false,
		Cursor:  "",
	})

	result, found := suite.k.GetStakingRewardsDistributionQueue(suite.ctx)
	suite.Require().True(found)
	suite.Require().False(result.Pending)
}

func (suite *IntegrationTestSuite) TestStoreStakingReward_DistributionQueue_RemoveNonExistent() {
	// Removing non-existent queue should not panic
	suite.Require().NotPanics(func() {
		suite.k.RemoveStakingRewardsDistributionQueue(suite.ctx)
	})
}

func (suite *IntegrationTestSuite) TestStoreStakingReward_IterateAllEarlyStop() {
	rewards := []types.StakingReward{
		{
			RewardId:         "stop-reward-1",
			PrizeAmount:      math.NewInt(100),
			PrizeDenom:       "ubze",
			StakingDenom:     "ubze",
			Duration:         30,
			Payouts:          5,
			MinStake:         10,
			Lock:             1,
			StakedAmount:     math.NewInt(50),
			DistributedStake: math.LegacyMustNewDecFromStr("25"),
		},
		{
			RewardId:         "stop-reward-2",
			PrizeAmount:      math.NewInt(200),
			PrizeDenom:       "utoken",
			StakingDenom:     "ustake",
			Duration:         60,
			Payouts:          10,
			MinStake:         20,
			Lock:             2,
			StakedAmount:     math.NewInt(100),
			DistributedStake: math.LegacyMustNewDecFromStr("50"),
		},
		{
			RewardId:         "stop-reward-3",
			PrizeAmount:      math.NewInt(300),
			PrizeDenom:       "ucoin",
			StakingDenom:     "ucoin",
			Duration:         90,
			Payouts:          15,
			MinStake:         30,
			Lock:             3,
			StakedAmount:     math.NewInt(150),
			DistributedStake: math.LegacyMustNewDecFromStr("75"),
		},
	}

	// Set all rewards
	for _, reward := range rewards {
		suite.k.SetStakingReward(suite.ctx, reward)
	}

	// Iterate but stop after first item
	callCount := 0
	suite.k.IterateAllStakingRewards(suite.ctx, func(ctx sdk.Context, sr types.StakingReward) bool {
		callCount++
		return true // Stop after first iteration
	})

	suite.Require().Equal(1, callCount)
}
