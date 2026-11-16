package keeper_test

import (
	"cosmossdk.io/math"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/bze-alphateam/bze/x/dao/types"
)

// A ballot is found by proposal and voter; any other pair is NotFound.
func (suite *IntegrationTestSuite) TestQueryBallot() {
	suite.createStaticDao(noConfigChange)
	id, err := suite.submit(memberA, 1, false)
	suite.Require().NoError(err)
	suite.Require().NoError(suite.vote(memberA, id, types.VoteOption_YES))

	res, err := suite.k.Ballot(suite.ctx, &types.QueryBallotRequest{ProposalId: id, Voter: memberA.String()})
	suite.Require().NoError(err)
	stored, found := suite.k.GetBallot(suite.ctx, id, memberA)
	suite.Require().True(found)
	suite.Require().Equal(stored, res.Ballot)
	suite.Require().Equal(types.VoteOption_YES, res.Ballot.Option)
	suite.Require().Equal(math.OneInt(), res.Ballot.Weight)

	for _, req := range []*types.QueryBallotRequest{
		{ProposalId: id, Voter: memberB.String()},
		{ProposalId: id + 1, Voter: memberA.String()},
	} {
		_, err = suite.k.Ballot(suite.ctx, req)
		suite.Require().Equal(codes.NotFound, status.Code(err))
	}

	_, err = suite.k.Ballot(suite.ctx, &types.QueryBallotRequest{ProposalId: id, Voter: "bad"})
	suite.Require().Equal(codes.InvalidArgument, status.Code(err))
	_, err = suite.k.Ballot(suite.ctx, nil)
	suite.Require().Equal(codes.InvalidArgument, status.Code(err))
}

func (suite *IntegrationTestSuite) votingPower(daoId uint64, addr string) *types.QueryVotingPowerResponse {
	res, err := suite.k.VotingPower(suite.ctx, &types.QueryVotingPowerRequest{DaoId: daoId, Address: addr})
	suite.Require().NoError(err)

	return res
}

// In a static-list DAO a member weighs 1 now and on each proposal in voting, a
// non-member 0; a proposal that left voting is no longer listed.
func (suite *IntegrationTestSuite) TestQueryVotingPower_Static() {
	suite.createStaticDao(noConfigChange)
	first, err := suite.submit(memberA, 1, false)
	suite.Require().NoError(err)
	second, err := suite.submit(memberA, 1, false)
	suite.Require().NoError(err)

	res := suite.votingPower(1, memberA.String())
	suite.Require().Equal(math.OneInt(), res.Current)
	suite.Require().Equal([]types.ProposalVotingPower{
		{ProposalId: first, EligibleWeight: math.OneInt()},
		{ProposalId: second, EligibleWeight: math.OneInt()},
	}, res.Proposals)

	res = suite.votingPower(1, attackerAddr.String())
	suite.Require().Equal(math.ZeroInt(), res.Current)
	suite.Require().Equal([]types.ProposalVotingPower{
		{ProposalId: first, EligibleWeight: math.ZeroInt()},
		{ProposalId: second, EligibleWeight: math.ZeroInt()},
	}, res.Proposals)

	suite.Require().NoError(suite.cancel(memberA, first))
	res = suite.votingPower(1, memberA.String())
	suite.Require().Equal([]types.ProposalVotingPower{{ProposalId: second, EligibleWeight: math.OneInt()}}, res.Proposals)
}

// In a stake DAO the current weight is all of the address's stake in the
// DAO's sources; on a proposal, stake that joined at or after its submit height
// weighs 0, and counts on a proposal submitted later.
func (suite *IntegrationTestSuite) TestQueryVotingPower_StakeIneligibleJoin() {
	s := suite.createStakeDao()
	s.joinSr("1", memberA, 30)
	s.joinDr(stakeDenom, memberA, 20)
	suite.atBlock(10)
	first, err := suite.submit(memberA, 1, false)
	suite.Require().NoError(err)

	suite.atBlock(11)
	s.joinSr("1", memberC, 40)
	suite.Require().NoError(suite.k.StakingRewardHooks().AfterStakingRewardJoin(suite.ctx, "1", memberC.String(), math.NewInt(40), stakeDenom))

	res := suite.votingPower(1, memberA.String())
	suite.Require().Equal(math.NewInt(50), res.Current)
	suite.Require().Equal([]types.ProposalVotingPower{{ProposalId: first, EligibleWeight: math.NewInt(50), Shares: []types.BallotShare{
		{Source: string(srSource), Amount: math.NewInt(30)},
		{Source: string(drSource), Amount: math.NewInt(20)},
	}}}, res.Proposals)

	// the join record makes memberC's stake ineligible on the first proposal only
	res = suite.votingPower(1, memberC.String())
	suite.Require().Equal(math.NewInt(40), res.Current)
	suite.Require().Len(res.Proposals, 1)
	suite.Require().Equal(math.ZeroInt(), res.Proposals[0].EligibleWeight)
	suite.Require().Empty(res.Proposals[0].Shares)
	suite.Require().ErrorIs(suite.vote(memberC, first, types.VoteOption_YES), types.ErrNoVotingPower)

	suite.atBlock(12)
	second, err := suite.submit(memberA, 1, false)
	suite.Require().NoError(err)
	res = suite.votingPower(1, memberC.String())
	suite.Require().Len(res.Proposals, 2)
	suite.Require().Equal(math.ZeroInt(), res.Proposals[0].EligibleWeight)
	suite.Require().Equal(types.ProposalVotingPower{ProposalId: second, EligibleWeight: math.NewInt(40), Shares: []types.BallotShare{
		{Source: string(srSource), Amount: math.NewInt(40)},
	}}, res.Proposals[1])

	// no stake: 0 everywhere
	res = suite.votingPower(1, attackerAddr.String())
	suite.Require().Equal(math.ZeroInt(), res.Current)
	suite.Require().Equal(math.ZeroInt(), res.Proposals[0].EligibleWeight)
	suite.Require().Equal(math.ZeroInt(), res.Proposals[1].EligibleWeight)
}

// An unknown DAO is NotFound; a bad address or a nil request is refused.
func (suite *IntegrationTestSuite) TestQueryVotingPower_Refused() {
	_, err := suite.k.VotingPower(suite.ctx, &types.QueryVotingPowerRequest{DaoId: 1, Address: memberA.String()})
	suite.Require().Equal(codes.NotFound, status.Code(err))

	suite.createStaticDao(noConfigChange)
	_, err = suite.k.VotingPower(suite.ctx, &types.QueryVotingPowerRequest{DaoId: 1, Address: "bad"})
	suite.Require().Equal(codes.InvalidArgument, status.Code(err))
	_, err = suite.k.VotingPower(suite.ctx, nil)
	suite.Require().Equal(codes.InvalidArgument, status.Code(err))
}

// A join on a pinned source is recorded at its height; no record (an unpinned
// source, or an exit) is NotFound.
func (suite *IntegrationTestSuite) TestQueryJoinRecord() {
	s := suite.createStakeDao()
	suite.atBlock(7)
	s.joinSr("1", memberC, 40)
	suite.Require().NoError(suite.k.StakingRewardHooks().AfterStakingRewardJoin(suite.ctx, "1", memberC.String(), math.NewInt(40), stakeDenom))
	suite.Require().NoError(suite.k.StakingRewardHooks().AfterStakingRewardJoin(suite.ctx, "5", memberC.String(), math.NewInt(40), stakeDenom))

	res, err := suite.k.JoinRecord(suite.ctx, &types.QueryJoinRecordRequest{Source: "sr/1", Address: memberC.String()})
	suite.Require().NoError(err)
	suite.Require().Equal(int64(7), res.Height)

	for _, req := range []*types.QueryJoinRecordRequest{
		{Source: "sr/5", Address: memberC.String()},
		{Source: "sr/1", Address: memberA.String()},
		{Source: "dr/" + stakeDenom, Address: memberC.String()},
	} {
		_, err = suite.k.JoinRecord(suite.ctx, req)
		suite.Require().Equal(codes.NotFound, status.Code(err), req.Source)
	}

	suite.Require().NoError(suite.k.StakingRewardHooks().AfterStakingRewardExit(suite.ctx, "1", memberC.String(), math.NewInt(40), stakeDenom))
	_, err = suite.k.JoinRecord(suite.ctx, &types.QueryJoinRecordRequest{Source: "sr/1", Address: memberC.String()})
	suite.Require().Equal(codes.NotFound, status.Code(err))

	for _, req := range []*types.QueryJoinRecordRequest{
		{Source: "", Address: memberC.String()},
		{Source: "sr/1", Address: "bad"},
		nil,
	} {
		_, err = suite.k.JoinRecord(suite.ctx, req)
		suite.Require().Equal(codes.InvalidArgument, status.Code(err))
	}
}
