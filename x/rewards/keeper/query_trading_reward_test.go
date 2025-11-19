package keeper_test

import (
	"github.com/bze-alphateam/bze/x/rewards/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (suite *IntegrationTestSuite) TestQueryTradingReward_TradingRewardPending() {
	tradingReward := types.TradingReward{
		RewardId:    "pending-query-reward",
		PrizeAmount: "1000",
		PrizeDenom:  "ubze",
		Duration:    30,
		MarketId:    "market-1",
		Slots:       5,
		ExpireAt:    1000,
	}

	suite.k.SetPendingTradingReward(suite.ctx, tradingReward)

	req := &types.QueryTradingRewardRequest{
		RewardId: "pending-query-reward",
	}

	response, err := suite.k.TradingReward(suite.ctx, req)
	suite.Require().NoError(err)
	suite.Require().NotNil(response)
	suite.Require().Equal(tradingReward.RewardId, response.TradingReward.RewardId)
	suite.Require().Equal(tradingReward.PrizeAmount, response.TradingReward.PrizeAmount)
	suite.Require().Equal(tradingReward.PrizeDenom, response.TradingReward.PrizeDenom)
	suite.Require().Equal(tradingReward.Duration, response.TradingReward.Duration)
	suite.Require().Equal(tradingReward.MarketId, response.TradingReward.MarketId)
	suite.Require().Equal(tradingReward.Slots, response.TradingReward.Slots)
	suite.Require().Equal(tradingReward.ExpireAt, response.TradingReward.ExpireAt)
}

func (suite *IntegrationTestSuite) TestQueryTradingReward_TradingRewardActive() {
	tradingReward := types.TradingReward{
		RewardId:    "active-query-reward",
		PrizeAmount: "2000",
		PrizeDenom:  "utoken",
		Duration:    60,
		MarketId:    "market-2",
		Slots:       10,
		ExpireAt:    2000,
	}

	suite.k.SetActiveTradingReward(suite.ctx, tradingReward)

	req := &types.QueryTradingRewardRequest{
		RewardId: "active-query-reward",
	}

	response, err := suite.k.TradingReward(suite.ctx, req)
	suite.Require().NoError(err)
	suite.Require().NotNil(response)
	suite.Require().Equal(tradingReward.RewardId, response.TradingReward.RewardId)
	suite.Require().Equal(tradingReward.PrizeAmount, response.TradingReward.PrizeAmount)
	suite.Require().Equal(tradingReward.PrizeDenom, response.TradingReward.PrizeDenom)
	suite.Require().Equal(tradingReward.Duration, response.TradingReward.Duration)
	suite.Require().Equal(tradingReward.MarketId, response.TradingReward.MarketId)
	suite.Require().Equal(tradingReward.Slots, response.TradingReward.Slots)
	suite.Require().Equal(tradingReward.ExpireAt, response.TradingReward.ExpireAt)
}

func (suite *IntegrationTestSuite) TestQueryTradingReward_TradingRewardPendingPriority() {
	// When same reward exists in both pending and active, pending should be returned first
	pendingReward := types.TradingReward{
		RewardId:    "priority-reward",
		PrizeAmount: "1000",
		PrizeDenom:  "ubze",
		Duration:    30,
		MarketId:    "market-1",
		Slots:       5,
		ExpireAt:    1000,
	}

	activeReward := types.TradingReward{
		RewardId:    "priority-reward",
		PrizeAmount: "2000",
		PrizeDenom:  "utoken",
		Duration:    60,
		MarketId:    "market-2",
		Slots:       10,
		ExpireAt:    2000,
	}

	suite.k.SetPendingTradingReward(suite.ctx, pendingReward)
	suite.k.SetActiveTradingReward(suite.ctx, activeReward)

	req := &types.QueryTradingRewardRequest{
		RewardId: "priority-reward",
	}

	response, err := suite.k.TradingReward(suite.ctx, req)
	suite.Require().NoError(err)
	suite.Require().NotNil(response)
	// Should return pending reward (first checked)
	suite.Require().Equal("1000", response.TradingReward.PrizeAmount)
	suite.Require().Equal("ubze", response.TradingReward.PrizeDenom)
}

func (suite *IntegrationTestSuite) TestQueryTradingReward_TradingRewardNilRequest() {
	response, err := suite.k.TradingReward(suite.ctx, nil)

	suite.Require().Error(err)
	suite.Require().Nil(response)
	suite.Require().Equal(codes.InvalidArgument, status.Code(err))
	suite.Require().Contains(err.Error(), "invalid request")
}

func (suite *IntegrationTestSuite) TestQueryTradingReward_TradingRewardNotFound() {
	req := &types.QueryTradingRewardRequest{
		RewardId: "non-existent-reward",
	}

	response, err := suite.k.TradingReward(suite.ctx, req)

	suite.Require().Error(err)
	suite.Require().Nil(response)
	suite.Require().Equal(codes.NotFound, status.Code(err))
	suite.Require().Contains(err.Error(), "not found")
}

func (suite *IntegrationTestSuite) TestQueryTradingReward_AllTradingRewardsActive() {
	activeRewards := []types.TradingReward{
		{
			RewardId:    "active-all-1",
			PrizeAmount: "1000",
			PrizeDenom:  "ubze",
			Duration:    30,
			MarketId:    "market-1",
			Slots:       5,
			ExpireAt:    1000,
		},
		{
			RewardId:    "active-all-2",
			PrizeAmount: "2000",
			PrizeDenom:  "utoken",
			Duration:    60,
			MarketId:    "market-2",
			Slots:       10,
			ExpireAt:    2000,
		},
	}

	for _, reward := range activeRewards {
		suite.k.SetActiveTradingReward(suite.ctx, reward)
	}

	req := &types.QueryAllTradingRewardsRequest{
		State:      "active",
		Pagination: nil,
	}

	response, err := suite.k.AllTradingRewards(suite.ctx, req)
	suite.Require().NoError(err)
	suite.Require().NotNil(response)
	suite.Require().Len(response.List, 2)

	// Verify all active rewards are present
	rewardIds := make(map[string]bool)
	for _, reward := range response.List {
		rewardIds[reward.RewardId] = true
	}

	suite.Require().True(rewardIds["active-all-1"])
	suite.Require().True(rewardIds["active-all-2"])
}

func (suite *IntegrationTestSuite) TestQueryTradingReward_AllTradingRewardsPending() {
	pendingRewards := []types.TradingReward{
		{
			RewardId:    "pending-all-1",
			PrizeAmount: "1500",
			PrizeDenom:  "ucoin",
			Duration:    45,
			MarketId:    "market-3",
			Slots:       8,
			ExpireAt:    1500,
		},
		{
			RewardId:    "pending-all-2",
			PrizeAmount: "2500",
			PrizeDenom:  "uother",
			Duration:    90,
			MarketId:    "market-4",
			Slots:       12,
			ExpireAt:    2500,
		},
	}

	for _, reward := range pendingRewards {
		suite.k.SetPendingTradingReward(suite.ctx, reward)
	}

	req := &types.QueryAllTradingRewardsRequest{
		State:      "pending",
		Pagination: nil,
	}

	response, err := suite.k.AllTradingRewards(suite.ctx, req)
	suite.Require().NoError(err)
	suite.Require().NotNil(response)
	suite.Require().Len(response.List, 2)

	// Verify all pending rewards are present
	rewardIds := make(map[string]bool)
	for _, reward := range response.List {
		rewardIds[reward.RewardId] = true
	}

	suite.Require().True(rewardIds["pending-all-1"])
	suite.Require().True(rewardIds["pending-all-2"])
}

func (suite *IntegrationTestSuite) TestQueryTradingReward_AllTradingRewardsDefaultActive() {
	activeReward := types.TradingReward{
		RewardId:    "default-active",
		PrizeAmount: "1000",
		PrizeDenom:  "ubze",
		Duration:    30,
		MarketId:    "market-1",
		Slots:       5,
		ExpireAt:    1000,
	}

	suite.k.SetActiveTradingReward(suite.ctx, activeReward)

	req := &types.QueryAllTradingRewardsRequest{
		State:      "", // Empty state defaults to active
		Pagination: nil,
	}

	response, err := suite.k.AllTradingRewards(suite.ctx, req)
	suite.Require().NoError(err)
	suite.Require().NotNil(response)
	suite.Require().Len(response.List, 1)
	suite.Require().Equal("default-active", response.List[0].RewardId)
}

func (suite *IntegrationTestSuite) TestQueryTradingReward_AllTradingRewardsNilRequest() {
	response, err := suite.k.AllTradingRewards(suite.ctx, nil)

	suite.Require().Error(err)
	suite.Require().Nil(response)
	suite.Require().Equal(codes.InvalidArgument, status.Code(err))
	suite.Require().Contains(err.Error(), "invalid request")
}

func (suite *IntegrationTestSuite) TestQueryTradingReward_AllTradingRewardsEmpty() {
	req := &types.QueryAllTradingRewardsRequest{
		State:      "active",
		Pagination: nil,
	}

	response, err := suite.k.AllTradingRewards(suite.ctx, req)
	suite.Require().NoError(err)
	suite.Require().NotNil(response)
	suite.Require().Empty(response.List)
}

func (suite *IntegrationTestSuite) TestQueryTradingReward_AllTradingRewardsPagination() {
	activeRewards := []types.TradingReward{
		{
			RewardId:    "page-active-1",
			PrizeAmount: "1000",
			PrizeDenom:  "ubze",
			Duration:    30,
			MarketId:    "market-1",
			Slots:       5,
			ExpireAt:    1000,
		},
		{
			RewardId:    "page-active-2",
			PrizeAmount: "2000",
			PrizeDenom:  "utoken",
			Duration:    60,
			MarketId:    "market-2",
			Slots:       10,
			ExpireAt:    2000,
		},
	}

	for _, reward := range activeRewards {
		suite.k.SetActiveTradingReward(suite.ctx, reward)
	}

	req := &types.QueryAllTradingRewardsRequest{
		State: "active",
		Pagination: &query.PageRequest{
			Limit: 1,
		},
	}

	response, err := suite.k.AllTradingRewards(suite.ctx, req)
	suite.Require().NoError(err)
	suite.Require().NotNil(response)
	suite.Require().Len(response.List, 1)
	suite.Require().NotNil(response.Pagination)
}

func (suite *IntegrationTestSuite) TestQueryTradingReward_TradingRewardLeaderboard() {
	leaderboard := types.TradingRewardLeaderboard{
		RewardId: "leaderboard-query-reward",
		List: []types.TradingRewardLeaderboardEntry{
			{
				Amount:    "5000",
				Address:   "bze1leader1",
				CreatedAt: 1000,
			},
			{
				Amount:    "3000",
				Address:   "bze1leader2",
				CreatedAt: 2000,
			},
		},
	}

	suite.k.SetTradingRewardLeaderboard(suite.ctx, leaderboard)

	req := &types.QueryTradingRewardLeaderboardRequest{
		RewardId: "leaderboard-query-reward",
	}

	response, err := suite.k.TradingRewardLeaderboard(suite.ctx, req)
	suite.Require().NoError(err)
	suite.Require().NotNil(response)
	suite.Require().NotNil(response.Leaderboard)
	suite.Require().Equal(leaderboard.RewardId, response.Leaderboard.RewardId)
	suite.Require().Len(response.Leaderboard.List, 2)
	suite.Require().Equal("bze1leader1", response.Leaderboard.List[0].Address)
	suite.Require().Equal("5000", response.Leaderboard.List[0].Amount)
}

func (suite *IntegrationTestSuite) TestQueryTradingReward_TradingRewardLeaderboardNilRequest() {
	response, err := suite.k.TradingRewardLeaderboard(suite.ctx, nil)

	suite.Require().Error(err)
	suite.Require().Nil(response)
	suite.Require().Equal(codes.InvalidArgument, status.Code(err))
	suite.Require().Contains(err.Error(), "invalid request")
}

func (suite *IntegrationTestSuite) TestQueryTradingReward_TradingRewardLeaderboardNotFound() {
	req := &types.QueryTradingRewardLeaderboardRequest{
		RewardId: "non-existent-leaderboard",
	}

	response, err := suite.k.TradingRewardLeaderboard(suite.ctx, req)

	suite.Require().Error(err)
	suite.Require().Nil(response)
	suite.Require().Equal(codes.NotFound, status.Code(err))
	suite.Require().Contains(err.Error(), "not found")
}

func (suite *IntegrationTestSuite) TestQueryTradingReward_MarketTradingReward() {
	marketReward := types.MarketIdTradingRewardId{
		RewardId: "market-query-reward",
		MarketId: "query-market-1",
	}

	suite.k.SetMarketIdRewardId(suite.ctx, marketReward)

	req := &types.QueryMarketTradingRewardRequest{
		MarketId: "query-market-1",
	}

	response, err := suite.k.MarketTradingReward(suite.ctx, req)
	suite.Require().NoError(err)
	suite.Require().NotNil(response)
	suite.Require().NotNil(response.MarketReward)
	suite.Require().Equal(marketReward.RewardId, response.MarketReward.RewardId)
	suite.Require().Equal(marketReward.MarketId, response.MarketReward.MarketId)
}

func (suite *IntegrationTestSuite) TestQueryTradingReward_MarketTradingRewardNilRequest() {
	response, err := suite.k.MarketTradingReward(suite.ctx, nil)

	suite.Require().Error(err)
	suite.Require().Nil(response)
	suite.Require().Equal(codes.InvalidArgument, status.Code(err))
	suite.Require().Contains(err.Error(), "invalid request")
}

func (suite *IntegrationTestSuite) TestQueryTradingReward_MarketTradingRewardNotFound() {
	req := &types.QueryMarketTradingRewardRequest{
		MarketId: "non-existent-market",
	}

	response, err := suite.k.MarketTradingReward(suite.ctx, req)

	suite.Require().Error(err)
	suite.Require().Nil(response)
	suite.Require().Equal(codes.NotFound, status.Code(err))
	suite.Require().Contains(err.Error(), "not found")
}
