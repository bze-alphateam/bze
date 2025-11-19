package keeper_test

import (
	"cosmossdk.io/math"
	"fmt"
	"github.com/bze-alphateam/bze/x/rewards/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"time"
)

func (suite *IntegrationTestSuite) TestServiceHooks_GetDistributeAllStakingRewardsHook() {
	hook := suite.k.GetDistributeAllStakingRewardsHook()

	// Test with wrong epoch identifier
	err := hook.AfterEpochEnd(suite.ctx, "wrong-epoch", 100)
	suite.Require().NoError(err)

	// Test with correct epoch identifier - should call DistributeAllStakingRewards
	// Since it's a private method, we test by setting up a staking reward and checking state changes
	stakingReward := types.StakingReward{
		RewardId:         "hook-test-reward",
		PrizeAmount:      "1000",
		PrizeDenom:       "ubze",
		StakingDenom:     "ubze",
		Duration:         5,
		Payouts:          2,
		MinStake:         100,
		Lock:             7,
		StakedAmount:     "5000",
		DistributedStake: "100",
	}

	suite.k.SetStakingReward(suite.ctx, stakingReward)

	err = hook.AfterEpochEnd(suite.ctx, "day", 100)
	suite.Require().NoError(err)

	// Verify reward was processed (payouts incremented)
	retrievedReward, found := suite.k.GetStakingReward(suite.ctx, "hook-test-reward")
	suite.Require().True(found)
	suite.Require().Equal(uint32(3), retrievedReward.Payouts)
}

func (suite *IntegrationTestSuite) TestServiceHooks_GetUnlockPendingUnlockParticipantsHook() {
	hook := suite.k.GetUnlockPendingUnlockParticipantsHook()

	// Test with wrong epoch identifier
	err := hook.AfterEpochEnd(suite.ctx, "wrong-epoch", 100)
	suite.Require().NoError(err)

	// Test with correct epoch identifier
	epochNumber := int64(100)
	addr := sdk.AccAddress("addr1")
	participant := types.PendingUnlockParticipant{
		Index:   fmt.Sprintf("%d/%s", epochNumber, fmt.Sprintf("%s/%s", "reward_1", addr.String())),
		Address: addr.String(),
		Amount:  "1000",
		Denom:   "ubze",
	}

	suite.bank.EXPECT().
		SendCoinsFromModuleToAccount(
			suite.ctx,
			types.ModuleName,
			addr,
			sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(1000))),
		).
		Return(nil).
		Times(1)

	suite.k.SetPendingUnlockParticipant(suite.ctx, participant)

	err = hook.AfterEpochEnd(suite.ctx, "hour", epochNumber)
	suite.Require().NoError(err)

	// Verify participant was processed
	participants := suite.k.GetAllEpochPendingUnlockParticipant(suite.ctx, epochNumber)
	suite.Require().Empty(participants)
}

func (suite *IntegrationTestSuite) TestServiceHooks_GetRemoveExpiredPendingTradingRewardsHook() {
	hook := suite.k.GetRemoveExpiredPendingTradingRewardsHook()

	// Test with wrong epoch identifier
	err := hook.AfterEpochEnd(suite.ctx, "wrong-epoch", 100)
	suite.Require().NoError(err)

	// Test with correct epoch identifier
	epochNumber := int64(100)

	// Set up pending trading reward and expiration
	tradingReward := types.TradingReward{
		RewardId:    "expired-reward",
		PrizeAmount: "1000",
		PrizeDenom:  "ubze",
		Duration:    30,
		MarketId:    "market-1",
		Slots:       5,
		ExpireAt:    uint32(epochNumber),
	}

	expiration := types.TradingRewardExpiration{
		RewardId: "expired-reward",
		ExpireAt: uint32(epochNumber),
	}

	suite.k.SetPendingTradingReward(suite.ctx, tradingReward)
	suite.k.SetPendingTradingRewardExpiration(suite.ctx, expiration)

	// Set up mock expectation for burning coins
	suite.bank.EXPECT().
		BurnCoins(
			suite.ctx,
			types.ModuleName,
			sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(5000))), // 1000 * 5 slots
		).
		Return(nil).
		Times(1)

	err = hook.AfterEpochEnd(suite.ctx, "hour", epochNumber)
	suite.Require().NoError(err)

	// Verify reward and expiration were removed
	_, found := suite.k.GetPendingTradingReward(suite.ctx, "expired-reward")
	suite.Require().False(found)

	expirations := suite.k.GetAllPendingTradingRewardExpirationByExpireAt(suite.ctx, uint32(epochNumber))
	suite.Require().Empty(expirations)
}

func (suite *IntegrationTestSuite) TestServiceHooks_GetTradingRewardsDistributionHook() {
	hook := suite.k.GetTradingRewardsDistributionHook()

	// Test with wrong epoch identifier
	err := hook.AfterEpochEnd(suite.ctx, "wrong-epoch", 100)
	suite.Require().NoError(err)

	// Test with correct epoch identifier
	epochNumber := int64(100)
	addr1 := sdk.AccAddress("addr1")
	addr2 := sdk.AccAddress("addr2")

	// Set up active trading reward
	tradingReward := types.TradingReward{
		RewardId:    "distribute-reward",
		PrizeAmount: "1000",
		PrizeDenom:  "ubze",
		Duration:    30,
		MarketId:    "market-1",
		Slots:       2,
		ExpireAt:    uint32(epochNumber),
	}

	expiration := types.TradingRewardExpiration{
		RewardId: "distribute-reward",
		ExpireAt: uint32(epochNumber),
	}

	leaderboard := types.TradingRewardLeaderboard{
		RewardId: "distribute-reward",
		List: []types.TradingRewardLeaderboardEntry{
			{
				Amount:    "5000",
				Address:   addr1.String(),
				CreatedAt: 1000,
			},
			{
				Amount:    "3000",
				Address:   addr2.String(),
				CreatedAt: 2000,
			},
		},
	}

	suite.k.SetActiveTradingReward(suite.ctx, tradingReward)
	suite.k.SetActiveTradingRewardExpiration(suite.ctx, expiration)
	suite.k.SetTradingRewardLeaderboard(suite.ctx, leaderboard)
	suite.k.SetMarketIdRewardId(suite.ctx, types.MarketIdTradingRewardId{
		RewardId: "distribute-reward",
		MarketId: "market-1",
	})

	// Set up mock expectations for reward distribution
	rewardPerSlot := sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(1000)))

	suite.bank.EXPECT().
		SendCoinsFromModuleToAccount(
			suite.ctx,
			types.ModuleName,
			addr1,
			rewardPerSlot,
		).
		Return(nil).
		Times(1)

	suite.bank.EXPECT().
		SendCoinsFromModuleToAccount(
			suite.ctx,
			types.ModuleName,
			addr2,
			rewardPerSlot,
		).
		Return(nil).
		Times(1)

	err = hook.AfterEpochEnd(suite.ctx, "hour", epochNumber)
	suite.Require().NoError(err)

	// Verify market ID mapping was removed
	_, found := suite.k.GetMarketIdRewardId(suite.ctx, "market-1")
	suite.Require().False(found)

	// Verify expiration was extended
	updatedExpirations := suite.k.GetAllActiveTradingRewardExpirationByExpireAt(suite.ctx, uint32(epochNumber+168)) // +1 week in hours
	suite.Require().Len(updatedExpirations, 1)
}

func (suite *IntegrationTestSuite) TestServiceHooks_GetOnOrderFillHookNoReward() {
	hook := suite.k.GetOnOrderFillHook()

	// Test with market that has no reward
	suite.Require().NotPanics(func() {
		hook(suite.ctx, "non-existent-market", "1000", "bze1user")
	})
}

func (suite *IntegrationTestSuite) TestServiceHooks_GetOnOrderFillHookNewCandidate() {
	hook := suite.k.GetOnOrderFillHook()

	// Set up active trading reward
	tradingReward := types.TradingReward{
		RewardId:    "order-fill-reward",
		PrizeAmount: "1000",
		PrizeDenom:  "ubze",
		Duration:    30,
		MarketId:    "market-1",
		Slots:       3,
		ExpireAt:    1000,
	}

	marketMapping := types.MarketIdTradingRewardId{
		RewardId: "order-fill-reward",
		MarketId: "market-1",
	}

	suite.k.SetActiveTradingReward(suite.ctx, tradingReward)
	suite.k.SetMarketIdRewardId(suite.ctx, marketMapping)

	userAddr := "bze1user"

	// Mock block time
	blockTime := time.Unix(1000, 0)
	suite.ctx = suite.ctx.WithBlockTime(blockTime)

	hook(suite.ctx, "market-1", "500", userAddr)

	// Verify candidate was created
	candidate, found := suite.k.GetTradingRewardCandidate(suite.ctx, "order-fill-reward", userAddr)
	suite.Require().True(found)
	suite.Require().Equal("500", candidate.Amount)

	// Verify leaderboard was created
	leaderboard, found := suite.k.GetTradingRewardLeaderboard(suite.ctx, "order-fill-reward")
	suite.Require().True(found)
	suite.Require().Len(leaderboard.List, 1)
	suite.Require().Equal(userAddr, leaderboard.List[0].Address)
	suite.Require().Equal("500", leaderboard.List[0].Amount)
}

func (suite *IntegrationTestSuite) TestServiceHooks_GetOnOrderFillHookExistingCandidate() {
	hook := suite.k.GetOnOrderFillHook()

	// Set up active trading reward
	tradingReward := types.TradingReward{
		RewardId:    "order-fill-reward",
		PrizeAmount: "1000",
		PrizeDenom:  "ubze",
		Duration:    30,
		MarketId:    "market-1",
		Slots:       3,
		ExpireAt:    1000,
	}

	marketMapping := types.MarketIdTradingRewardId{
		RewardId: "order-fill-reward",
		MarketId: "market-1",
	}

	userAddr := "bze1user"

	// Set up existing candidate
	existingCandidate := types.TradingRewardCandidate{
		RewardId: "order-fill-reward",
		Amount:   "300",
		Address:  userAddr,
	}

	suite.k.SetActiveTradingReward(suite.ctx, tradingReward)
	suite.k.SetMarketIdRewardId(suite.ctx, marketMapping)
	suite.k.SetTradingRewardCandidate(suite.ctx, existingCandidate)

	// Mock block time
	blockTime := time.Unix(1000, 0)
	suite.ctx = suite.ctx.WithBlockTime(blockTime)

	hook(suite.ctx, "market-1", "200", userAddr)

	// Verify candidate amount was updated
	candidate, found := suite.k.GetTradingRewardCandidate(suite.ctx, "order-fill-reward", userAddr)
	suite.Require().True(found)
	suite.Require().Equal("500", candidate.Amount) // 300 + 200
}

func (suite *IntegrationTestSuite) TestServiceHooks_GetOnOrderFillHookLeaderboardUpdate() {
	hook := suite.k.GetOnOrderFillHook()

	// Set up active trading reward
	tradingReward := types.TradingReward{
		RewardId:    "order-fill-reward",
		PrizeAmount: "1000",
		PrizeDenom:  "ubze",
		Duration:    30,
		MarketId:    "market-1",
		Slots:       2,
		ExpireAt:    1000,
	}

	marketMapping := types.MarketIdTradingRewardId{
		RewardId: "order-fill-reward",
		MarketId: "market-1",
	}

	user1 := "bze1user1"
	user2 := "bze1user2"

	// Set up existing leaderboard
	existingLeaderboard := types.TradingRewardLeaderboard{
		RewardId: "order-fill-reward",
		List: []types.TradingRewardLeaderboardEntry{
			{
				Amount:    "1000",
				Address:   user1,
				CreatedAt: 1000,
			},
		},
	}

	suite.k.SetActiveTradingReward(suite.ctx, tradingReward)
	suite.k.SetMarketIdRewardId(suite.ctx, marketMapping)
	suite.k.SetTradingRewardLeaderboard(suite.ctx, existingLeaderboard)

	// Mock block time
	blockTime := time.Unix(2000, 0)
	suite.ctx = suite.ctx.WithBlockTime(blockTime)

	hook(suite.ctx, "market-1", "1500", user2)

	// Verify leaderboard was updated and sorted
	leaderboard, found := suite.k.GetTradingRewardLeaderboard(suite.ctx, "order-fill-reward")
	suite.Require().True(found)
	suite.Require().Len(leaderboard.List, 2)
}

func (suite *IntegrationTestSuite) TestServiceHooks_GetOnOrderFillHookLeaderboardTrim() {
	hook := suite.k.GetOnOrderFillHook()

	// Set up active trading reward with only 2 slots
	tradingReward := types.TradingReward{
		RewardId:    "order-fill-reward",
		PrizeAmount: "1000",
		PrizeDenom:  "ubze",
		Duration:    30,
		MarketId:    "market-1",
		Slots:       2,
		ExpireAt:    1000,
	}

	marketMapping := types.MarketIdTradingRewardId{
		RewardId: "order-fill-reward",
		MarketId: "market-1",
	}

	user1 := "bze1user1"
	user2 := "bze1user2"
	user3 := "bze1user3"

	// Set up existing leaderboard with 2 entries
	existingLeaderboard := types.TradingRewardLeaderboard{
		RewardId: "order-fill-reward",
		List: []types.TradingRewardLeaderboardEntry{
			{
				Amount:    "2000",
				Address:   user1,
				CreatedAt: 1000,
			},
			{
				Amount:    "1500",
				Address:   user2,
				CreatedAt: 1500,
			},
		},
	}

	suite.k.SetActiveTradingReward(suite.ctx, tradingReward)
	suite.k.SetMarketIdRewardId(suite.ctx, marketMapping)
	suite.k.SetTradingRewardLeaderboard(suite.ctx, existingLeaderboard)

	// Mock block time
	blockTime := time.Unix(2000, 0)
	suite.ctx = suite.ctx.WithBlockTime(blockTime)

	hook(suite.ctx, "market-1", "1000", user3)

	// Verify leaderboard was trimmed to 2 slots
	leaderboard, found := suite.k.GetTradingRewardLeaderboard(suite.ctx, "order-fill-reward")
	suite.Require().True(found)
	suite.Require().Len(leaderboard.List, 2)
}

func (suite *IntegrationTestSuite) TestServiceHooks_GetOnOrderFillHookInvalidAmounts() {
	hook := suite.k.GetOnOrderFillHook()

	// Set up active trading reward
	tradingReward := types.TradingReward{
		RewardId:    "order-fill-reward",
		PrizeAmount: "1000",
		PrizeDenom:  "ubze",
		Duration:    30,
		MarketId:    "market-1",
		Slots:       3,
		ExpireAt:    1000,
	}

	marketMapping := types.MarketIdTradingRewardId{
		RewardId: "order-fill-reward",
		MarketId: "market-1",
	}

	// Set up candidate with invalid amount
	invalidCandidate := types.TradingRewardCandidate{
		RewardId: "order-fill-reward",
		Amount:   "invalid-amount",
		Address:  "bze1user",
	}

	suite.k.SetActiveTradingReward(suite.ctx, tradingReward)
	suite.k.SetMarketIdRewardId(suite.ctx, marketMapping)
	suite.k.SetTradingRewardCandidate(suite.ctx, invalidCandidate)

	// Should not panic with invalid amounts
	suite.Require().NotPanics(func() {
		hook(suite.ctx, "market-1", "invalid-traded-amount", "bze1user")
	})

	suite.Require().NotPanics(func() {
		hook(suite.ctx, "market-1", "500", "bze1user")
	})
}
