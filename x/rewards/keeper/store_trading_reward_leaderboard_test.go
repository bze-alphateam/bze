package keeper_test

import (
	"cosmossdk.io/math"
	"github.com/bze-alphateam/bze/x/rewards/types"
)

// Test Trading Reward Leaderboard
func (suite *IntegrationTestSuite) TestStoreTradingRewardLeaderboard_SetAndGet() {
	leaderboard := types.TradingRewardLeaderboard{
		RewardId: "leaderboard-reward-1",
		List: []types.TradingRewardLeaderboardEntry{
			{
				Amount:    math.NewInt(1000),
				Address:   "bze1first",
				CreatedAt: 1704067200,
			},
			{
				Amount:    math.NewInt(750),
				Address:   "bze1second",
				CreatedAt: 1704067300,
			},
			{
				Amount:    math.NewInt(500),
				Address:   "bze1third",
				CreatedAt: 1704067400,
			},
		},
	}

	// Test SetTradingRewardLeaderboard
	suite.k.SetTradingRewardLeaderboard(suite.ctx, leaderboard)

	// Test GetTradingRewardLeaderboard
	retrievedLeaderboard, found := suite.k.GetTradingRewardLeaderboard(suite.ctx, "leaderboard-reward-1")
	suite.Require().True(found)
	suite.Require().Equal(leaderboard.RewardId, retrievedLeaderboard.RewardId)
	suite.Require().Len(retrievedLeaderboard.List, 3)

	// Verify leaderboard entries
	suite.Require().Equal(leaderboard.List[0].Amount, retrievedLeaderboard.List[0].Amount)
	suite.Require().Equal(leaderboard.List[0].Address, retrievedLeaderboard.List[0].Address)
	suite.Require().Equal(leaderboard.List[0].CreatedAt, retrievedLeaderboard.List[0].CreatedAt)

	suite.Require().Equal(leaderboard.List[1].Amount, retrievedLeaderboard.List[1].Amount)
	suite.Require().Equal(leaderboard.List[1].Address, retrievedLeaderboard.List[1].Address)

	suite.Require().Equal(leaderboard.List[2].Amount, retrievedLeaderboard.List[2].Amount)
	suite.Require().Equal(leaderboard.List[2].Address, retrievedLeaderboard.List[2].Address)
}

func (suite *IntegrationTestSuite) TestStoreTradingRewardLeaderboard_GetNonExistent() {
	// Test getting non-existent leaderboard
	_, found := suite.k.GetTradingRewardLeaderboard(suite.ctx, "non-existent-leaderboard")
	suite.Require().False(found)
}

func (suite *IntegrationTestSuite) TestStoreTradingRewardLeaderboard_EmptyList() {
	leaderboard := types.TradingRewardLeaderboard{
		RewardId: "empty-leaderboard",
		List:     []types.TradingRewardLeaderboardEntry{},
	}

	// Test setting leaderboard with empty list
	suite.k.SetTradingRewardLeaderboard(suite.ctx, leaderboard)

	// Test retrieving empty leaderboard
	retrievedLeaderboard, found := suite.k.GetTradingRewardLeaderboard(suite.ctx, "empty-leaderboard")
	suite.Require().True(found)
	suite.Require().Equal(leaderboard.RewardId, retrievedLeaderboard.RewardId)
	suite.Require().Empty(retrievedLeaderboard.List)
}

func (suite *IntegrationTestSuite) TestStoreTradingRewardLeaderboard_SetMultiple() {
	leaderboard1 := types.TradingRewardLeaderboard{
		RewardId: "leaderboard-1",
		List: []types.TradingRewardLeaderboardEntry{
			{
				Amount:    math.NewInt(100),
				Address:   "bze1user1",
				CreatedAt: 1704067200,
			},
		},
	}

	leaderboard2 := types.TradingRewardLeaderboard{
		RewardId: "leaderboard-2",
		List: []types.TradingRewardLeaderboardEntry{
			{
				Amount:    math.NewInt(200),
				Address:   "bze1user2",
				CreatedAt: 1704067300,
			},
			{
				Amount:    math.NewInt(150),
				Address:   "bze1user3",
				CreatedAt: 1704067400,
			},
		},
	}

	// Set both leaderboards
	suite.k.SetTradingRewardLeaderboard(suite.ctx, leaderboard1)
	suite.k.SetTradingRewardLeaderboard(suite.ctx, leaderboard2)

	// Verify both can be retrieved independently
	retrieved1, found1 := suite.k.GetTradingRewardLeaderboard(suite.ctx, "leaderboard-1")
	suite.Require().True(found1)
	suite.Require().Equal(leaderboard1.RewardId, retrieved1.RewardId)
	suite.Require().Len(retrieved1.List, 1)

	retrieved2, found2 := suite.k.GetTradingRewardLeaderboard(suite.ctx, "leaderboard-2")
	suite.Require().True(found2)
	suite.Require().Equal(leaderboard2.RewardId, retrieved2.RewardId)
	suite.Require().Len(retrieved2.List, 2)
}

func (suite *IntegrationTestSuite) TestStoreTradingRewardLeaderboard_UpdateExisting() {
	originalLeaderboard := types.TradingRewardLeaderboard{
		RewardId: "update-leaderboard",
		List: []types.TradingRewardLeaderboardEntry{
			{
				Amount:    math.NewInt(100),
				Address:   "bze1original",
				CreatedAt: 1704067200,
			},
		},
	}

	updatedLeaderboard := types.TradingRewardLeaderboard{
		RewardId: "update-leaderboard",
		List: []types.TradingRewardLeaderboardEntry{
			{
				Amount:    math.NewInt(200),
				Address:   "bze1updated",
				CreatedAt: 1704067300,
			},
			{
				Amount:    math.NewInt(150),
				Address:   "bze1new",
				CreatedAt: 1704067400,
			},
		},
	}

	// Set original leaderboard
	suite.k.SetTradingRewardLeaderboard(suite.ctx, originalLeaderboard)

	// Update the leaderboard
	suite.k.SetTradingRewardLeaderboard(suite.ctx, updatedLeaderboard)

	// Verify the leaderboard was updated
	retrieved, found := suite.k.GetTradingRewardLeaderboard(suite.ctx, "update-leaderboard")
	suite.Require().True(found)
	suite.Require().Len(retrieved.List, 2)
	suite.Require().Equal(math.NewInt(200), retrieved.List[0].Amount)
	suite.Require().Equal("bze1updated", retrieved.List[0].Address)
}

func (suite *IntegrationTestSuite) TestStoreTradingRewardLeaderboard_Remove() {
	leaderboard := types.TradingRewardLeaderboard{
		RewardId: "leaderboard-to-remove",
		List: []types.TradingRewardLeaderboardEntry{
			{
				Amount:    math.NewInt(500),
				Address:   "bze1remove",
				CreatedAt: 1704067200,
			},
		},
	}

	// Set the leaderboard
	suite.k.SetTradingRewardLeaderboard(suite.ctx, leaderboard)

	// Verify it exists
	_, found := suite.k.GetTradingRewardLeaderboard(suite.ctx, "leaderboard-to-remove")
	suite.Require().True(found)

	// Remove the leaderboard
	suite.k.RemoveTradingRewardLeaderboard(suite.ctx, "leaderboard-to-remove")

	// Verify it no longer exists
	_, found = suite.k.GetTradingRewardLeaderboard(suite.ctx, "leaderboard-to-remove")
	suite.Require().False(found)
}

func (suite *IntegrationTestSuite) TestStoreTradingRewardLeaderboard_RemoveNonExistent() {
	// Removing non-existent leaderboard should not cause issues
	suite.Require().NotPanics(func() {
		suite.k.RemoveTradingRewardLeaderboard(suite.ctx, "non-existent-leaderboard")
	})
}

func (suite *IntegrationTestSuite) TestStoreTradingRewardLeaderboard_GetAllEmpty() {
	// Test GetAllTradingRewardLeaderboard when no leaderboards exist
	allLeaderboards := suite.k.GetAllTradingRewardLeaderboard(suite.ctx)
	suite.Require().Empty(allLeaderboards)
}

func (suite *IntegrationTestSuite) TestStoreTradingRewardLeaderboard_GetAllMultiple() {
	leaderboards := []types.TradingRewardLeaderboard{
		{
			RewardId: "leaderboard-all-1",
			List: []types.TradingRewardLeaderboardEntry{
				{
					Amount:    math.NewInt(300),
					Address:   "bze1leader1",
					CreatedAt: 1704067200,
				},
			},
		},
		{
			RewardId: "leaderboard-all-2",
			List: []types.TradingRewardLeaderboardEntry{
				{
					Amount:    math.NewInt(400),
					Address:   "bze1leader2",
					CreatedAt: 1704067300,
				},
			},
		},
		{
			RewardId: "leaderboard-all-3",
			List: []types.TradingRewardLeaderboardEntry{
				{
					Amount:    math.NewInt(500),
					Address:   "bze1leader3",
					CreatedAt: 1704067400,
				},
			},
		},
	}

	// Set all leaderboards
	for _, leaderboard := range leaderboards {
		suite.k.SetTradingRewardLeaderboard(suite.ctx, leaderboard)
	}

	// Get all leaderboards
	allLeaderboards := suite.k.GetAllTradingRewardLeaderboard(suite.ctx)
	suite.Require().Len(allLeaderboards, 3)

	// Verify all leaderboards are present
	rewardIds := make(map[string]bool)
	for _, leaderboard := range allLeaderboards {
		rewardIds[leaderboard.RewardId] = true
	}

	suite.Require().True(rewardIds["leaderboard-all-1"])
	suite.Require().True(rewardIds["leaderboard-all-2"])
	suite.Require().True(rewardIds["leaderboard-all-3"])
}

// Test Trading Reward Candidate
func (suite *IntegrationTestSuite) TestStoreTradingRewardLeaderboard_SetAndGetCandidate() {
	candidate := types.TradingRewardCandidate{
		RewardId: "candidate-reward-1",
		Amount:   math.NewInt(750),
		Address:  "bze1candidate",
	}

	// Test SetTradingRewardCandidate
	suite.k.SetTradingRewardCandidate(suite.ctx, candidate)

	// Test GetTradingRewardCandidate
	retrievedCandidate, found := suite.k.GetTradingRewardCandidate(suite.ctx, "candidate-reward-1", "bze1candidate")
	suite.Require().True(found)
	suite.Require().Equal(candidate.RewardId, retrievedCandidate.RewardId)
	suite.Require().Equal(candidate.Amount, retrievedCandidate.Amount)
	suite.Require().Equal(candidate.Address, retrievedCandidate.Address)
}

func (suite *IntegrationTestSuite) TestStoreTradingRewardLeaderboard_GetCandidateNonExistent() {
	// Test getting non-existent candidate
	_, found := suite.k.GetTradingRewardCandidate(suite.ctx, "non-existent-reward", "bze1nonexistent")
	suite.Require().False(found)

	// Test with existing reward but non-existent address
	candidate := types.TradingRewardCandidate{
		RewardId: "existing-reward",
		Amount:   math.NewInt(100),
		Address:  "bze1existing",
	}
	suite.k.SetTradingRewardCandidate(suite.ctx, candidate)

	_, found = suite.k.GetTradingRewardCandidate(suite.ctx, "existing-reward", "bze1nonexistent")
	suite.Require().False(found)
}

func (suite *IntegrationTestSuite) TestStoreTradingRewardLeaderboard_SetMultipleCandidates() {
	candidates := []types.TradingRewardCandidate{
		{
			RewardId: "candidate-reward-1",
			Amount:   math.NewInt(500),
			Address:  "bze1candidate1",
		},
		{
			RewardId: "candidate-reward-1",
			Amount:   math.NewInt(750),
			Address:  "bze1candidate2",
		},
		{
			RewardId: "candidate-reward-2",
			Amount:   math.NewInt(300),
			Address:  "bze1candidate1",
		},
	}

	// Set multiple candidates
	for _, candidate := range candidates {
		suite.k.SetTradingRewardCandidate(suite.ctx, candidate)
	}

	// Verify each can be retrieved independently
	retrieved1, found1 := suite.k.GetTradingRewardCandidate(suite.ctx, "candidate-reward-1", "bze1candidate1")
	suite.Require().True(found1)
	suite.Require().Equal(math.NewInt(500), retrieved1.Amount)

	retrieved2, found2 := suite.k.GetTradingRewardCandidate(suite.ctx, "candidate-reward-1", "bze1candidate2")
	suite.Require().True(found2)
	suite.Require().Equal(math.NewInt(750), retrieved2.Amount)

	retrieved3, found3 := suite.k.GetTradingRewardCandidate(suite.ctx, "candidate-reward-2", "bze1candidate1")
	suite.Require().True(found3)
	suite.Require().Equal(math.NewInt(300), retrieved3.Amount)
}

func (suite *IntegrationTestSuite) TestStoreTradingRewardLeaderboard_UpdateCandidate() {
	originalCandidate := types.TradingRewardCandidate{
		RewardId: "update-candidate-reward",
		Amount:   math.NewInt(100),
		Address:  "bze1update",
	}

	updatedCandidate := types.TradingRewardCandidate{
		RewardId: "update-candidate-reward",
		Amount:   math.NewInt(250),
		Address:  "bze1update",
	}

	// Set original candidate
	suite.k.SetTradingRewardCandidate(suite.ctx, originalCandidate)

	// Update the candidate
	suite.k.SetTradingRewardCandidate(suite.ctx, updatedCandidate)

	// Verify the candidate was updated
	retrieved, found := suite.k.GetTradingRewardCandidate(suite.ctx, "update-candidate-reward", "bze1update")
	suite.Require().True(found)
	suite.Require().Equal(math.NewInt(250), retrieved.Amount)
}

func (suite *IntegrationTestSuite) TestStoreTradingRewardLeaderboard_RemoveCandidate() {
	candidate := types.TradingRewardCandidate{
		RewardId: "candidate-to-remove",
		Amount:   math.NewInt(600),
		Address:  "bze1remove",
	}

	// Set the candidate
	suite.k.SetTradingRewardCandidate(suite.ctx, candidate)

	// Verify it exists
	_, found := suite.k.GetTradingRewardCandidate(suite.ctx, "candidate-to-remove", "bze1remove")
	suite.Require().True(found)

	// Remove the candidate
	suite.k.RemoveTradingRewardCandidate(suite.ctx, "candidate-to-remove", "bze1remove")

	// Verify it no longer exists
	_, found = suite.k.GetTradingRewardCandidate(suite.ctx, "candidate-to-remove", "bze1remove")
	suite.Require().False(found)
}

func (suite *IntegrationTestSuite) TestStoreTradingRewardLeaderboard_RemoveCandidateSpecific() {
	candidate1 := types.TradingRewardCandidate{
		RewardId: "reward-keep",
		Amount:   math.NewInt(100),
		Address:  "bze1keep",
	}

	candidate2 := types.TradingRewardCandidate{
		RewardId: "reward-keep",
		Amount:   math.NewInt(200),
		Address:  "bze1remove",
	}

	// Set both candidates
	suite.k.SetTradingRewardCandidate(suite.ctx, candidate1)
	suite.k.SetTradingRewardCandidate(suite.ctx, candidate2)

	// Remove only one candidate
	suite.k.RemoveTradingRewardCandidate(suite.ctx, "reward-keep", "bze1remove")

	// Verify only the correct one was removed
	_, found1 := suite.k.GetTradingRewardCandidate(suite.ctx, "reward-keep", "bze1keep")
	suite.Require().True(found1)

	_, found2 := suite.k.GetTradingRewardCandidate(suite.ctx, "reward-keep", "bze1remove")
	suite.Require().False(found2)
}

func (suite *IntegrationTestSuite) TestStoreTradingRewardLeaderboard_GetAllCandidatesEmpty() {
	// Test GetAllTradingRewardCandidate when no candidates exist
	allCandidates := suite.k.GetAllTradingRewardCandidate(suite.ctx)
	suite.Require().Empty(allCandidates)
}

func (suite *IntegrationTestSuite) TestStoreTradingRewardLeaderboard_GetAllCandidatesMultiple() {
	candidates := []types.TradingRewardCandidate{
		{
			RewardId: "all-candidate-reward-1",
			Amount:   math.NewInt(100),
			Address:  "bze1candidate1",
		},
		{
			RewardId: "all-candidate-reward-1",
			Amount:   math.NewInt(150),
			Address:  "bze1candidate2",
		},
		{
			RewardId: "all-candidate-reward-2",
			Amount:   math.NewInt(200),
			Address:  "bze1candidate3",
		},
	}

	// Set all candidates
	for _, candidate := range candidates {
		suite.k.SetTradingRewardCandidate(suite.ctx, candidate)
	}

	// Get all candidates
	allCandidates := suite.k.GetAllTradingRewardCandidate(suite.ctx)
	suite.Require().Len(allCandidates, 3)

	// Verify all candidates are present
	candidateKeys := make(map[string]bool)
	for _, candidate := range allCandidates {
		key := candidate.RewardId + "-" + candidate.Address
		candidateKeys[key] = true
	}

	suite.Require().True(candidateKeys["all-candidate-reward-1-bze1candidate1"])
	suite.Require().True(candidateKeys["all-candidate-reward-1-bze1candidate2"])
	suite.Require().True(candidateKeys["all-candidate-reward-2-bze1candidate3"])
}

func (suite *IntegrationTestSuite) TestStoreTradingRewardLeaderboard_GetCandidatesByRewardId() {
	candidates := []types.TradingRewardCandidate{
		{
			RewardId: "specific-reward",
			Amount:   math.NewInt(100),
			Address:  "bze1candidate1",
		},
		{
			RewardId: "specific-reward",
			Amount:   math.NewInt(150),
			Address:  "bze1candidate2",
		},
		{
			RewardId: "specific-reward",
			Amount:   math.NewInt(200),
			Address:  "bze1candidate3",
		},
		{
			RewardId: "different-reward",
			Amount:   math.NewInt(300),
			Address:  "bze1candidate4",
		},
	}

	// Set all candidates
	for _, candidate := range candidates {
		suite.k.SetTradingRewardCandidate(suite.ctx, candidate)
	}

	// Get candidates by specific reward ID
	specificCandidates := suite.k.GetTradingRewardCandidateByRewardId(suite.ctx, "specific-reward")
	suite.Require().Len(specificCandidates, 3)

	// Verify correct candidates are returned
	addresses := make(map[string]bool)
	for _, candidate := range specificCandidates {
		suite.Require().Equal("specific-reward", candidate.RewardId)
		addresses[candidate.Address] = true
	}

	suite.Require().True(addresses["bze1candidate1"])
	suite.Require().True(addresses["bze1candidate2"])
	suite.Require().True(addresses["bze1candidate3"])
	suite.Require().False(addresses["bze1candidate4"]) // Should not be present

	// Get candidates for different reward ID
	differentCandidates := suite.k.GetTradingRewardCandidateByRewardId(suite.ctx, "different-reward")
	suite.Require().Len(differentCandidates, 1)
	suite.Require().Equal("bze1candidate4", differentCandidates[0].Address)
}

func (suite *IntegrationTestSuite) TestStoreTradingRewardLeaderboard_GetCandidatesByRewardIdEmpty() {
	// Test getting candidates for non-existent reward ID
	candidates := suite.k.GetTradingRewardCandidateByRewardId(suite.ctx, "non-existent-reward")
	suite.Require().Empty(candidates)
}

func (suite *IntegrationTestSuite) TestStoreTradingRewardLeaderboard_CompositeKey() {
	// Test that the composite key (rewardId + address) works correctly
	candidate1 := types.TradingRewardCandidate{
		RewardId: "same-reward",
		Amount:   math.NewInt(100),
		Address:  "bze1same",
	}

	candidate2 := types.TradingRewardCandidate{
		RewardId: "same-reward",
		Amount:   math.NewInt(200),
		Address:  "bze1different",
	}

	candidate3 := types.TradingRewardCandidate{
		RewardId: "different-reward",
		Amount:   math.NewInt(300),
		Address:  "bze1same",
	}

	// Set all candidates
	suite.k.SetTradingRewardCandidate(suite.ctx, candidate1)
	suite.k.SetTradingRewardCandidate(suite.ctx, candidate2)
	suite.k.SetTradingRewardCandidate(suite.ctx, candidate3)

	// Verify each can be retrieved with correct composite key
	retrieved1, found1 := suite.k.GetTradingRewardCandidate(suite.ctx, "same-reward", "bze1same")
	suite.Require().True(found1)
	suite.Require().Equal(math.NewInt(100), retrieved1.Amount)

	retrieved2, found2 := suite.k.GetTradingRewardCandidate(suite.ctx, "same-reward", "bze1different")
	suite.Require().True(found2)
	suite.Require().Equal(math.NewInt(200), retrieved2.Amount)

	retrieved3, found3 := suite.k.GetTradingRewardCandidate(suite.ctx, "different-reward", "bze1same")
	suite.Require().True(found3)
	suite.Require().Equal(math.NewInt(300), retrieved3.Amount)

	// Verify wrong combinations don't exist
	_, found4 := suite.k.GetTradingRewardCandidate(suite.ctx, "different-reward", "bze1different")
	suite.Require().False(found4)
}

func (suite *IntegrationTestSuite) TestStoreTradingRewardLeaderboard_RemoveCandidateNonExistent() {
	// Removing non-existent candidate should not cause issues
	suite.Require().NotPanics(func() {
		suite.k.RemoveTradingRewardCandidate(suite.ctx, "non-existent-reward", "bze1nonexistent")
	})
}
