package keeper_test

import (
	"github.com/bze-alphateam/bze/x/rewards/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
)

func (suite *IntegrationTestSuite) TestMsgServerGov_ActivateTradingRewardSuccess() {
	authority := authtypes.NewModuleAddress(govtypes.ModuleName).String()

	// Set up pending trading reward
	pendingReward := types.TradingReward{
		RewardId:    "activate-reward",
		PrizeAmount: "1000",
		PrizeDenom:  "ubze",
		Duration:    30,
		MarketId:    "market-1",
		Slots:       5,
		ExpireAt:    1000,
	}

	pendingExpiration := types.TradingRewardExpiration{
		RewardId: "activate-reward",
		ExpireAt: 1000,
	}

	suite.k.SetPendingTradingReward(suite.ctx, pendingReward)
	suite.k.SetPendingTradingRewardExpiration(suite.ctx, pendingExpiration)

	// Mock epoch keeper call
	suite.epoch.EXPECT().
		GetEpochCountByIdentifier(suite.ctx, "hour").
		Return(int64(100)).
		Times(1)

	msg := &types.MsgActivateTradingReward{
		Creator:  authority,
		RewardId: "activate-reward",
	}

	response, err := suite.msgServer.ActivateTradingReward(suite.ctx, msg)
	suite.Require().NoError(err)
	suite.Require().NotNil(response)

	// Verify pending reward was removed
	_, found := suite.k.GetPendingTradingReward(suite.ctx, "activate-reward")
	suite.Require().False(found)

	// Verify pending expiration was removed
	pendingExpirations := suite.k.GetAllPendingTradingRewardExpirationByExpireAt(suite.ctx, 1000)
	suite.Require().Empty(pendingExpirations)

	// Verify active reward was created
	activeReward, found := suite.k.GetActiveTradingReward(suite.ctx, "activate-reward")
	suite.Require().True(found)
	suite.Require().Equal(pendingReward.RewardId, activeReward.RewardId)
	suite.Require().Equal(pendingReward.PrizeAmount, activeReward.PrizeAmount)
	suite.Require().Equal(pendingReward.MarketId, activeReward.MarketId)
	suite.Require().NotEqual(uint32(1000), activeReward.ExpireAt)     // Should have new expiration
	suite.Require().Equal(uint32(100+(30*24)), activeReward.ExpireAt) // epoch(100) + Duration(30) * 24

	// Verify active expiration was created
	activeExpirations := suite.k.GetAllActiveTradingRewardExpirationByExpireAt(suite.ctx, activeReward.ExpireAt)
	suite.Require().Len(activeExpirations, 1)
	suite.Require().Equal("activate-reward", activeExpirations[0].RewardId)

	// Verify market mapping was created
	marketMapping, found := suite.k.GetMarketIdRewardId(suite.ctx, "market-1")
	suite.Require().True(found)
	suite.Require().Equal("activate-reward", marketMapping.RewardId)
	suite.Require().Equal("market-1", marketMapping.MarketId)
}

func (suite *IntegrationTestSuite) TestMsgServerGov_ActivateTradingRewardNilRequest() {
	response, err := suite.msgServer.ActivateTradingReward(suite.ctx, nil)

	suite.Require().Error(err)
	suite.Require().Nil(response)
	suite.Require().Equal(sdkerrors.ErrInvalidRequest, err)
}

func (suite *IntegrationTestSuite) TestMsgServerGov_ActivateTradingRewardInvalidAuthority() {
	invalidAuthority := "bze1invalidauthority"

	msg := &types.MsgActivateTradingReward{
		Creator:  invalidAuthority,
		RewardId: "test-reward",
	}

	response, err := suite.msgServer.ActivateTradingReward(suite.ctx, msg)

	suite.Require().Error(err)
	suite.Require().Nil(response)
	suite.Require().Contains(err.Error(), "invalid authority")
}

func (suite *IntegrationTestSuite) TestMsgServerGov_ActivateTradingRewardAlreadyActive() {
	authority := authtypes.NewModuleAddress(govtypes.ModuleName).String()

	// Set up active trading reward
	activeReward := types.TradingReward{
		RewardId:    "already-active-reward",
		PrizeAmount: "1000",
		PrizeDenom:  "ubze",
		Duration:    30,
		MarketId:    "market-1",
		Slots:       5,
		ExpireAt:    2000,
	}

	suite.k.SetActiveTradingReward(suite.ctx, activeReward)

	msg := &types.MsgActivateTradingReward{
		Creator:  authority,
		RewardId: "already-active-reward",
	}

	response, err := suite.msgServer.ActivateTradingReward(suite.ctx, msg)

	suite.Require().Error(err)
	suite.Require().Nil(response)
	suite.Require().Contains(err.Error(), "trading reward already active")
}

func (suite *IntegrationTestSuite) TestMsgServerGov_ActivateTradingRewardNotFound() {
	authority := authtypes.NewModuleAddress(govtypes.ModuleName).String()

	msg := &types.MsgActivateTradingReward{
		Creator:  authority,
		RewardId: "non-existent-reward",
	}

	response, err := suite.msgServer.ActivateTradingReward(suite.ctx, msg)

	suite.Require().Error(err)
	suite.Require().Nil(response)
	suite.Require().Contains(err.Error(), "trading reward not found")
}

func (suite *IntegrationTestSuite) TestMsgServerGov_ActivateTradingRewardPendingButAlreadyActive() {
	authority := authtypes.NewModuleAddress(govtypes.ModuleName).String()

	// Set up both pending and active rewards with same ID
	pendingReward := types.TradingReward{
		RewardId:    "conflict-reward",
		PrizeAmount: "1000",
		PrizeDenom:  "ubze",
		Duration:    30,
		MarketId:    "market-1",
		Slots:       5,
		ExpireAt:    1000,
	}

	activeReward := types.TradingReward{
		RewardId:    "conflict-reward",
		PrizeAmount: "2000",
		PrizeDenom:  "utoken",
		Duration:    60,
		MarketId:    "market-2",
		Slots:       10,
		ExpireAt:    2000,
	}

	suite.k.SetPendingTradingReward(suite.ctx, pendingReward)
	suite.k.SetActiveTradingReward(suite.ctx, activeReward)

	msg := &types.MsgActivateTradingReward{
		Creator:  authority,
		RewardId: "conflict-reward",
	}

	response, err := suite.msgServer.ActivateTradingReward(suite.ctx, msg)

	suite.Require().Error(err)
	suite.Require().Nil(response)
	suite.Require().Contains(err.Error(), "trading reward already active")
}

func (suite *IntegrationTestSuite) TestMsgServerGov_ActivateTradingRewardWithExistingMarketMapping() {
	authority := authtypes.NewModuleAddress(govtypes.ModuleName).String()

	// Set up existing market mapping (simulates another active reward on this market)
	existingMapping := types.MarketIdTradingRewardId{
		RewardId: "existing-reward",
		MarketId: "market-1",
	}
	suite.k.SetMarketIdRewardId(suite.ctx, existingMapping)

	// Set up pending trading reward for same market
	pendingReward := types.TradingReward{
		RewardId:    "new-reward",
		PrizeAmount: "1000",
		PrizeDenom:  "ubze",
		Duration:    30,
		MarketId:    "market-1", // Same market as existing mapping
		Slots:       5,
		ExpireAt:    1000,
	}

	suite.k.SetPendingTradingReward(suite.ctx, pendingReward)

	msg := &types.MsgActivateTradingReward{
		Creator:  authority,
		RewardId: "new-reward",
	}

	response, err := suite.msgServer.ActivateTradingReward(suite.ctx, msg)
	suite.Require().Error(err)
	suite.Require().Nil(response)
	suite.Require().ErrorIs(err, types.ErrRewardAlreadyExists)

	// Verify existing market mapping was NOT overwritten
	marketMapping, found := suite.k.GetMarketIdRewardId(suite.ctx, "market-1")
	suite.Require().True(found)
	suite.Require().Equal("existing-reward", marketMapping.RewardId)

	// Verify the pending reward was NOT removed
	_, found = suite.k.GetPendingTradingReward(suite.ctx, "new-reward")
	suite.Require().True(found)
}

func (suite *IntegrationTestSuite) TestMsgServerGov_ActivateTradingRewardSecondMarketRewardBlocked() {
	authority := authtypes.NewModuleAddress(govtypes.ModuleName).String()

	// Set up first pending reward for market-1
	pendingReward1 := types.TradingReward{
		RewardId:    "first-reward",
		PrizeAmount: "1000",
		PrizeDenom:  "ubze",
		Duration:    30,
		MarketId:    "market-1",
		Slots:       5,
		ExpireAt:    1000,
	}
	pendingExpiration1 := types.TradingRewardExpiration{
		RewardId: "first-reward",
		ExpireAt: 1000,
	}
	suite.k.SetPendingTradingReward(suite.ctx, pendingReward1)
	suite.k.SetPendingTradingRewardExpiration(suite.ctx, pendingExpiration1)

	// Set up second pending reward for the same market
	pendingReward2 := types.TradingReward{
		RewardId:    "second-reward",
		PrizeAmount: "2000",
		PrizeDenom:  "ubze",
		Duration:    60,
		MarketId:    "market-1",
		Slots:       10,
		ExpireAt:    1000,
	}
	pendingExpiration2 := types.TradingRewardExpiration{
		RewardId: "second-reward",
		ExpireAt: 1000,
	}
	suite.k.SetPendingTradingReward(suite.ctx, pendingReward2)
	suite.k.SetPendingTradingRewardExpiration(suite.ctx, pendingExpiration2)

	// Mock epoch keeper - only first activation should reach this
	suite.epoch.EXPECT().
		GetEpochCountByIdentifier(suite.ctx, "hour").
		Return(int64(100)).
		Times(1)

	// Activate first reward - should succeed
	msg1 := &types.MsgActivateTradingReward{
		Creator:  authority,
		RewardId: "first-reward",
	}
	response, err := suite.msgServer.ActivateTradingReward(suite.ctx, msg1)
	suite.Require().NoError(err)
	suite.Require().NotNil(response)

	// Try to activate second reward for same market - should fail
	msg2 := &types.MsgActivateTradingReward{
		Creator:  authority,
		RewardId: "second-reward",
	}
	response, err = suite.msgServer.ActivateTradingReward(suite.ctx, msg2)
	suite.Require().Error(err)
	suite.Require().Nil(response)
	suite.Require().ErrorIs(err, types.ErrRewardAlreadyExists)

	// Verify market mapping still points to first reward
	marketMapping, found := suite.k.GetMarketIdRewardId(suite.ctx, "market-1")
	suite.Require().True(found)
	suite.Require().Equal("first-reward", marketMapping.RewardId)

	// Verify second pending reward was NOT removed
	_, found = suite.k.GetPendingTradingReward(suite.ctx, "second-reward")
	suite.Require().True(found)
}

func (suite *IntegrationTestSuite) TestMsgServerGov_ActivateTradingRewardPreservesRewardData() {
	authority := authtypes.NewModuleAddress(govtypes.ModuleName).String()

	// Set up pending trading reward with all fields
	pendingReward := types.TradingReward{
		RewardId:    "preserve-data-reward",
		PrizeAmount: "5000",
		PrizeDenom:  "uspecial",
		Duration:    120,
		MarketId:    "special-market",
		Slots:       25,
		ExpireAt:    1500,
	}

	pendingExpiration := types.TradingRewardExpiration{
		RewardId: "preserve-data-reward",
		ExpireAt: 1500,
	}

	suite.k.SetPendingTradingReward(suite.ctx, pendingReward)
	suite.k.SetPendingTradingRewardExpiration(suite.ctx, pendingExpiration)

	// Mock epoch keeper call
	suite.epoch.EXPECT().
		GetEpochCountByIdentifier(suite.ctx, "hour").
		Return(int64(200)).
		Times(1)

	msg := &types.MsgActivateTradingReward{
		Creator:  authority,
		RewardId: "preserve-data-reward",
	}

	response, err := suite.msgServer.ActivateTradingReward(suite.ctx, msg)
	suite.Require().NoError(err)
	suite.Require().NotNil(response)

	// Verify all original data is preserved except ExpireAt
	activeReward, found := suite.k.GetActiveTradingReward(suite.ctx, "preserve-data-reward")
	suite.Require().True(found)
	suite.Require().Equal(pendingReward.RewardId, activeReward.RewardId)
	suite.Require().Equal(pendingReward.PrizeAmount, activeReward.PrizeAmount)
	suite.Require().Equal(pendingReward.PrizeDenom, activeReward.PrizeDenom)
	suite.Require().Equal(pendingReward.Duration, activeReward.Duration)
	suite.Require().Equal(pendingReward.MarketId, activeReward.MarketId)
	suite.Require().Equal(pendingReward.Slots, activeReward.Slots)
	suite.Require().NotEqual(pendingReward.ExpireAt, activeReward.ExpireAt) // Should be updated
	suite.Require().Equal(uint32(200+(120*24)), activeReward.ExpireAt)      // epoch(200) + Duration(120) * 24
}

func (suite *IntegrationTestSuite) TestMsgServerGov_ActivateTradingRewardUsesRewardDuration() {
	authority := authtypes.NewModuleAddress(govtypes.ModuleName).String()

	// Set up pending trading reward with Duration=7 (7-day competition)
	pendingReward := types.TradingReward{
		RewardId:    "short-duration-reward",
		PrizeAmount: "1000",
		PrizeDenom:  "ubze",
		Duration:    7,
		MarketId:    "market-1",
		Slots:       5,
		ExpireAt:    1000,
	}

	pendingExpiration := types.TradingRewardExpiration{
		RewardId: "short-duration-reward",
		ExpireAt: 1000,
	}

	suite.k.SetPendingTradingReward(suite.ctx, pendingReward)
	suite.k.SetPendingTradingRewardExpiration(suite.ctx, pendingExpiration)

	// Mock epoch keeper call - current epoch is 500
	suite.epoch.EXPECT().
		GetEpochCountByIdentifier(suite.ctx, "hour").
		Return(int64(500)).
		Times(1)

	msg := &types.MsgActivateTradingReward{
		Creator:  authority,
		RewardId: "short-duration-reward",
	}

	response, err := suite.msgServer.ActivateTradingReward(suite.ctx, msg)
	suite.Require().NoError(err)
	suite.Require().NotNil(response)

	// Verify active reward uses Duration for expiration: epoch(500) + Duration(7) * 24 = 668
	activeReward, found := suite.k.GetActiveTradingReward(suite.ctx, "short-duration-reward")
	suite.Require().True(found)
	suite.Require().Equal(uint32(500+(7*24)), activeReward.ExpireAt)
	suite.Require().Equal(uint32(668), activeReward.ExpireAt)

	// Verify active expiration was created at the correct time
	activeExpirations := suite.k.GetAllActiveTradingRewardExpirationByExpireAt(suite.ctx, activeReward.ExpireAt)
	suite.Require().Len(activeExpirations, 1)
	suite.Require().Equal("short-duration-reward", activeExpirations[0].RewardId)
}

func (suite *IntegrationTestSuite) TestMsgServerGov_ActivateTradingRewardCleanup() {
	authority := authtypes.NewModuleAddress(govtypes.ModuleName).String()

	// Set up multiple pending rewards, we'll activate one
	pendingReward1 := types.TradingReward{
		RewardId:    "cleanup-reward-1",
		PrizeAmount: "1000",
		PrizeDenom:  "ubze",
		Duration:    30,
		MarketId:    "market-1",
		Slots:       5,
		ExpireAt:    1000,
	}

	pendingReward2 := types.TradingReward{
		RewardId:    "cleanup-reward-2",
		PrizeAmount: "2000",
		PrizeDenom:  "utoken",
		Duration:    60,
		MarketId:    "market-2",
		Slots:       10,
		ExpireAt:    1000,
	}

	pendingExpiration1 := types.TradingRewardExpiration{
		RewardId: "cleanup-reward-1",
		ExpireAt: 1000,
	}

	pendingExpiration2 := types.TradingRewardExpiration{
		RewardId: "cleanup-reward-2",
		ExpireAt: 1000,
	}

	suite.k.SetPendingTradingReward(suite.ctx, pendingReward1)
	suite.k.SetPendingTradingReward(suite.ctx, pendingReward2)
	suite.k.SetPendingTradingRewardExpiration(suite.ctx, pendingExpiration1)
	suite.k.SetPendingTradingRewardExpiration(suite.ctx, pendingExpiration2)

	// Mock epoch keeper call
	suite.epoch.EXPECT().
		GetEpochCountByIdentifier(suite.ctx, "hour").
		Return(int64(250)).
		Times(1)

	msg := &types.MsgActivateTradingReward{
		Creator:  authority,
		RewardId: "cleanup-reward-1",
	}

	response, err := suite.msgServer.ActivateTradingReward(suite.ctx, msg)
	suite.Require().NoError(err)
	suite.Require().NotNil(response)

	// Verify only the activated reward was removed from pending
	_, found1 := suite.k.GetPendingTradingReward(suite.ctx, "cleanup-reward-1")
	suite.Require().False(found1)

	_, found2 := suite.k.GetPendingTradingReward(suite.ctx, "cleanup-reward-2")
	suite.Require().True(found2)

	// Verify only the activated reward's expiration was removed
	pendingExpirations := suite.k.GetAllPendingTradingRewardExpirationByExpireAt(suite.ctx, 1000)
	suite.Require().Len(pendingExpirations, 1)
	suite.Require().Equal("cleanup-reward-2", pendingExpirations[0].RewardId)
}
