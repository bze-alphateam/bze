package keeper_test

import (
	"strings"

	"cosmossdk.io/math"

	"github.com/bze-alphateam/bze/x/dao/types"
)

// x/rewards reports addresses in their standard lowercase spelling. An Exit
// hook carrying any other spelling of an address (bech32 also decodes the
// all-uppercase form to the same account) is ignored: the join records,
// ballots and tallies of the account stay as they are.
func (suite *IntegrationTestSuite) TestExitHook_NonStandardAddressIgnored() {
	s := suite.createStakeDao()
	s.joinSr("1", memberA, 30)
	s.joinDr(stakeDenom, memberA, 20)
	suite.atBlock(5)
	suite.Require().NoError(suite.k.StakingRewardHooks().AfterStakingRewardJoin(suite.ctx, "1", memberA.String(), math.NewInt(30), stakeDenom))
	suite.Require().NoError(suite.k.DenomRewardHooks().AfterDenomRewardJoin(suite.ctx, stakeDenom, memberA.String(), math.NewInt(20)))

	suite.atBlock(10)
	id, err := suite.submit(memberA, 1, false)
	suite.Require().NoError(err)
	suite.Require().NoError(suite.vote(memberA, id, types.VoteOption_YES))
	before, found := suite.k.GetProposal(suite.ctx, id)
	suite.Require().True(found)

	upper := strings.ToUpper(memberA.String())
	suite.Require().NoError(suite.k.StakingRewardHooks().AfterStakingRewardExit(suite.ctx, "1", upper, math.NewInt(30), stakeDenom))
	suite.Require().NoError(suite.k.DenomRewardHooks().AfterDenomRewardExit(suite.ctx, stakeDenom, upper, math.NewInt(20)))

	srRecord, found := suite.k.GetJoinRecord(suite.ctx, srSource, memberA)
	suite.Require().True(found)
	suite.Require().Equal(int64(5), srRecord.Height)
	drRecord, found := suite.k.GetJoinRecord(suite.ctx, drSource, memberA)
	suite.Require().True(found)
	suite.Require().Equal(int64(5), drRecord.Height)
	suite.Require().True(suite.k.HasVoteIndex(suite.ctx, srSource, memberA, id))
	suite.Require().True(suite.k.HasVoteIndex(suite.ctx, drSource, memberA, id))
	after, _ := suite.k.GetProposal(suite.ctx, id)
	suite.Require().True(before.Yes.IsPositive())
	suite.Require().Equal(before.Yes, after.Yes)
	suite.Require().Equal(0, suite.ballotReducedEvents())

	// the standard spelling still exits
	suite.Require().NoError(suite.k.StakingRewardHooks().AfterStakingRewardExit(suite.ctx, "1", memberA.String(), math.NewInt(30), stakeDenom))
	_, found = suite.k.GetJoinRecord(suite.ctx, srSource, memberA)
	suite.Require().False(found)
	suite.Require().Equal(1, suite.ballotReducedEvents())
}

// A Join or top-up reported with a non-standard spelling still refreshes the
// decoded account's join record: a newer join height only ever makes stake
// ineligible for proposals already open.
func (suite *IntegrationTestSuite) TestJoinHook_NonStandardAddressStillRecordsJoin() {
	s := suite.createStakeDao()
	s.joinSr("1", memberA, 30)
	s.joinDr(stakeDenom, memberA, 20)
	suite.atBlock(5)
	suite.Require().NoError(suite.k.StakingRewardHooks().AfterStakingRewardJoin(suite.ctx, "1", memberA.String(), math.NewInt(30), stakeDenom))
	suite.Require().NoError(suite.k.DenomRewardHooks().AfterDenomRewardJoin(suite.ctx, stakeDenom, memberA.String(), math.NewInt(20)))

	upper := strings.ToUpper(memberA.String())
	suite.atBlock(8)
	suite.Require().NoError(suite.k.StakingRewardHooks().AfterStakingRewardIncrease(suite.ctx, "1", upper, math.NewInt(1), math.NewInt(31), stakeDenom))
	suite.Require().NoError(suite.k.DenomRewardHooks().AfterDenomRewardJoin(suite.ctx, stakeDenom, upper, math.NewInt(1)))

	srRecord, found := suite.k.GetJoinRecord(suite.ctx, srSource, memberA)
	suite.Require().True(found)
	suite.Require().Equal(int64(8), srRecord.Height)
	drRecord, found := suite.k.GetJoinRecord(suite.ctx, drSource, memberA)
	suite.Require().True(found)
	suite.Require().Equal(int64(8), drRecord.Height)
}
