package keeper_test

import (
	"github.com/bze-alphateam/bze/x/rewards/types"
	"strconv"
)

func (suite *IntegrationTestSuite) TestPendingTradingReward() {
	list := suite.k.GetAllPendingTradingReward(suite.ctx)
	suite.Require().Empty(list)

	_, f := suite.k.GetPendingTradingReward(suite.ctx, "fake")
	suite.Require().False(f)

	counter := suite.k.GetTradingRewardsCounter(suite.ctx)
	suite.Require().EqualValues(counter, 0)

	max := 10
	for i := 0; i < max; i++ {
		tr := types.TradingReward{RewardId: strconv.Itoa(i), Slots: uint32(i)}
		suite.k.SetPendingTradingReward(suite.ctx, tr)

		newSr, f := suite.k.GetPendingTradingReward(suite.ctx, tr.RewardId)
		suite.Require().True(f)
		suite.Require().EqualValues(newSr, tr)
	}

	list = suite.k.GetAllPendingTradingReward(suite.ctx)
	suite.Require().NotEmpty(list)
	suite.Require().EqualValues(len(list), max)

	counter = suite.k.GetTradingRewardsCounter(suite.ctx)
	suite.Require().EqualValues(counter, max)

	suite.k.RemovePendingTradingReward(suite.ctx, "0")
	_, f = suite.k.GetPendingTradingReward(suite.ctx, "0")
	suite.Require().False(f)

	list = suite.k.GetAllPendingTradingReward(suite.ctx)
	suite.Require().NotEmpty(list)
	suite.Require().EqualValues(len(list), max-1)
}

func (suite *IntegrationTestSuite) TestActiveTradingReward() {
	list := suite.k.GetAllActiveTradingReward(suite.ctx)
	suite.Require().Empty(list)

	_, f := suite.k.GetActiveTradingReward(suite.ctx, "fake")
	suite.Require().False(f)

	counter := suite.k.GetTradingRewardsCounter(suite.ctx)
	suite.Require().EqualValues(counter, 0)

	max := 10
	for i := 0; i < max; i++ {
		tr := types.TradingReward{RewardId: strconv.Itoa(i), Slots: uint32(i)}
		suite.k.SetActiveTradingReward(suite.ctx, tr)

		newSr, f := suite.k.GetActiveTradingReward(suite.ctx, tr.RewardId)
		suite.Require().True(f)
		suite.Require().EqualValues(newSr, tr)
	}

	list = suite.k.GetAllActiveTradingReward(suite.ctx)
	suite.Require().NotEmpty(list)
	suite.Require().EqualValues(len(list), max)

	counter = suite.k.GetTradingRewardsCounter(suite.ctx)
	suite.Require().EqualValues(counter, max)

	suite.k.RemoveActiveTradingReward(suite.ctx, "0")
	_, f = suite.k.GetActiveTradingReward(suite.ctx, "0")
	suite.Require().False(f)

	list = suite.k.GetAllActiveTradingReward(suite.ctx)
	suite.Require().NotEmpty(list)
	suite.Require().EqualValues(len(list), max-1)
}

func (suite *IntegrationTestSuite) TestMarketIdRewardId() {
	list := suite.k.GetAllMarketIdRewardId(suite.ctx)
	suite.Require().Empty(list)

	_, f := suite.k.GetMarketIdRewardId(suite.ctx, "fake")
	suite.Require().False(f)

	max := 10
	for i := 0; i < max; i++ {
		mid := types.MarketIdTradingRewardId{MarketId: strconv.Itoa(i)}
		suite.k.SetMarketIdRewardId(suite.ctx, mid)

		newSr, f := suite.k.GetMarketIdRewardId(suite.ctx, mid.MarketId)
		suite.Require().True(f)
		suite.Require().EqualValues(newSr, mid)
	}

	list = suite.k.GetAllMarketIdRewardId(suite.ctx)
	suite.Require().NotEmpty(list)
	suite.Require().EqualValues(len(list), max)

	suite.k.RemoveMarketIdRewardId(suite.ctx, "0")
	_, f = suite.k.GetMarketIdRewardId(suite.ctx, "0")
	suite.Require().False(f)

	list = suite.k.GetAllMarketIdRewardId(suite.ctx)
	suite.Require().NotEmpty(list)
	suite.Require().EqualValues(len(list), max-1)
}
