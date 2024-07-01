package keeper_test

import (
	"github.com/bze-alphateam/bze/x/rewards/types"
)

func (suite *IntegrationTestSuite) TestHandleActivateTradingRewardProposal_AlreadyActiveError() {
	tr := types.TradingReward{RewardId: "123"}
	suite.k.SetActiveTradingReward(suite.ctx, tr)

	prop := types.ActivateTradingRewardProposal{
		Title:       "",
		Description: "",
		RewardId:    tr.RewardId,
	}

	suite.Require().Error(suite.k.HandleActivateTradingRewardProposal(suite.ctx, &prop))
}

func (suite *IntegrationTestSuite) TestHandleActivateTradingRewardProposal_TradingRewardNotFoundError() {
	prop := types.ActivateTradingRewardProposal{
		Title:       "",
		Description: "",
		RewardId:    "0",
	}

	suite.Require().Error(suite.k.HandleActivateTradingRewardProposal(suite.ctx, &prop))
}

func (suite *IntegrationTestSuite) TestHandleActivateTradingRewardProposal_Success() {
	tr := types.TradingReward{
		RewardId: "123",
		ExpireAt: 123,
		MarketId: "da",
	}
	suite.k.SetPendingTradingReward(suite.ctx, tr)

	trExp := types.TradingRewardExpiration{
		RewardId: tr.RewardId,
		ExpireAt: tr.ExpireAt,
	}
	suite.k.SetPendingTradingRewardExpiration(suite.ctx, trExp)

	prop := types.ActivateTradingRewardProposal{
		Title:       "",
		Description: "",
		RewardId:    tr.RewardId,
	}

	suite.Require().NoError(suite.k.HandleActivateTradingRewardProposal(suite.ctx, &prop))

	//check the reward is not in pending anymore
	_, found := suite.k.GetPendingTradingReward(suite.ctx, tr.RewardId)
	suite.Require().False(found)

	//check the expiration does not exist
	allPendingExp := suite.k.GetAllPendingTradingRewardExpiration(suite.ctx)
	suite.Require().Empty(allPendingExp)

	//check it is active
	activeTr, found := suite.k.GetActiveTradingReward(suite.ctx, tr.RewardId)
	suite.Require().True(found)
	//check the expire at was modified in the document
	suite.Require().NotEqual(activeTr.ExpireAt, tr.ExpireAt)

	//check there's an active exp
	allActiveExp := suite.k.GetAllActiveTradingRewardExpiration(suite.ctx)
	suite.Require().Equal(len(allActiveExp), 1)
	//check the reward id exists in active exp
	suite.Require().Equal(allActiveExp[0].RewardId, tr.RewardId)

	//check we can find the TR by market id
	_, found = suite.k.GetMarketIdRewardId(suite.ctx, tr.MarketId)
	suite.Require().True(found)
}
