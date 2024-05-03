package keeper_test

import (
	"github.com/bze-alphateam/bze/x/rewards/types"
	"strconv"
)

func (suite *IntegrationTestSuite) TestPendingTradingRewardExpiration() {
	list := suite.k.GetAllPendingTradingRewardExpiration(suite.ctx)
	suite.Require().Empty(list)

	max := 10
	for i := 1; i <= max; i++ {
		ptre := types.TradingRewardExpiration{RewardId: "01", ExpireAt: uint32(i)}
		suite.k.SetPendingTradingRewardExpiration(suite.ctx, ptre)

		ptre = types.TradingRewardExpiration{RewardId: strconv.Itoa(i), ExpireAt: uint32(i)}
		suite.k.SetPendingTradingRewardExpiration(suite.ctx, ptre)

		list = suite.k.GetAllPendingTradingRewardExpiration(suite.ctx)
		suite.Require().EqualValues(len(list), i*2)

		//just check the other list is not altered
		list = suite.k.GetAllActiveTradingRewardExpiration(suite.ctx)
		suite.Require().Empty(list)

		list = suite.k.GetAllPendingTradingRewardExpirationByExpireAt(suite.ctx, uint32(i))
		suite.Require().EqualValues(len(list), 2)
	}

	suite.k.RemovePendingTradingRewardExpiration(suite.ctx, 1, "1")
	list = suite.k.GetAllPendingTradingRewardExpirationByExpireAt(suite.ctx, uint32(1))
	suite.Require().EqualValues(len(list), 1)
}

func (suite *IntegrationTestSuite) TestActiveTradingRewardExpiration() {
	list := suite.k.GetAllActiveTradingRewardExpiration(suite.ctx)
	suite.Require().Empty(list)

	max := 10
	for i := 1; i <= max; i++ {
		ptre := types.TradingRewardExpiration{RewardId: "01", ExpireAt: uint32(i)}
		suite.k.SetActiveTradingRewardExpiration(suite.ctx, ptre)

		ptre = types.TradingRewardExpiration{RewardId: strconv.Itoa(i), ExpireAt: uint32(i)}
		suite.k.SetActiveTradingRewardExpiration(suite.ctx, ptre)

		list = suite.k.GetAllActiveTradingRewardExpiration(suite.ctx)
		suite.Require().EqualValues(len(list), i*2)

		//just check the other list is not altered
		list = suite.k.GetAllPendingTradingRewardExpiration(suite.ctx)
		suite.Require().Empty(list)

		list = suite.k.GetAllActiveTradingRewardExpirationByExpireAt(suite.ctx, uint32(i))
		suite.Require().EqualValues(len(list), 2)
	}

	suite.k.RemoveActiveTradingRewardExpiration(suite.ctx, 1, "1")
	list = suite.k.GetAllActiveTradingRewardExpirationByExpireAt(suite.ctx, uint32(1))
	suite.Require().EqualValues(len(list), 1)
}
