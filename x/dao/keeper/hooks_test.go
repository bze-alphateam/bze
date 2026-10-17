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

// Join records for a staking reward: a join while a DAO uses the source is
// recorded at the block height, a top-up keeps it, an exit deletes it. A join
// to a source nobody uses is not recorded.
func (suite *IntegrationTestSuite) TestStakingRewardHooks_JoinRecordLifecycle() {
	suite.stubRewards(defaultRewardsState())
	suite.expectCreation(1)
	_, err := suite.msgServer.CreateDao(suite.ctx, validStakeDaoMsg([]string{"1"}, stakeDenom))
	suite.Require().NoError(err)

	hooks := suite.k.StakingRewardHooks()
	source := types.StakingRewardSource("1")
	suite.ctx = suite.ctx.WithBlockHeight(7)

	suite.Require().NoError(hooks.AfterStakingRewardJoin(suite.ctx, "1", memberA.String(), math.NewInt(10), stakeDenom))
	record, found := suite.k.GetJoinRecord(suite.ctx, source, memberA)
	suite.Require().True(found)
	suite.Require().Equal(types.JoinRecord{Source: "sr/1", Address: memberA.String(), Height: 7}, record)

	suite.ctx = suite.ctx.WithBlockHeight(9)
	suite.Require().NoError(hooks.AfterStakingRewardIncrease(suite.ctx, "1", memberA.String(), math.NewInt(5), math.NewInt(15), stakeDenom))
	record, _ = suite.k.GetJoinRecord(suite.ctx, source, memberA)
	suite.Require().Equal(int64(7), record.Height)

	suite.Require().NoError(hooks.AfterStakingRewardExit(suite.ctx, "1", memberA.String(), math.NewInt(15), stakeDenom))
	_, found = suite.k.GetJoinRecord(suite.ctx, source, memberA)
	suite.Require().False(found)

	// staking reward 5 is in no electorate
	suite.Require().NoError(hooks.AfterStakingRewardJoin(suite.ctx, "5", memberA.String(), math.NewInt(10), stakeDenom))
	_, found = suite.k.GetJoinRecord(suite.ctx, types.StakingRewardSource("5"), memberA)
	suite.Require().False(found)

	// an exit without a record is fine
	suite.Require().NoError(hooks.AfterStakingRewardExit(suite.ctx, "5", memberA.String(), math.NewInt(10), stakeDenom))
}

// The same lifecycle for a Denom Reward.
func (suite *IntegrationTestSuite) TestDenomRewardHooks_JoinRecordLifecycle() {
	suite.stubRewards(defaultRewardsState())
	suite.expectCreation(1)
	_, err := suite.msgServer.CreateDao(suite.ctx, validStakeDaoMsg([]string{"1"}, stakeDenom))
	suite.Require().NoError(err)

	hooks := suite.k.DenomRewardHooks()
	source := types.DenomRewardSource(stakeDenom)
	suite.ctx = suite.ctx.WithBlockHeight(7)

	suite.Require().NoError(hooks.AfterDenomRewardJoin(suite.ctx, stakeDenom, memberB.String(), math.NewInt(10)))
	record, found := suite.k.GetJoinRecord(suite.ctx, source, memberB)
	suite.Require().True(found)
	suite.Require().Equal(types.JoinRecord{Source: "dr/ubze", Address: memberB.String(), Height: 7}, record)

	suite.ctx = suite.ctx.WithBlockHeight(9)
	suite.Require().NoError(hooks.AfterDenomRewardIncrease(suite.ctx, stakeDenom, memberB.String(), math.NewInt(5), math.NewInt(15)))
	record, _ = suite.k.GetJoinRecord(suite.ctx, source, memberB)
	suite.Require().Equal(int64(7), record.Height)

	suite.Require().NoError(hooks.AfterDenomRewardExit(suite.ctx, stakeDenom, memberB.String(), math.NewInt(15)))
	_, found = suite.k.GetJoinRecord(suite.ctx, source, memberB)
	suite.Require().False(found)

	// the uother Denom Reward is in no electorate
	suite.Require().NoError(hooks.AfterDenomRewardJoin(suite.ctx, otherDenom, memberB.String(), math.NewInt(10)))
	_, found = suite.k.GetJoinRecord(suite.ctx, types.DenomRewardSource(otherDenom), memberB)
	suite.Require().False(found)
}

// The After* hooks would abort the x/rewards transaction on error, so an
// address this module can't parse is skipped instead.
func (suite *IntegrationTestSuite) TestRewardHooks_InvalidAddressNeverFails() {
	suite.stubRewards(defaultRewardsState())
	suite.expectCreation(1)
	_, err := suite.msgServer.CreateDao(suite.ctx, validStakeDaoMsg([]string{"1"}, stakeDenom))
	suite.Require().NoError(err)

	sr := suite.k.StakingRewardHooks()
	dr := suite.k.DenomRewardHooks()
	suite.Require().NoError(sr.AfterStakingRewardJoin(suite.ctx, "1", "bze1nope", math.NewInt(1), stakeDenom))
	suite.Require().NoError(sr.AfterStakingRewardExit(suite.ctx, "1", "bze1nope", math.NewInt(1), stakeDenom))
	suite.Require().NoError(dr.AfterDenomRewardJoin(suite.ctx, stakeDenom, "bze1nope", math.NewInt(1)))
	suite.Require().NoError(dr.AfterDenomRewardExit(suite.ctx, stakeDenom, "bze1nope", math.NewInt(1)))
}
