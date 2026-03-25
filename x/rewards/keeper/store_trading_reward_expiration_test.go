package keeper_test

import (
	"fmt"
	"github.com/bze-alphateam/bze/x/rewards/types"
)

// Test Pending Trading Reward Expirations
func (suite *IntegrationTestSuite) TestStoreTradingRewardExpiration_SetAndGetPending() {
	expiration := types.TradingRewardExpiration{
		RewardId: "pending-exp-reward-1",
		ExpireAt: 1704067200, // 2024-01-01 00:00:00 UTC
	}

	// Test SetPendingTradingRewardExpiration
	suite.k.SetPendingTradingRewardExpiration(suite.ctx, expiration)

	// Test GetAllPendingTradingRewardExpiration to verify it was stored
	allExpirations := suite.k.GetAllPendingTradingRewardExpiration(suite.ctx)
	suite.Require().Len(allExpirations, 1)
	suite.Require().Equal(expiration.RewardId, allExpirations[0].RewardId)
	suite.Require().Equal(expiration.ExpireAt, allExpirations[0].ExpireAt)
}

func (suite *IntegrationTestSuite) TestStoreTradingRewardExpiration_SetMultiplePending() {
	expiration1 := types.TradingRewardExpiration{
		RewardId: "pending-exp-1",
		ExpireAt: 1704067200,
	}

	expiration2 := types.TradingRewardExpiration{
		RewardId: "pending-exp-2",
		ExpireAt: 1704153600, // 2024-01-02 00:00:00 UTC
	}

	expiration3 := types.TradingRewardExpiration{
		RewardId: "pending-exp-3",
		ExpireAt: 1704067200, // Same expiration as first
	}

	// Set multiple pending expirations
	suite.k.SetPendingTradingRewardExpiration(suite.ctx, expiration1)
	suite.k.SetPendingTradingRewardExpiration(suite.ctx, expiration2)
	suite.k.SetPendingTradingRewardExpiration(suite.ctx, expiration3)

	// Get all pending expirations
	allExpirations := suite.k.GetAllPendingTradingRewardExpiration(suite.ctx)
	suite.Require().Len(allExpirations, 3)

	// Verify all expirations are present
	rewardIds := make(map[string]bool)
	for _, exp := range allExpirations {
		rewardIds[exp.RewardId] = true
	}

	suite.Require().True(rewardIds["pending-exp-1"])
	suite.Require().True(rewardIds["pending-exp-2"])
	suite.Require().True(rewardIds["pending-exp-3"])
}

func (suite *IntegrationTestSuite) TestStoreTradingRewardExpiration_GetPendingByExpireAt() {
	expiration1 := types.TradingRewardExpiration{
		RewardId: "pending-exp-time-1",
		ExpireAt: 1704067200,
	}

	expiration2 := types.TradingRewardExpiration{
		RewardId: "pending-exp-time-2",
		ExpireAt: 1704067200, // Same expiration time
	}

	expiration3 := types.TradingRewardExpiration{
		RewardId: "pending-exp-time-3",
		ExpireAt: 1704153600, // Different expiration time
	}

	// Set all expirations
	suite.k.SetPendingTradingRewardExpiration(suite.ctx, expiration1)
	suite.k.SetPendingTradingRewardExpiration(suite.ctx, expiration2)
	suite.k.SetPendingTradingRewardExpiration(suite.ctx, expiration3)

	// Get expirations by specific expiration time
	expirationsAt1704067200 := suite.k.GetAllPendingTradingRewardExpirationByExpireAt(suite.ctx, 1704067200)
	suite.Require().Len(expirationsAt1704067200, 2)

	// Verify correct rewards are returned
	rewardIds := make(map[string]bool)
	for _, exp := range expirationsAt1704067200 {
		rewardIds[exp.RewardId] = true
		suite.Require().Equal(uint32(1704067200), exp.ExpireAt)
	}

	suite.Require().True(rewardIds["pending-exp-time-1"])
	suite.Require().True(rewardIds["pending-exp-time-2"])
	suite.Require().False(rewardIds["pending-exp-time-3"]) // Should not be present

	// Get expirations for different time
	expirationsAt1704153600 := suite.k.GetAllPendingTradingRewardExpirationByExpireAt(suite.ctx, 1704153600)
	suite.Require().Len(expirationsAt1704153600, 1)
	suite.Require().Equal("pending-exp-time-3", expirationsAt1704153600[0].RewardId)
}

func (suite *IntegrationTestSuite) TestStoreTradingRewardExpiration_GetPendingByExpireAtEmpty() {
	// Test getting expirations for non-existent expiration time
	expirations := suite.k.GetAllPendingTradingRewardExpirationByExpireAt(suite.ctx, uint32(4294967295))
	suite.Require().Empty(expirations)
}

func (suite *IntegrationTestSuite) TestStoreTradingRewardExpiration_RemovePending() {
	expiration := types.TradingRewardExpiration{
		RewardId: "pending-exp-to-remove",
		ExpireAt: 1704067200,
	}

	// Set the expiration
	suite.k.SetPendingTradingRewardExpiration(suite.ctx, expiration)

	// Verify it exists
	allExpirations := suite.k.GetAllPendingTradingRewardExpiration(suite.ctx)
	suite.Require().Len(allExpirations, 1)

	// Remove the expiration
	suite.k.RemovePendingTradingRewardExpiration(suite.ctx, 1704067200, "pending-exp-to-remove")

	// Verify it no longer exists
	allExpirations = suite.k.GetAllPendingTradingRewardExpiration(suite.ctx)
	suite.Require().Empty(allExpirations)
}

func (suite *IntegrationTestSuite) TestStoreTradingRewardExpiration_RemovePendingSpecific() {
	expiration1 := types.TradingRewardExpiration{
		RewardId: "pending-exp-keep",
		ExpireAt: 1704067200,
	}

	expiration2 := types.TradingRewardExpiration{
		RewardId: "pending-exp-remove",
		ExpireAt: 1704067200, // Same expiration time
	}

	// Set both expirations
	suite.k.SetPendingTradingRewardExpiration(suite.ctx, expiration1)
	suite.k.SetPendingTradingRewardExpiration(suite.ctx, expiration2)

	// Remove only one expiration
	suite.k.RemovePendingTradingRewardExpiration(suite.ctx, 1704067200, "pending-exp-remove")

	// Verify only the correct one was removed
	allExpirations := suite.k.GetAllPendingTradingRewardExpiration(suite.ctx)
	suite.Require().Len(allExpirations, 1)
	suite.Require().Equal("pending-exp-keep", allExpirations[0].RewardId)
}

func (suite *IntegrationTestSuite) TestStoreTradingRewardExpiration_GetAllPendingEmpty() {
	// Test GetAllPendingTradingRewardExpiration when no expirations exist
	allExpirations := suite.k.GetAllPendingTradingRewardExpiration(suite.ctx)
	suite.Require().Empty(allExpirations)
}

// Test Active Trading Reward Expirations
func (suite *IntegrationTestSuite) TestStoreTradingRewardExpiration_SetAndGetActive() {
	expiration := types.TradingRewardExpiration{
		RewardId: "active-exp-reward-1",
		ExpireAt: 1704067200,
	}

	// Test SetActiveTradingRewardExpiration
	suite.k.SetActiveTradingRewardExpiration(suite.ctx, expiration)

	// Test GetAllActiveTradingRewardExpiration to verify it was stored
	allExpirations := suite.k.GetAllActiveTradingRewardExpiration(suite.ctx)
	suite.Require().Len(allExpirations, 1)
	suite.Require().Equal(expiration.RewardId, allExpirations[0].RewardId)
	suite.Require().Equal(expiration.ExpireAt, allExpirations[0].ExpireAt)
}

func (suite *IntegrationTestSuite) TestStoreTradingRewardExpiration_SetMultipleActive() {
	expiration1 := types.TradingRewardExpiration{
		RewardId: "active-exp-1",
		ExpireAt: 1704067200,
	}

	expiration2 := types.TradingRewardExpiration{
		RewardId: "active-exp-2",
		ExpireAt: 1704153600,
	}

	// Set multiple active expirations
	suite.k.SetActiveTradingRewardExpiration(suite.ctx, expiration1)
	suite.k.SetActiveTradingRewardExpiration(suite.ctx, expiration2)

	// Get all active expirations
	allExpirations := suite.k.GetAllActiveTradingRewardExpiration(suite.ctx)
	suite.Require().Len(allExpirations, 2)

	// Verify all expirations are present
	rewardIds := make(map[string]bool)
	for _, exp := range allExpirations {
		rewardIds[exp.RewardId] = true
	}

	suite.Require().True(rewardIds["active-exp-1"])
	suite.Require().True(rewardIds["active-exp-2"])
}

func (suite *IntegrationTestSuite) TestStoreTradingRewardExpiration_GetActiveByExpireAt() {
	expiration1 := types.TradingRewardExpiration{
		RewardId: "active-exp-time-1",
		ExpireAt: 1704067200,
	}

	expiration2 := types.TradingRewardExpiration{
		RewardId: "active-exp-time-2",
		ExpireAt: 1704067200, // Same expiration time
	}

	expiration3 := types.TradingRewardExpiration{
		RewardId: "active-exp-time-3",
		ExpireAt: 1704153600, // Different expiration time
	}

	// Set all expirations
	suite.k.SetActiveTradingRewardExpiration(suite.ctx, expiration1)
	suite.k.SetActiveTradingRewardExpiration(suite.ctx, expiration2)
	suite.k.SetActiveTradingRewardExpiration(suite.ctx, expiration3)

	// Get expirations by specific expiration time
	expirationsAt1704067200 := suite.k.GetAllActiveTradingRewardExpirationByExpireAt(suite.ctx, 1704067200)
	suite.Require().Len(expirationsAt1704067200, 2)

	// Verify correct rewards are returned
	rewardIds := make(map[string]bool)
	for _, exp := range expirationsAt1704067200 {
		rewardIds[exp.RewardId] = true
		suite.Require().Equal(uint32(1704067200), exp.ExpireAt)
	}

	suite.Require().True(rewardIds["active-exp-time-1"])
	suite.Require().True(rewardIds["active-exp-time-2"])
	suite.Require().False(rewardIds["active-exp-time-3"]) // Should not be present
}

func (suite *IntegrationTestSuite) TestStoreTradingRewardExpiration_RemoveActive() {
	expiration := types.TradingRewardExpiration{
		RewardId: "active-exp-to-remove",
		ExpireAt: 1704067200,
	}

	// Set the expiration
	suite.k.SetActiveTradingRewardExpiration(suite.ctx, expiration)

	// Verify it exists
	allExpirations := suite.k.GetAllActiveTradingRewardExpiration(suite.ctx)
	suite.Require().Len(allExpirations, 1)

	// Remove the expiration
	suite.k.RemoveActiveTradingRewardExpiration(suite.ctx, 1704067200, "active-exp-to-remove")

	// Verify it no longer exists
	allExpirations = suite.k.GetAllActiveTradingRewardExpiration(suite.ctx)
	suite.Require().Empty(allExpirations)
}

func (suite *IntegrationTestSuite) TestStoreTradingRewardExpiration_GetAllActiveEmpty() {
	// Test GetAllActiveTradingRewardExpiration when no expirations exist
	allExpirations := suite.k.GetAllActiveTradingRewardExpiration(suite.ctx)
	suite.Require().Empty(allExpirations)
}

// Test Active and Pending Independence
func (suite *IntegrationTestSuite) TestStoreTradingRewardExpiration_PendingActiveIndependence() {
	pendingExpiration := types.TradingRewardExpiration{
		RewardId: "independence-pending-exp",
		ExpireAt: 1704067200,
	}

	activeExpiration := types.TradingRewardExpiration{
		RewardId: "independence-active-exp",
		ExpireAt: 1704067200, // Same expiration time
	}

	// Set both expirations
	suite.k.SetPendingTradingRewardExpiration(suite.ctx, pendingExpiration)
	suite.k.SetActiveTradingRewardExpiration(suite.ctx, activeExpiration)

	// Verify pending expiration exists only in pending store
	pendingExpirations := suite.k.GetAllPendingTradingRewardExpiration(suite.ctx)
	suite.Require().Len(pendingExpirations, 1)
	suite.Require().Equal("independence-pending-exp", pendingExpirations[0].RewardId)

	// Verify active expiration exists only in active store
	activeExpirations := suite.k.GetAllActiveTradingRewardExpiration(suite.ctx)
	suite.Require().Len(activeExpirations, 1)
	suite.Require().Equal("independence-active-exp", activeExpirations[0].RewardId)

	// Verify by expiration time queries are also independent
	pendingByTime := suite.k.GetAllPendingTradingRewardExpirationByExpireAt(suite.ctx, 1704067200)
	suite.Require().Len(pendingByTime, 1)
	suite.Require().Equal("independence-pending-exp", pendingByTime[0].RewardId)

	activeByTime := suite.k.GetAllActiveTradingRewardExpirationByExpireAt(suite.ctx, 1704067200)
	suite.Require().Len(activeByTime, 1)
	suite.Require().Equal("independence-active-exp", activeByTime[0].RewardId)
}

// Test Composite Key Functionality (ExpireAt + RewardId)
func (suite *IntegrationTestSuite) TestStoreTradingRewardExpiration_CompositeKey() {
	// Test that the composite key (expireAt + rewardId) works correctly
	expiration1 := types.TradingRewardExpiration{
		RewardId: "same-reward-id",
		ExpireAt: 1704067200,
	}

	expiration2 := types.TradingRewardExpiration{
		RewardId: "same-reward-id",
		ExpireAt: 1704153600, // Different expiration time, same reward ID
	}

	expiration3 := types.TradingRewardExpiration{
		RewardId: "different-reward-id",
		ExpireAt: 1704067200, // Same expiration time, different reward ID
	}

	// Set all expirations in pending store
	suite.k.SetPendingTradingRewardExpiration(suite.ctx, expiration1)
	suite.k.SetPendingTradingRewardExpiration(suite.ctx, expiration2)
	suite.k.SetPendingTradingRewardExpiration(suite.ctx, expiration3)

	// Verify all three exist independently
	allExpirations := suite.k.GetAllPendingTradingRewardExpiration(suite.ctx)
	suite.Require().Len(allExpirations, 3)

	// Verify querying by expiration time returns correct results
	expirationsAt1704067200 := suite.k.GetAllPendingTradingRewardExpirationByExpireAt(suite.ctx, 1704067200)
	suite.Require().Len(expirationsAt1704067200, 2) // expiration1 and expiration3

	expirationsAt1704153600 := suite.k.GetAllPendingTradingRewardExpirationByExpireAt(suite.ctx, 1704153600)
	suite.Require().Len(expirationsAt1704153600, 1) // expiration2

	// Remove one specific expiration and verify others remain
	suite.k.RemovePendingTradingRewardExpiration(suite.ctx, 1704067200, "same-reward-id")

	// Verify correct expiration was removed
	allExpirations = suite.k.GetAllPendingTradingRewardExpiration(suite.ctx)
	suite.Require().Len(allExpirations, 2)

	expirationsAt1704067200 = suite.k.GetAllPendingTradingRewardExpirationByExpireAt(suite.ctx, 1704067200)
	suite.Require().Len(expirationsAt1704067200, 1)
	suite.Require().Equal("different-reward-id", expirationsAt1704067200[0].RewardId)
}

// --- GetBatchPendingTradingRewardExpirationByExpireAt tests ---

func (suite *IntegrationTestSuite) TestStoreTradingRewardExpiration_GetBatchByExpireAt() {
	expireAt := uint32(500)

	for i := 1; i <= 5; i++ {
		suite.k.SetPendingTradingRewardExpiration(suite.ctx, types.TradingRewardExpiration{
			RewardId: fmt.Sprintf("batch-exp-reward-%d", i),
			ExpireAt: expireAt,
		})
	}

	// Limit to 2 - should return only 2
	batch := suite.k.GetBatchPendingTradingRewardExpirationByExpireAt(suite.ctx, expireAt, 2)
	suite.Require().Len(batch, 2)
	for _, exp := range batch {
		suite.Require().Equal(expireAt, exp.ExpireAt)
	}
}

func (suite *IntegrationTestSuite) TestStoreTradingRewardExpiration_GetBatchLimitHigherThanEntries() {
	expireAt := uint32(600)

	for i := 1; i <= 3; i++ {
		suite.k.SetPendingTradingRewardExpiration(suite.ctx, types.TradingRewardExpiration{
			RewardId: fmt.Sprintf("batch-limit-reward-%d", i),
			ExpireAt: expireAt,
		})
	}

	// Limit higher than available entries - should return all 3
	batch := suite.k.GetBatchPendingTradingRewardExpirationByExpireAt(suite.ctx, expireAt, 100)
	suite.Require().Len(batch, 3)
}

func (suite *IntegrationTestSuite) TestStoreTradingRewardExpiration_GetBatchEmptyEpoch() {
	// No expirations for this epoch
	batch := suite.k.GetBatchPendingTradingRewardExpirationByExpireAt(suite.ctx, 999, 10)
	suite.Require().Empty(batch)
}

func (suite *IntegrationTestSuite) TestStoreTradingRewardExpiration_GetBatchOnlyMatchesExpireAt() {
	// Set expirations for different expire_at values
	suite.k.SetPendingTradingRewardExpiration(suite.ctx, types.TradingRewardExpiration{
		RewardId: "epoch-700-reward",
		ExpireAt: 700,
	})
	suite.k.SetPendingTradingRewardExpiration(suite.ctx, types.TradingRewardExpiration{
		RewardId: "epoch-800-reward",
		ExpireAt: 800,
	})

	// Batch for epoch 700 should only return epoch 700 entries
	batch := suite.k.GetBatchPendingTradingRewardExpirationByExpireAt(suite.ctx, 700, 10)
	suite.Require().Len(batch, 1)
	suite.Require().Equal("epoch-700-reward", batch[0].RewardId)

	// Batch for epoch 800 should only return epoch 800 entries
	batch = suite.k.GetBatchPendingTradingRewardExpirationByExpireAt(suite.ctx, 800, 10)
	suite.Require().Len(batch, 1)
	suite.Require().Equal("epoch-800-reward", batch[0].RewardId)
}

// --- TradingRewardExpirationQueue store tests ---

func (suite *IntegrationTestSuite) TestStoreTradingRewardExpiration_Queue_SetAndGet() {
	queue := types.TradingRewardExpirationQueue{
		RemovalEpochs: []uint32{10, 20, 30},
	}
	suite.k.SetTradingRewardExpirationQueue(suite.ctx, queue)

	result, found := suite.k.GetTradingRewardExpirationQueue(suite.ctx)
	suite.Require().True(found)
	suite.Require().Equal(queue.RemovalEpochs, result.RemovalEpochs)
}

func (suite *IntegrationTestSuite) TestStoreTradingRewardExpiration_Queue_NotFound() {
	_, found := suite.k.GetTradingRewardExpirationQueue(suite.ctx)
	suite.Require().False(found)
}

func (suite *IntegrationTestSuite) TestStoreTradingRewardExpiration_Queue_Update() {
	suite.k.SetTradingRewardExpirationQueue(suite.ctx, types.TradingRewardExpirationQueue{
		RemovalEpochs: []uint32{100},
	})

	// Update the queue
	suite.k.SetTradingRewardExpirationQueue(suite.ctx, types.TradingRewardExpirationQueue{
		RemovalEpochs: []uint32{100, 200, 300},
	})

	result, found := suite.k.GetTradingRewardExpirationQueue(suite.ctx)
	suite.Require().True(found)
	suite.Require().Len(result.RemovalEpochs, 3)
	suite.Require().Equal(uint32(100), result.RemovalEpochs[0])
	suite.Require().Equal(uint32(200), result.RemovalEpochs[1])
	suite.Require().Equal(uint32(300), result.RemovalEpochs[2])
}

func (suite *IntegrationTestSuite) TestStoreTradingRewardExpiration_Queue_EmptyEpochs() {
	suite.k.SetTradingRewardExpirationQueue(suite.ctx, types.TradingRewardExpirationQueue{
		RemovalEpochs: []uint32{},
	})

	result, found := suite.k.GetTradingRewardExpirationQueue(suite.ctx)
	suite.Require().True(found)
	suite.Require().Empty(result.RemovalEpochs)
}

func (suite *IntegrationTestSuite) TestStoreTradingRewardExpiration_Queue_NilEpochs() {
	suite.k.SetTradingRewardExpirationQueue(suite.ctx, types.TradingRewardExpirationQueue{
		RemovalEpochs: nil,
	})

	result, found := suite.k.GetTradingRewardExpirationQueue(suite.ctx)
	suite.Require().True(found)
	suite.Require().Empty(result.RemovalEpochs)
}

func (suite *IntegrationTestSuite) TestStoreTradingRewardExpiration_Queue_IndependentFromExpirationStore() {
	// Set a queue
	suite.k.SetTradingRewardExpirationQueue(suite.ctx, types.TradingRewardExpirationQueue{
		RemovalEpochs: []uint32{100},
	})

	// Set a pending expiration
	suite.k.SetPendingTradingRewardExpiration(suite.ctx, types.TradingRewardExpiration{
		RewardId: "independent-test",
		ExpireAt: 100,
	})

	// Both should be independently retrievable
	queue, found := suite.k.GetTradingRewardExpirationQueue(suite.ctx)
	suite.Require().True(found)
	suite.Require().Len(queue.RemovalEpochs, 1)

	expirations := suite.k.GetAllPendingTradingRewardExpiration(suite.ctx)
	suite.Require().Len(expirations, 1)
}

func (suite *IntegrationTestSuite) TestStoreTradingRewardExpiration_RemoveNonExistent() {
	// Removing non-existent expiration should not cause issues
	suite.Require().NotPanics(func() {
		suite.k.RemovePendingTradingRewardExpiration(suite.ctx, 1704067200, "non-existent-reward")
		suite.k.RemoveActiveTradingRewardExpiration(suite.ctx, 1704067200, "non-existent-reward")
	})
}

func (suite *IntegrationTestSuite) TestStoreTradingRewardExpiration_UpdateExpiration() {
	originalExpiration := types.TradingRewardExpiration{
		RewardId: "update-expiration-test",
		ExpireAt: 1704067200,
	}

	updatedExpiration := types.TradingRewardExpiration{
		RewardId: "update-expiration-test",
		ExpireAt: 1704153600, // New expiration time
	}

	// Set original expiration
	suite.k.SetPendingTradingRewardExpiration(suite.ctx, originalExpiration)

	// Remove old expiration and set new one (simulating update)
	suite.k.RemovePendingTradingRewardExpiration(suite.ctx, 1704067200, "update-expiration-test")
	suite.k.SetPendingTradingRewardExpiration(suite.ctx, updatedExpiration)

	// Verify old expiration is gone
	oldExpirations := suite.k.GetAllPendingTradingRewardExpirationByExpireAt(suite.ctx, 1704067200)
	suite.Require().Empty(oldExpirations)

	// Verify new expiration exists
	newExpirations := suite.k.GetAllPendingTradingRewardExpirationByExpireAt(suite.ctx, 1704153600)
	suite.Require().Len(newExpirations, 1)
	suite.Require().Equal("update-expiration-test", newExpirations[0].RewardId)
	suite.Require().Equal(uint32(1704153600), newExpirations[0].ExpireAt)
}
