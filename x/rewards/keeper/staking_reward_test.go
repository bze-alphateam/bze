package keeper_test

import (
	"github.com/bze-alphateam/bze/x/rewards/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"strconv"
)

func (suite *IntegrationTestSuite) TestStakingReward() {
	list := suite.k.GetAllStakingReward(suite.ctx)
	suite.Require().Empty(list)

	_, f := suite.k.GetStakingReward(suite.ctx, "fake")
	suite.Require().False(f)

	counter := suite.k.GetStakingRewardsCounter(suite.ctx)
	suite.Require().EqualValues(counter, 0)

	max := 10
	for i := 0; i < max; i++ {
		sr := types.StakingReward{RewardId: strconv.Itoa(i), Lock: uint32(i)}
		suite.k.SetStakingReward(suite.ctx, sr)

		newSr, f := suite.k.GetStakingReward(suite.ctx, sr.RewardId)
		suite.Require().True(f)
		suite.Require().EqualValues(newSr, sr)
	}

	list = suite.k.GetAllStakingReward(suite.ctx)
	suite.Require().NotEmpty(list)
	suite.Require().EqualValues(len(list), max)

	suite.k.IterateAllStakingRewards(suite.ctx, func(ctx sdk.Context, sr types.StakingReward) (stop bool) {
		suite.Require().LessOrEqual(sr.Lock, uint32(max))

		return false
	})

	counter = suite.k.GetStakingRewardsCounter(suite.ctx)
	suite.Require().EqualValues(counter, max)

	suite.k.RemoveStakingReward(suite.ctx, "0")
	_, f = suite.k.GetStakingReward(suite.ctx, "0")
	suite.Require().False(f)

	list = suite.k.GetAllStakingReward(suite.ctx)
	suite.Require().NotEmpty(list)
	suite.Require().EqualValues(len(list), max-1)
}
