package keeper_test

import (
	"github.com/bze-alphateam/bze/x/rewards/types"
	"strconv"
)

func (suite *IntegrationTestSuite) TestStakingRewardParticipant() {
	list := suite.k.GetAllStakingRewardParticipant(suite.ctx)
	suite.Require().Empty(list)

	_, f := suite.k.GetStakingRewardParticipant(suite.ctx, "fake", "fake2")
	suite.Require().False(f)

	max := 10
	for i := 0; i < max; i++ {
		convI := strconv.Itoa(i)
		srp := types.StakingRewardParticipant{Address: convI, RewardId: convI}
		suite.k.SetStakingRewardParticipant(suite.ctx, srp)

		newSrp, f := suite.k.GetStakingRewardParticipant(suite.ctx, convI, convI)
		suite.Require().True(f)
		suite.Require().EqualValues(newSrp, srp)
	}

	list = suite.k.GetAllStakingRewardParticipant(suite.ctx)
	suite.Require().NotEmpty(list)
	suite.Require().EqualValues(len(list), max)

	suite.k.RemoveStakingRewardParticipant(suite.ctx, "0", "0")
	_, f = suite.k.GetStakingRewardParticipant(suite.ctx, "0", "0")
	suite.Require().False(f)

	list = suite.k.GetAllStakingRewardParticipant(suite.ctx)
	suite.Require().NotEmpty(list)
	suite.Require().EqualValues(len(list), max-1)
}
