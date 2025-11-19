package keeper_test

import (
	"github.com/bze-alphateam/bze/x/rewards/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (suite *IntegrationTestSuite) TestStoreStakingReward_SetAndGet() {
	stakingReward := types.StakingReward{
		RewardId:         "reward-1",
		PrizeAmount:      "1000",
		PrizeDenom:       "ubze",
		StakingDenom:     "ubze",
		Duration:         30,
		Payouts:          10,
		MinStake:         100,
		Lock:             7,
		StakedAmount:     "0",
		DistributedStake: "0",
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
		PrizeAmount:      "1000",
		PrizeDenom:       "ubze",
		StakingDenom:     "ubze",
		Duration:         30,
		Payouts:          10,
		MinStake:         100,
		Lock:             7,
		StakedAmount:     "500",
		DistributedStake: "250",
	}

	reward2 := types.StakingReward{
		RewardId:         "reward-2",
		PrizeAmount:      "2000",
		PrizeDenom:       "utoken",
		StakingDenom:     "ustake",
		Duration:         60,
		Payouts:          20,
		MinStake:         200,
		Lock:             14,
		StakedAmount:     "1000",
		DistributedStake: "500",
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
		PrizeAmount:      "500",
		PrizeDenom:       "ubze",
		StakingDenom:     "ubze",
		Duration:         30,
		Payouts:          5,
		MinStake:         50,
		Lock:             3,
		StakedAmount:     "100",
		DistributedStake: "50",
	}

	updatedReward := types.StakingReward{
		RewardId:         "reward-update",
		PrizeAmount:      "1500",
		PrizeDenom:       "utoken",
		StakingDenom:     "ustake",
		Duration:         90,
		Payouts:          15,
		MinStake:         150,
		Lock:             10,
		StakedAmount:     "300",
		DistributedStake: "150",
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
		PrizeAmount:      "750",
		PrizeDenom:       "ubze",
		StakingDenom:     "ubze",
		Duration:         45,
		Payouts:          8,
		MinStake:         75,
		Lock:             5,
		StakedAmount:     "200",
		DistributedStake: "100",
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
			PrizeAmount:      "100",
			PrizeDenom:       "ubze",
			StakingDenom:     "ubze",
			Duration:         30,
			Payouts:          5,
			MinStake:         10,
			Lock:             1,
			StakedAmount:     "50",
			DistributedStake: "25",
		},
		{
			RewardId:         "reward-2",
			PrizeAmount:      "200",
			PrizeDenom:       "utoken",
			StakingDenom:     "ustake",
			Duration:         60,
			Payouts:          10,
			MinStake:         20,
			Lock:             2,
			StakedAmount:     "100",
			DistributedStake: "50",
		},
		{
			RewardId:         "reward-3",
			PrizeAmount:      "300",
			PrizeDenom:       "ucoin",
			StakingDenom:     "ucoin",
			Duration:         90,
			Payouts:          15,
			MinStake:         30,
			Lock:             3,
			StakedAmount:     "150",
			DistributedStake: "75",
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
			PrizeAmount:      "100",
			PrizeDenom:       "ubze",
			StakingDenom:     "ubze",
			Duration:         30,
			Payouts:          5,
			MinStake:         10,
			Lock:             1,
			StakedAmount:     "50",
			DistributedStake: "25",
		},
		{
			RewardId:         "iter-reward-2",
			PrizeAmount:      "200",
			PrizeDenom:       "utoken",
			StakingDenom:     "ustake",
			Duration:         60,
			Payouts:          10,
			MinStake:         20,
			Lock:             2,
			StakedAmount:     "100",
			DistributedStake: "50",
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

func (suite *IntegrationTestSuite) TestStoreStakingReward_IterateAllEarlyStop() {
	rewards := []types.StakingReward{
		{
			RewardId:         "stop-reward-1",
			PrizeAmount:      "100",
			PrizeDenom:       "ubze",
			StakingDenom:     "ubze",
			Duration:         30,
			Payouts:          5,
			MinStake:         10,
			Lock:             1,
			StakedAmount:     "50",
			DistributedStake: "25",
		},
		{
			RewardId:         "stop-reward-2",
			PrizeAmount:      "200",
			PrizeDenom:       "utoken",
			StakingDenom:     "ustake",
			Duration:         60,
			Payouts:          10,
			MinStake:         20,
			Lock:             2,
			StakedAmount:     "100",
			DistributedStake: "50",
		},
		{
			RewardId:         "stop-reward-3",
			PrizeAmount:      "300",
			PrizeDenom:       "ucoin",
			StakingDenom:     "ucoin",
			Duration:         90,
			Payouts:          15,
			MinStake:         30,
			Lock:             3,
			StakedAmount:     "150",
			DistributedStake: "75",
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

func (suite *IntegrationTestSuite) TestStoreStakingReward_CounterIncrement() {
	// Get initial counter
	initialCounter := suite.k.GetStakingRewardsCounter(suite.ctx)

	reward := types.StakingReward{
		RewardId:         "counter-test",
		PrizeAmount:      "500",
		PrizeDenom:       "ubze",
		StakingDenom:     "ubze",
		Duration:         30,
		Payouts:          5,
		MinStake:         50,
		Lock:             3,
		StakedAmount:     "100",
		DistributedStake: "50",
	}

	// Set reward should increment counter
	suite.k.SetStakingReward(suite.ctx, reward)

	// Verify counter was incremented
	newCounter := suite.k.GetStakingRewardsCounter(suite.ctx)
	suite.Require().Equal(initialCounter+1, newCounter)

	// Set another reward should increment again
	reward2 := types.StakingReward{
		RewardId:         "counter-test-2",
		PrizeAmount:      "600",
		PrizeDenom:       "utoken",
		StakingDenom:     "ustake",
		Duration:         45,
		Payouts:          8,
		MinStake:         60,
		Lock:             5,
		StakedAmount:     "120",
		DistributedStake: "60",
	}

	suite.k.SetStakingReward(suite.ctx, reward2)

	finalCounter := suite.k.GetStakingRewardsCounter(suite.ctx)
	suite.Require().Equal(initialCounter+2, finalCounter)
}
