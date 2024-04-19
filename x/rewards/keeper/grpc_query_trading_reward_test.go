package keeper_test

import (
	"github.com/bze-alphateam/bze/x/rewards/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (suite *IntegrationTestSuite) TestQueryTradingRewardAll_Success_EmptyList() {
	goCtx := sdk.WrapSDKContext(suite.ctx)

	l, err := suite.k.TradingRewardAll(goCtx, &types.QueryAllTradingRewardRequest{})
	suite.Require().NoError(err)
	suite.Require().Empty(l.List)
}

func (suite *IntegrationTestSuite) TestQueryTradingRewardAll_InvalidRequest() {
	goCtx := sdk.WrapSDKContext(suite.ctx)
	_, err := suite.k.TradingRewardAll(goCtx, nil)
	suite.Require().Error(err)
}

func (suite *IntegrationTestSuite) TestQueryTradingRewardAll_Pending_Success() {
	goCtx := sdk.WrapSDKContext(suite.ctx)

	tr := types.TradingReward{
		RewardId: "2132",
	}
	suite.k.SetPendingTradingReward(suite.ctx, tr)

	l, err := suite.k.TradingRewardAll(goCtx, &types.QueryAllTradingRewardRequest{State: "pending"})
	suite.Require().NoError(err)
	suite.Require().NotEmpty(l.List)
	suite.Require().EqualValues(l.List[0], tr)
}

func (suite *IntegrationTestSuite) TestQueryTradingRewardAll_Active_Success() {
	goCtx := sdk.WrapSDKContext(suite.ctx)

	tr := types.TradingReward{
		RewardId: "2132",
	}
	suite.k.SetActiveTradingReward(suite.ctx, tr)

	shouldBeEmpty, err := suite.k.TradingRewardAll(goCtx, &types.QueryAllTradingRewardRequest{State: "pending"})
	suite.Require().Empty(shouldBeEmpty.List)

	l, err := suite.k.TradingRewardAll(goCtx, &types.QueryAllTradingRewardRequest{State: "active"})
	suite.Require().NoError(err)
	suite.Require().NotEmpty(l.List)
	suite.Require().EqualValues(l.List[0], tr)
}

func (suite *IntegrationTestSuite) TestQueryTradingReward_InvalidRequest() {
	goCtx := sdk.WrapSDKContext(suite.ctx)
	_, err := suite.k.TradingReward(goCtx, nil)
	suite.Require().Error(err)
}

func (suite *IntegrationTestSuite) TestQueryTradingReward_NotFound() {
	goCtx := sdk.WrapSDKContext(suite.ctx)
	_, err := suite.k.TradingReward(goCtx, &types.QueryGetTradingRewardRequest{RewardId: "da"})
	suite.Require().Error(err)
}

func (suite *IntegrationTestSuite) TestQueryTradingReward_Pending_Success() {
	goCtx := sdk.WrapSDKContext(suite.ctx)

	tr := types.TradingReward{
		RewardId: "2132",
	}
	suite.k.SetPendingTradingReward(suite.ctx, tr)

	resp, err := suite.k.TradingReward(goCtx, &types.QueryGetTradingRewardRequest{RewardId: tr.RewardId})
	suite.Require().NoError(err)
	suite.Require().EqualValues(resp.TradingReward, tr)
}

func (suite *IntegrationTestSuite) TestQueryTradingReward_Active_Success() {
	goCtx := sdk.WrapSDKContext(suite.ctx)

	tr := types.TradingReward{
		RewardId: "2132",
	}
	suite.k.SetActiveTradingReward(suite.ctx, tr)

	resp, err := suite.k.TradingReward(goCtx, &types.QueryGetTradingRewardRequest{RewardId: tr.RewardId})
	suite.Require().NoError(err)
	suite.Require().EqualValues(resp.TradingReward, tr)
}

func (suite *IntegrationTestSuite) TestQueryGetMarketIdTradingRewardIdHandler_InvalidRequest() {
	goCtx := sdk.WrapSDKContext(suite.ctx)
	_, err := suite.k.GetMarketIdTradingRewardIdHandler(goCtx, nil)
	suite.Require().Error(err)
}

func (suite *IntegrationTestSuite) TestQueryGetMarketIdTradingRewardIdHandler_NotFound() {
	goCtx := sdk.WrapSDKContext(suite.ctx)
	_, err := suite.k.GetMarketIdTradingRewardIdHandler(goCtx, &types.QueryGetMarketIdTradingRewardIdHandlerRequest{MarketId: "s"})
	suite.Require().Error(err)
}

func (suite *IntegrationTestSuite) TestQueryGetMarketIdTradingRewardIdHandler_Success() {
	goCtx := sdk.WrapSDKContext(suite.ctx)

	mid := types.MarketIdTradingRewardId{
		RewardId: "123",
		MarketId: "asd/dsa",
	}

	suite.k.SetMarketIdRewardId(suite.ctx, mid)

	resp, err := suite.k.GetMarketIdTradingRewardIdHandler(goCtx, &types.QueryGetMarketIdTradingRewardIdHandlerRequest{MarketId: mid.MarketId})
	suite.Require().NoError(err)
	suite.Require().EqualValues(resp.MarketIdRewardId, &mid)
}

func (suite *IntegrationTestSuite) TestQueryGetTradingRewardLeaderboardHandler_InvalidRequest() {
	goCtx := sdk.WrapSDKContext(suite.ctx)
	_, err := suite.k.GetTradingRewardLeaderboardHandler(goCtx, nil)
	suite.Require().Error(err)
}

func (suite *IntegrationTestSuite) TestQueryGetTradingRewardLeaderboardHandler_NotFound() {
	goCtx := sdk.WrapSDKContext(suite.ctx)
	_, err := suite.k.GetTradingRewardLeaderboardHandler(goCtx, &types.QueryGetTradingRewardLeaderboardRequest{RewardId: "a"})
	suite.Require().Error(err)
}

func (suite *IntegrationTestSuite) TestQueryGetTradingRewardLeaderboardHandler_Success() {
	goCtx := sdk.WrapSDKContext(suite.ctx)

	lb := types.TradingRewardLeaderboard{
		RewardId: "123",
	}

	suite.k.SetTradingRewardLeaderboard(suite.ctx, lb)

	resp, err := suite.k.GetTradingRewardLeaderboardHandler(goCtx, &types.QueryGetTradingRewardLeaderboardRequest{RewardId: lb.RewardId})
	suite.Require().NoError(err)
	suite.Require().EqualValues(resp.Leaderboard, &lb)
}
