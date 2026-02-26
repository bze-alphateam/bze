package keeper_test

import (
	"cosmossdk.io/math"
	"fmt"
	burnermoduletypes "github.com/bze-alphateam/bze/x/burner/types"
	"github.com/bze-alphateam/bze/x/rewards/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"go.uber.org/mock/gomock"
)

// --- EnqueueExpiredTradingRewardRemoval tests ---

func (suite *IntegrationTestSuite) TestEnqueueExpiredTradingRewardRemoval_Empty() {
	// No expirations - should not create queue
	suite.k.EnqueueExpiredTradingRewardRemoval(suite.ctx, 100)

	_, found := suite.k.GetTradingRewardExpirationQueue(suite.ctx)
	suite.Require().False(found)
}

func (suite *IntegrationTestSuite) TestEnqueueExpiredTradingRewardRemoval_WithExpirations() {
	epochNumber := int64(100)

	suite.k.SetPendingTradingRewardExpiration(suite.ctx, types.TradingRewardExpiration{
		RewardId: "reward-1",
		ExpireAt: uint32(epochNumber),
	})

	suite.k.EnqueueExpiredTradingRewardRemoval(suite.ctx, epochNumber)

	queue, found := suite.k.GetTradingRewardExpirationQueue(suite.ctx)
	suite.Require().True(found)
	suite.Require().Len(queue.RemovalEpochs, 1)
	suite.Require().Equal(uint32(epochNumber), queue.RemovalEpochs[0])
}

func (suite *IntegrationTestSuite) TestEnqueueExpiredTradingRewardRemoval_MultipleEpochs() {
	for _, epoch := range []int64{100, 200} {
		suite.k.SetPendingTradingRewardExpiration(suite.ctx, types.TradingRewardExpiration{
			RewardId: fmt.Sprintf("reward-epoch-%d", epoch),
			ExpireAt: uint32(epoch),
		})
		suite.k.EnqueueExpiredTradingRewardRemoval(suite.ctx, epoch)
	}

	queue, found := suite.k.GetTradingRewardExpirationQueue(suite.ctx)
	suite.Require().True(found)
	suite.Require().Len(queue.RemovalEpochs, 2)
	suite.Require().Equal(uint32(100), queue.RemovalEpochs[0])
	suite.Require().Equal(uint32(200), queue.RemovalEpochs[1])
}

// --- ProcessExpiredTradingRewardRemovalQueue tests ---

func (suite *IntegrationTestSuite) TestProcessExpiredTradingRewardQueue_EmptyQueue() {
	// No queue at all - should not panic
	suite.Require().NotPanics(func() {
		suite.k.ProcessExpiredTradingRewardRemovalQueue(suite.ctx)
	})
}

func (suite *IntegrationTestSuite) TestProcessExpiredTradingRewardQueue_SingleEntry() {
	epochNumber := int64(100)

	tradingReward := types.TradingReward{
		RewardId:    "expired-reward-1",
		PrizeAmount: math.NewInt(1000),
		PrizeDenom:  "ubze",
		Duration:    30,
		MarketId:    "market-1",
		Slots:       5,
		ExpireAt:    uint32(epochNumber),
	}

	suite.k.SetPendingTradingReward(suite.ctx, tradingReward)
	suite.k.SetPendingTradingRewardExpiration(suite.ctx, types.TradingRewardExpiration{
		RewardId: "expired-reward-1",
		ExpireAt: uint32(epochNumber),
	})

	// Enqueue
	suite.k.EnqueueExpiredTradingRewardRemoval(suite.ctx, epochNumber)

	// Set up mock expectation for burning coins
	suite.bank.EXPECT().
		SendCoinsFromModuleToModule(
			gomock.Any(),
			types.ModuleName,
			burnermoduletypes.ModuleName,
			sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(5000))), // 1000 * 5 slots
		).
		Return(nil).
		Times(1)

	suite.k.ProcessExpiredTradingRewardRemovalQueue(suite.ctx)

	// Verify reward and expiration were removed
	_, found := suite.k.GetPendingTradingReward(suite.ctx, "expired-reward-1")
	suite.Require().False(found)

	expirations := suite.k.GetAllPendingTradingRewardExpirationByExpireAt(suite.ctx, uint32(epochNumber))
	suite.Require().Empty(expirations)

	// Verify queue epoch was removed
	queue, found := suite.k.GetTradingRewardExpirationQueue(suite.ctx)
	suite.Require().True(found)
	suite.Require().Empty(queue.RemovalEpochs)
}

func (suite *IntegrationTestSuite) TestProcessExpiredTradingRewardQueue_MultipleEntries() {
	epochNumber := int64(100)

	for i := 1; i <= 3; i++ {
		suite.k.SetPendingTradingReward(suite.ctx, types.TradingReward{
			RewardId:    fmt.Sprintf("expired-reward-%d", i),
			PrizeAmount: math.NewInt(1000),
			PrizeDenom:  "ubze",
			Duration:    30,
			MarketId:    fmt.Sprintf("market-%d", i),
			Slots:       2,
			ExpireAt:    uint32(epochNumber),
		})
		suite.k.SetPendingTradingRewardExpiration(suite.ctx, types.TradingRewardExpiration{
			RewardId: fmt.Sprintf("expired-reward-%d", i),
			ExpireAt: uint32(epochNumber),
		})
	}

	suite.k.EnqueueExpiredTradingRewardRemoval(suite.ctx, epochNumber)

	suite.bank.EXPECT().
		SendCoinsFromModuleToModule(
			gomock.Any(),
			types.ModuleName,
			burnermoduletypes.ModuleName,
			sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(2000))), // 1000 * 2 slots
		).
		Return(nil).
		Times(3)

	suite.k.ProcessExpiredTradingRewardRemovalQueue(suite.ctx)

	// Verify all rewards were removed
	for i := 1; i <= 3; i++ {
		_, found := suite.k.GetPendingTradingReward(suite.ctx, fmt.Sprintf("expired-reward-%d", i))
		suite.Require().False(found)
	}

	// Verify epoch was removed from queue
	queue, found := suite.k.GetTradingRewardExpirationQueue(suite.ctx)
	suite.Require().True(found)
	suite.Require().Empty(queue.RemovalEpochs)
}

func (suite *IntegrationTestSuite) TestProcessExpiredTradingRewardQueue_BurnErrorKeepsEpochInQueue() {
	epochNumber := int64(100)

	suite.k.SetPendingTradingReward(suite.ctx, types.TradingReward{
		RewardId:    "expired-reward-1",
		PrizeAmount: math.NewInt(1000),
		PrizeDenom:  "ubze",
		Duration:    30,
		MarketId:    "market-1",
		Slots:       5,
		ExpireAt:    uint32(epochNumber),
	})
	suite.k.SetPendingTradingRewardExpiration(suite.ctx, types.TradingRewardExpiration{
		RewardId: "expired-reward-1",
		ExpireAt: uint32(epochNumber),
	})

	suite.k.EnqueueExpiredTradingRewardRemoval(suite.ctx, epochNumber)

	suite.bank.EXPECT().
		SendCoinsFromModuleToModule(
			gomock.Any(),
			types.ModuleName,
			burnermoduletypes.ModuleName,
			sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(5000))),
		).
		Return(fmt.Errorf("burn failed")).
		Times(1)

	suite.Require().NotPanics(func() {
		suite.k.ProcessExpiredTradingRewardRemovalQueue(suite.ctx)
	})

	// Epoch should remain in queue for retry
	queue, found := suite.k.GetTradingRewardExpirationQueue(suite.ctx)
	suite.Require().True(found)
	suite.Require().Len(queue.RemovalEpochs, 1)
	suite.Require().Equal(uint32(epochNumber), queue.RemovalEpochs[0])

	// Verify state was rolled back - expiration and reward still exist for retry
	_, found = suite.k.GetPendingTradingReward(suite.ctx, "expired-reward-1")
	suite.Require().True(found)

	expirations := suite.k.GetAllPendingTradingRewardExpirationByExpireAt(suite.ctx, uint32(epochNumber))
	suite.Require().Len(expirations, 1)
}

func (suite *IntegrationTestSuite) TestProcessExpiredTradingRewardQueue_BurnErrorRetrySucceeds() {
	epochNumber := int64(100)

	suite.k.SetPendingTradingReward(suite.ctx, types.TradingReward{
		RewardId:    "retry-reward",
		PrizeAmount: math.NewInt(1000),
		PrizeDenom:  "ubze",
		Duration:    30,
		MarketId:    "market-1",
		Slots:       2,
		ExpireAt:    uint32(epochNumber),
	})
	suite.k.SetPendingTradingRewardExpiration(suite.ctx, types.TradingRewardExpiration{
		RewardId: "retry-reward",
		ExpireAt: uint32(epochNumber),
	})

	suite.k.EnqueueExpiredTradingRewardRemoval(suite.ctx, epochNumber)

	burnCoins := sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(2000)))

	// First call fails
	suite.bank.EXPECT().
		SendCoinsFromModuleToModule(gomock.Any(), types.ModuleName, burnermoduletypes.ModuleName, burnCoins).
		Return(fmt.Errorf("burn failed")).
		Times(1)

	suite.k.ProcessExpiredTradingRewardRemovalQueue(suite.ctx)

	// Epoch still in queue, entries still exist
	queue, found := suite.k.GetTradingRewardExpirationQueue(suite.ctx)
	suite.Require().True(found)
	suite.Require().Len(queue.RemovalEpochs, 1)

	_, found = suite.k.GetPendingTradingReward(suite.ctx, "retry-reward")
	suite.Require().True(found)

	// Second call succeeds
	suite.bank.EXPECT().
		SendCoinsFromModuleToModule(gomock.Any(), types.ModuleName, burnermoduletypes.ModuleName, burnCoins).
		Return(nil).
		Times(1)

	suite.k.ProcessExpiredTradingRewardRemovalQueue(suite.ctx)

	// Now reward should be removed and queue drained
	_, found = suite.k.GetPendingTradingReward(suite.ctx, "retry-reward")
	suite.Require().False(found)

	expirations := suite.k.GetAllPendingTradingRewardExpirationByExpireAt(suite.ctx, uint32(epochNumber))
	suite.Require().Empty(expirations)

	queue, found = suite.k.GetTradingRewardExpirationQueue(suite.ctx)
	suite.Require().True(found)
	suite.Require().Empty(queue.RemovalEpochs)
}

func (suite *IntegrationTestSuite) TestProcessExpiredTradingRewardQueue_MultipleEpochs() {
	// Set up rewards for two different epochs
	for _, epoch := range []int64{100, 200} {
		suite.k.SetPendingTradingReward(suite.ctx, types.TradingReward{
			RewardId:    fmt.Sprintf("reward-epoch-%d", epoch),
			PrizeAmount: math.NewInt(500),
			PrizeDenom:  "ubze",
			Duration:    30,
			MarketId:    fmt.Sprintf("market-%d", epoch),
			Slots:       2,
			ExpireAt:    uint32(epoch),
		})
		suite.k.SetPendingTradingRewardExpiration(suite.ctx, types.TradingRewardExpiration{
			RewardId: fmt.Sprintf("reward-epoch-%d", epoch),
			ExpireAt: uint32(epoch),
		})
		suite.k.EnqueueExpiredTradingRewardRemoval(suite.ctx, epoch)
	}

	suite.bank.EXPECT().
		SendCoinsFromModuleToModule(
			gomock.Any(),
			types.ModuleName,
			burnermoduletypes.ModuleName,
			sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(1000))), // 500 * 2 slots
		).
		Return(nil).
		Times(2)

	suite.k.ProcessExpiredTradingRewardRemovalQueue(suite.ctx)

	// Both rewards should be removed
	_, found := suite.k.GetPendingTradingReward(suite.ctx, "reward-epoch-100")
	suite.Require().False(found)
	_, found = suite.k.GetPendingTradingReward(suite.ctx, "reward-epoch-200")
	suite.Require().False(found)

	// Queue should be empty
	queue, found := suite.k.GetTradingRewardExpirationQueue(suite.ctx)
	suite.Require().True(found)
	suite.Require().Empty(queue.RemovalEpochs)
}

func (suite *IntegrationTestSuite) TestProcessExpiredTradingRewardQueue_MultiBlockDrain() {
	epochNumber := int64(100)
	totalEntries := types.MaxTradingRewardRemovalsPerBlock + 50 // 150 entries

	for i := 1; i <= totalEntries; i++ {
		rewardId := fmt.Sprintf("drain-reward-%d", i)
		suite.k.SetPendingTradingReward(suite.ctx, types.TradingReward{
			RewardId:    rewardId,
			PrizeAmount: math.NewInt(100),
			PrizeDenom:  "ubze",
			Duration:    30,
			MarketId:    fmt.Sprintf("market-%d", i),
			Slots:       1,
			ExpireAt:    uint32(epochNumber),
		})
		suite.k.SetPendingTradingRewardExpiration(suite.ctx, types.TradingRewardExpiration{
			RewardId: rewardId,
			ExpireAt: uint32(epochNumber),
		})
	}

	suite.k.EnqueueExpiredTradingRewardRemoval(suite.ctx, epochNumber)

	burnCoins := sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(100)))

	// First block: process MaxTradingRewardRemovalsPerBlock entries
	suite.bank.EXPECT().
		SendCoinsFromModuleToModule(gomock.Any(), types.ModuleName, burnermoduletypes.ModuleName, burnCoins).
		Return(nil).
		Times(types.MaxTradingRewardRemovalsPerBlock)

	suite.k.ProcessExpiredTradingRewardRemovalQueue(suite.ctx)

	// Epoch should still be in queue (not fully drained)
	queue, found := suite.k.GetTradingRewardExpirationQueue(suite.ctx)
	suite.Require().True(found)
	suite.Require().Len(queue.RemovalEpochs, 1)

	// Remaining entries should still exist
	remaining := suite.k.GetAllPendingTradingRewardExpirationByExpireAt(suite.ctx, uint32(epochNumber))
	suite.Require().Len(remaining, 50)

	// Second block: process remaining 50 entries
	suite.bank.EXPECT().
		SendCoinsFromModuleToModule(gomock.Any(), types.ModuleName, burnermoduletypes.ModuleName, burnCoins).
		Return(nil).
		Times(50)

	suite.k.ProcessExpiredTradingRewardRemovalQueue(suite.ctx)

	// All entries processed, epoch should be removed from queue
	queue, found = suite.k.GetTradingRewardExpirationQueue(suite.ctx)
	suite.Require().True(found)
	suite.Require().Empty(queue.RemovalEpochs)

	// All rewards and expirations should be removed
	remaining = suite.k.GetAllPendingTradingRewardExpirationByExpireAt(suite.ctx, uint32(epochNumber))
	suite.Require().Empty(remaining)

	for i := 1; i <= totalEntries; i++ {
		_, found := suite.k.GetPendingTradingReward(suite.ctx, fmt.Sprintf("drain-reward-%d", i))
		suite.Require().False(found)
	}
}

func (suite *IntegrationTestSuite) TestProcessExpiredTradingRewardQueue_DifferentEpochsOnlyQueuedProcessed() {
	// Set up rewards for two different epochs but only enqueue one
	for _, epoch := range []int64{100, 200} {
		suite.k.SetPendingTradingReward(suite.ctx, types.TradingReward{
			RewardId:    fmt.Sprintf("reward-epoch-%d", epoch),
			PrizeAmount: math.NewInt(500),
			PrizeDenom:  "ubze",
			Duration:    30,
			MarketId:    fmt.Sprintf("market-%d", epoch),
			Slots:       2,
			ExpireAt:    uint32(epoch),
		})
		suite.k.SetPendingTradingRewardExpiration(suite.ctx, types.TradingRewardExpiration{
			RewardId: fmt.Sprintf("reward-epoch-%d", epoch),
			ExpireAt: uint32(epoch),
		})
	}

	// Only enqueue epoch 100
	suite.k.EnqueueExpiredTradingRewardRemoval(suite.ctx, 100)

	suite.bank.EXPECT().
		SendCoinsFromModuleToModule(
			gomock.Any(),
			types.ModuleName,
			burnermoduletypes.ModuleName,
			sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(1000))),
		).
		Return(nil).
		Times(1)

	suite.k.ProcessExpiredTradingRewardRemovalQueue(suite.ctx)

	// Epoch 100 reward should be removed
	_, found := suite.k.GetPendingTradingReward(suite.ctx, "reward-epoch-100")
	suite.Require().False(found)

	// Epoch 200 reward should still exist
	_, found = suite.k.GetPendingTradingReward(suite.ctx, "reward-epoch-200")
	suite.Require().True(found)
}
