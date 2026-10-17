package keeper_test

import (
	"cosmossdk.io/math"

	"github.com/bze-alphateam/bze/x/dao/types"
)

// A staking reward used by a DAO electorate cannot be removed; one nobody
// uses can. A Denom Reward pin never blocks a staking reward.
func (suite *IntegrationTestSuite) TestStakingRewardHooks_RemovalVeto() {
	suite.stubRewards(defaultRewardsState())
	suite.expectCreation(1)
	_, err := suite.msgServer.CreateDao(suite.ctx, validStakeDaoMsg([]string{"1"}, stakeDenom))
	suite.Require().NoError(err)

	hooks := suite.k.StakingRewardHooks()

	err = hooks.BeforeStakingRewardRemoval(suite.ctx, "1")
	suite.Require().ErrorIs(err, types.ErrSourcePinned)
	suite.Require().ErrorContains(err, "staking reward 1")

	suite.Require().NoError(hooks.BeforeStakingRewardRemoval(suite.ctx, "2"))
	suite.Require().NoError(hooks.BeforeStakingRewardRemoval(suite.ctx, stakeDenom))
}

// The participation hooks do nothing yet and never block x/rewards.
func (suite *IntegrationTestSuite) TestStakingRewardHooks_ParticipationNoOps() {
	hooks := suite.k.StakingRewardHooks()
	addr := memberA.String()

	suite.Require().NoError(hooks.AfterStakingRewardJoin(suite.ctx, "1", addr, math.NewInt(10), stakeDenom))
	suite.Require().NoError(hooks.AfterStakingRewardIncrease(suite.ctx, "1", addr, math.NewInt(5), math.NewInt(15), stakeDenom))
	suite.Require().NoError(hooks.AfterStakingRewardExit(suite.ctx, "1", addr, math.NewInt(15), stakeDenom))
}
