package keeper_test

import (
	"github.com/bze-alphateam/bze/x/rewards/types"
)

// Test Pending Trading Rewards
func (suite *IntegrationTestSuite) TestStoreTradingReward_SetAndGetPending() {
	tradingReward := types.TradingReward{
		RewardId:    "pending-reward-1",
		PrizeAmount: "1000",
		PrizeDenom:  "ubze",
		Duration:    30,
		MarketId:    "market-1",
		Slots:       10,
		ExpireAt:    1704067200, // 2024-01-01 00:00:00 UTC
	}

	// Test SetPendingTradingReward
	suite.k.SetPendingTradingReward(suite.ctx, tradingReward)

	// Test GetPendingTradingReward
	retrievedReward, found := suite.k.GetPendingTradingReward(suite.ctx, "pending-reward-1")
	suite.Require().True(found)
	suite.Require().Equal(tradingReward.RewardId, retrievedReward.RewardId)
	suite.Require().Equal(tradingReward.PrizeAmount, retrievedReward.PrizeAmount)
	suite.Require().Equal(tradingReward.PrizeDenom, retrievedReward.PrizeDenom)
	suite.Require().Equal(tradingReward.Duration, retrievedReward.Duration)
	suite.Require().Equal(tradingReward.MarketId, retrievedReward.MarketId)
	suite.Require().Equal(tradingReward.Slots, retrievedReward.Slots)
	suite.Require().Equal(tradingReward.ExpireAt, retrievedReward.ExpireAt)
}

func (suite *IntegrationTestSuite) TestStoreTradingReward_GetPendingNonExistent() {
	// Test getting non-existent pending reward
	_, found := suite.k.GetPendingTradingReward(suite.ctx, "non-existent-pending")
	suite.Require().False(found)
}

func (suite *IntegrationTestSuite) TestStoreTradingReward_SetMultiplePending() {
	reward1 := types.TradingReward{
		RewardId:    "pending-1",
		PrizeAmount: "500",
		PrizeDenom:  "ubze",
		Duration:    15,
		MarketId:    "market-1",
		Slots:       5,
		ExpireAt:    1704067200,
	}

	reward2 := types.TradingReward{
		RewardId:    "pending-2",
		PrizeAmount: "1500",
		PrizeDenom:  "utoken",
		Duration:    45,
		MarketId:    "market-2",
		Slots:       15,
		ExpireAt:    1704153600,
	}

	// Set both pending rewards
	suite.k.SetPendingTradingReward(suite.ctx, reward1)
	suite.k.SetPendingTradingReward(suite.ctx, reward2)

	// Verify both can be retrieved independently
	retrieved1, found1 := suite.k.GetPendingTradingReward(suite.ctx, "pending-1")
	suite.Require().True(found1)
	suite.Require().Equal(reward1.RewardId, retrieved1.RewardId)
	suite.Require().Equal(reward1.MarketId, retrieved1.MarketId)

	retrieved2, found2 := suite.k.GetPendingTradingReward(suite.ctx, "pending-2")
	suite.Require().True(found2)
	suite.Require().Equal(reward2.RewardId, retrieved2.RewardId)
	suite.Require().Equal(reward2.PrizeAmount, retrieved2.PrizeAmount)
}

func (suite *IntegrationTestSuite) TestStoreTradingReward_RemovePending() {
	tradingReward := types.TradingReward{
		RewardId:    "pending-to-remove",
		PrizeAmount: "750",
		PrizeDenom:  "ubze",
		Duration:    20,
		MarketId:    "market-remove",
		Slots:       8,
		ExpireAt:    1704067200,
	}

	// Set the pending reward
	suite.k.SetPendingTradingReward(suite.ctx, tradingReward)

	// Verify it exists
	_, found := suite.k.GetPendingTradingReward(suite.ctx, "pending-to-remove")
	suite.Require().True(found)

	// Remove the pending reward
	suite.k.RemovePendingTradingReward(suite.ctx, "pending-to-remove")

	// Verify it no longer exists
	_, found = suite.k.GetPendingTradingReward(suite.ctx, "pending-to-remove")
	suite.Require().False(found)
}

func (suite *IntegrationTestSuite) TestStoreTradingReward_GetAllPendingEmpty() {
	// Test GetAllPendingTradingReward when no pending rewards exist
	allPending := suite.k.GetAllPendingTradingReward(suite.ctx)
	suite.Require().Empty(allPending)
}

func (suite *IntegrationTestSuite) TestStoreTradingReward_GetAllPendingMultiple() {
	rewards := []types.TradingReward{
		{
			RewardId:    "pending-all-1",
			PrizeAmount: "100",
			PrizeDenom:  "ubze",
			Duration:    10,
			MarketId:    "market-1",
			Slots:       5,
			ExpireAt:    1704067200,
		},
		{
			RewardId:    "pending-all-2",
			PrizeAmount: "200",
			PrizeDenom:  "utoken",
			Duration:    20,
			MarketId:    "market-2",
			Slots:       10,
			ExpireAt:    1704153600,
		},
		{
			RewardId:    "pending-all-3",
			PrizeAmount: "300",
			PrizeDenom:  "ucoin",
			Duration:    30,
			MarketId:    "market-3",
			Slots:       15,
			ExpireAt:    1704240000,
		},
	}

	// Set all pending rewards
	for _, reward := range rewards {
		suite.k.SetPendingTradingReward(suite.ctx, reward)
	}

	// Get all pending rewards
	allPending := suite.k.GetAllPendingTradingReward(suite.ctx)
	suite.Require().Len(allPending, 3)

	// Verify all rewards are present
	rewardIds := make(map[string]bool)
	for _, reward := range allPending {
		rewardIds[reward.RewardId] = true
	}

	suite.Require().True(rewardIds["pending-all-1"])
	suite.Require().True(rewardIds["pending-all-2"])
	suite.Require().True(rewardIds["pending-all-3"])
}

// Test Active Trading Rewards
func (suite *IntegrationTestSuite) TestStoreTradingReward_SetAndGetActive() {
	tradingReward := types.TradingReward{
		RewardId:    "active-reward-1",
		PrizeAmount: "2000",
		PrizeDenom:  "ubze",
		Duration:    60,
		MarketId:    "market-active",
		Slots:       20,
		ExpireAt:    1704067200,
	}

	// Test SetActiveTradingReward
	suite.k.SetActiveTradingReward(suite.ctx, tradingReward)

	// Test GetActiveTradingReward
	retrievedReward, found := suite.k.GetActiveTradingReward(suite.ctx, "active-reward-1")
	suite.Require().True(found)
	suite.Require().Equal(tradingReward.RewardId, retrievedReward.RewardId)
	suite.Require().Equal(tradingReward.PrizeAmount, retrievedReward.PrizeAmount)
	suite.Require().Equal(tradingReward.PrizeDenom, retrievedReward.PrizeDenom)
	suite.Require().Equal(tradingReward.Duration, retrievedReward.Duration)
	suite.Require().Equal(tradingReward.MarketId, retrievedReward.MarketId)
	suite.Require().Equal(tradingReward.Slots, retrievedReward.Slots)
	suite.Require().Equal(tradingReward.ExpireAt, retrievedReward.ExpireAt)
}

func (suite *IntegrationTestSuite) TestStoreTradingReward_GetActiveNonExistent() {
	// Test getting non-existent active reward
	_, found := suite.k.GetActiveTradingReward(suite.ctx, "non-existent-active")
	suite.Require().False(found)
}

func (suite *IntegrationTestSuite) TestStoreTradingReward_RemoveActive() {
	tradingReward := types.TradingReward{
		RewardId:    "active-to-remove",
		PrizeAmount: "1250",
		PrizeDenom:  "ubze",
		Duration:    40,
		MarketId:    "market-remove-active",
		Slots:       12,
		ExpireAt:    1704067200,
	}

	// Set the active reward
	suite.k.SetActiveTradingReward(suite.ctx, tradingReward)

	// Verify it exists
	_, found := suite.k.GetActiveTradingReward(suite.ctx, "active-to-remove")
	suite.Require().True(found)

	// Remove the active reward
	suite.k.RemoveActiveTradingReward(suite.ctx, "active-to-remove")

	// Verify it no longer exists
	_, found = suite.k.GetActiveTradingReward(suite.ctx, "active-to-remove")
	suite.Require().False(found)
}

func (suite *IntegrationTestSuite) TestStoreTradingReward_GetAllActiveEmpty() {
	// Test GetAllActiveTradingReward when no active rewards exist
	allActive := suite.k.GetAllActiveTradingReward(suite.ctx)
	suite.Require().Empty(allActive)
}

func (suite *IntegrationTestSuite) TestStoreTradingReward_GetAllActiveMultiple() {
	rewards := []types.TradingReward{
		{
			RewardId:    "active-all-1",
			PrizeAmount: "500",
			PrizeDenom:  "ubze",
			Duration:    25,
			MarketId:    "market-1",
			Slots:       8,
			ExpireAt:    1704067200,
		},
		{
			RewardId:    "active-all-2",
			PrizeAmount: "1000",
			PrizeDenom:  "utoken",
			Duration:    35,
			MarketId:    "market-2",
			Slots:       12,
			ExpireAt:    1704153600,
		},
	}

	// Set all active rewards
	for _, reward := range rewards {
		suite.k.SetActiveTradingReward(suite.ctx, reward)
	}

	// Get all active rewards
	allActive := suite.k.GetAllActiveTradingReward(suite.ctx)
	suite.Require().Len(allActive, 2)

	// Verify all rewards are present
	rewardIds := make(map[string]bool)
	for _, reward := range allActive {
		rewardIds[reward.RewardId] = true
	}

	suite.Require().True(rewardIds["active-all-1"])
	suite.Require().True(rewardIds["active-all-2"])
}

// Test Active and Pending Independence
func (suite *IntegrationTestSuite) TestStoreTradingReward_PendingActiveIndependence() {
	pendingReward := types.TradingReward{
		RewardId:    "independence-pending",
		PrizeAmount: "500",
		PrizeDenom:  "ubze",
		Duration:    30,
		MarketId:    "market-1",
		Slots:       10,
		ExpireAt:    1704067200,
	}

	activeReward := types.TradingReward{
		RewardId:    "independence-active",
		PrizeAmount: "1000",
		PrizeDenom:  "utoken",
		Duration:    60,
		MarketId:    "market-2",
		Slots:       20,
		ExpireAt:    1704153600,
	}

	// Set both rewards
	suite.k.SetPendingTradingReward(suite.ctx, pendingReward)
	suite.k.SetActiveTradingReward(suite.ctx, activeReward)

	// Verify pending exists but not in active
	_, pendingFound := suite.k.GetPendingTradingReward(suite.ctx, "independence-pending")
	_, pendingInActive := suite.k.GetActiveTradingReward(suite.ctx, "independence-pending")
	suite.Require().True(pendingFound)
	suite.Require().False(pendingInActive)

	// Verify active exists but not in pending
	_, activeFound := suite.k.GetActiveTradingReward(suite.ctx, "independence-active")
	_, activeInPending := suite.k.GetPendingTradingReward(suite.ctx, "independence-active")
	suite.Require().True(activeFound)
	suite.Require().False(activeInPending)
}

// Test Market ID to Reward ID Mapping
func (suite *IntegrationTestSuite) TestStoreTradingReward_SetAndGetMarketIdRewardId() {
	marketIdRewardId := types.MarketIdTradingRewardId{
		RewardId: "reward-for-market",
		MarketId: "market-123",
	}

	// Test SetMarketIdRewardId
	suite.k.SetMarketIdRewardId(suite.ctx, marketIdRewardId)

	// Test GetMarketIdRewardId
	retrieved, found := suite.k.GetMarketIdRewardId(suite.ctx, "market-123")
	suite.Require().True(found)
	suite.Require().Equal(marketIdRewardId.RewardId, retrieved.RewardId)
	suite.Require().Equal(marketIdRewardId.MarketId, retrieved.MarketId)
}

func (suite *IntegrationTestSuite) TestStoreTradingReward_GetMarketIdRewardIdNonExistent() {
	// Test getting non-existent market id mapping
	_, found := suite.k.GetMarketIdRewardId(suite.ctx, "non-existent-market")
	suite.Require().False(found)
}

func (suite *IntegrationTestSuite) TestStoreTradingReward_SetMultipleMarketIdRewardId() {
	mapping1 := types.MarketIdTradingRewardId{
		RewardId: "reward-1",
		MarketId: "market-1",
	}

	mapping2 := types.MarketIdTradingRewardId{
		RewardId: "reward-2",
		MarketId: "market-2",
	}

	// Set both mappings
	suite.k.SetMarketIdRewardId(suite.ctx, mapping1)
	suite.k.SetMarketIdRewardId(suite.ctx, mapping2)

	// Verify both can be retrieved independently
	retrieved1, found1 := suite.k.GetMarketIdRewardId(suite.ctx, "market-1")
	suite.Require().True(found1)
	suite.Require().Equal(mapping1.RewardId, retrieved1.RewardId)

	retrieved2, found2 := suite.k.GetMarketIdRewardId(suite.ctx, "market-2")
	suite.Require().True(found2)
	suite.Require().Equal(mapping2.RewardId, retrieved2.RewardId)
}

func (suite *IntegrationTestSuite) TestStoreTradingReward_UpdateMarketIdRewardId() {
	originalMapping := types.MarketIdTradingRewardId{
		RewardId: "original-reward",
		MarketId: "market-update",
	}

	updatedMapping := types.MarketIdTradingRewardId{
		RewardId: "updated-reward",
		MarketId: "market-update",
	}

	// Set original mapping
	suite.k.SetMarketIdRewardId(suite.ctx, originalMapping)

	// Update the mapping
	suite.k.SetMarketIdRewardId(suite.ctx, updatedMapping)

	// Verify the mapping was updated
	retrieved, found := suite.k.GetMarketIdRewardId(suite.ctx, "market-update")
	suite.Require().True(found)
	suite.Require().Equal(updatedMapping.RewardId, retrieved.RewardId)
}

func (suite *IntegrationTestSuite) TestStoreTradingReward_RemoveMarketIdRewardId() {
	marketIdRewardId := types.MarketIdTradingRewardId{
		RewardId: "reward-to-remove",
		MarketId: "market-to-remove",
	}

	// Set the mapping
	suite.k.SetMarketIdRewardId(suite.ctx, marketIdRewardId)

	// Verify it exists
	_, found := suite.k.GetMarketIdRewardId(suite.ctx, "market-to-remove")
	suite.Require().True(found)

	// Remove the mapping
	suite.k.RemoveMarketIdRewardId(suite.ctx, "market-to-remove")

	// Verify it no longer exists
	_, found = suite.k.GetMarketIdRewardId(suite.ctx, "market-to-remove")
	suite.Require().False(found)
}

func (suite *IntegrationTestSuite) TestStoreTradingReward_GetAllMarketIdRewardIdEmpty() {
	// Test GetAllMarketIdRewardId when no mappings exist
	allMappings := suite.k.GetAllMarketIdRewardId(suite.ctx)
	suite.Require().Empty(allMappings)
}

func (suite *IntegrationTestSuite) TestStoreTradingReward_GetAllMarketIdRewardIdMultiple() {
	mappings := []types.MarketIdTradingRewardId{
		{
			RewardId: "reward-1",
			MarketId: "market-1",
		},
		{
			RewardId: "reward-2",
			MarketId: "market-2",
		},
		{
			RewardId: "reward-3",
			MarketId: "market-3",
		},
	}

	// Set all mappings
	for _, mapping := range mappings {
		suite.k.SetMarketIdRewardId(suite.ctx, mapping)
	}

	// Get all mappings
	allMappings := suite.k.GetAllMarketIdRewardId(suite.ctx)
	suite.Require().Len(allMappings, 3)

	// Verify all mappings are present
	marketIds := make(map[string]bool)
	for _, mapping := range allMappings {
		marketIds[mapping.MarketId] = true
	}

	suite.Require().True(marketIds["market-1"])
	suite.Require().True(marketIds["market-2"])
	suite.Require().True(marketIds["market-3"])
}
