package keeper_test

import (
	"time"

	"strconv"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/bze-alphateam/bze/x/dao/types"
)

func (suite *IntegrationTestSuite) queryProposal(id uint64) *types.QueryProposalResponse {
	res, err := suite.k.Proposal(suite.ctx, &types.QueryProposalRequest{Id: id})
	suite.Require().NoError(err)
	suite.Require().Equal(id, res.Proposal.Id)

	stored, found := suite.k.GetProposal(suite.ctx, id)
	suite.Require().True(found)
	suite.Require().Equal(stored, res.Proposal)

	return res
}

// requireLiveStatus checks the three computed flags of the proposal.
func (suite *IntegrationTestSuite) requireLiveStatus(id uint64, votingEnded bool, retryDeadline int64, expediteSettleable bool) {
	res := suite.queryProposal(id)
	suite.Require().Equal(votingEnded, res.VotingEnded, "voting_ended")
	suite.Require().Equal(retryDeadline, res.RetryDeadline, "retry_deadline")
	suite.Require().Equal(expediteSettleable, res.ExpediteSettleable, "expedite_settleable")
}

// voting_ended turns true at the voting end itself, and false again once the
// proposal is settled.
func (suite *IntegrationTestSuite) TestQueryProposal_VotingEnded() {
	suite.createStaticDao(noConfigChange)
	id, err := suite.submit(memberA, 1, false)
	suite.Require().NoError(err)
	suite.requireLiveStatus(id, false, 0, false)

	suite.endVoting(id, -1)
	suite.requireLiveStatus(id, false, 0, false)

	suite.endVoting(id, 0)
	suite.requireLiveStatus(id, true, 0, false)

	_, err = suite.settle(id)
	suite.Require().NoError(err)
	suite.requireLiveStatus(id, false, 0, false)
	suite.Require().Equal(types.ProposalStatus_REJECTED, suite.queryProposal(id).Proposal.Status)
}

// retry_deadline is voting_end + execution_window while the proposal is
// EXECUTION_FAILED (the last second a retry is accepted is one before it), 0
// before and once a retry executed it.
func (suite *IntegrationTestSuite) TestQueryProposal_RetryDeadline() {
	suite.createStaticDao(noConfigChange)
	id := suite.passedProposal(treasurySend(memberB, 30))
	suite.requireLiveStatus(id, false, 0, false)

	suite.endVoting(id, 0)
	status, err := suite.settle(id)
	suite.Require().NoError(err)
	suite.Require().Equal(types.ProposalStatus_EXECUTION_FAILED, status)

	proposal, _ := suite.k.GetProposal(suite.ctx, id)
	deadline := proposal.VotingEnd + int64(types.DefaultExecutionWindow.Seconds())
	suite.requireLiveStatus(id, false, deadline, false)

	// the deadline follows the current execution_window param
	params := suite.k.GetParams(suite.ctx)
	params.ExecutionWindow /= 2
	suite.Require().NoError(suite.k.SetParams(suite.ctx, params))
	deadline = proposal.VotingEnd + int64(params.ExecutionWindow.Seconds())
	suite.requireLiveStatus(id, false, deadline, false)

	suite.endVoting(id, deadline-proposal.VotingEnd-1)
	suite.ledger.fund(daoAddress(1), ubze(30))
	status, err = suite.settle(id)
	suite.Require().NoError(err)
	suite.Require().Equal(types.ProposalStatus_EXECUTED, status)
	suite.requireLiveStatus(id, false, 0, false)
}

// At the deadline a settle expires the proposal; it then has no deadline.
func (suite *IntegrationTestSuite) TestQueryProposal_RetryDeadlineExpired() {
	suite.createStaticDao(noConfigChange)
	id := suite.passedProposal(treasurySend(memberB, 30))
	suite.endVoting(id, 0)
	_, err := suite.settle(id)
	suite.Require().NoError(err)
	deadline := suite.queryProposal(id).RetryDeadline

	suite.endVoting(id, deadline-suite.queryProposal(id).Proposal.VotingEnd)
	status, err := suite.settle(id)
	suite.Require().NoError(err)
	suite.Require().Equal(types.ProposalStatus_EXPIRED, status)
	suite.requireLiveStatus(id, false, 0, false)
}

// expedite_settleable needs an expedited proposal, before its voting end, whose
// tally meets the expedite bars: one yes of two members misses them, two meet
// them, and from the voting end the flag gives way to voting_ended.
func (suite *IntegrationTestSuite) TestQueryProposal_ExpediteSettleable() {
	suite.setExpediteDelay(0)
	suite.createStaticDao(noConfigChange)
	expedited, err := suite.submit(memberA, 1, true)
	suite.Require().NoError(err)
	regular := suite.passedProposal()

	suite.Require().NoError(suite.vote(memberA, expedited, types.VoteOption_YES))
	suite.requireLiveStatus(expedited, false, 0, false)
	suite.Require().NoError(suite.vote(memberB, expedited, types.VoteOption_YES))
	suite.requireLiveStatus(expedited, false, 0, true)

	// a regular proposal never settles early, whatever its tally
	suite.requireLiveStatus(regular, false, 0, false)

	suite.endVoting(expedited, -1)
	suite.requireLiveStatus(expedited, false, 0, true)
	suite.endVoting(expedited, 0)
	suite.requireLiveStatus(expedited, true, 0, false)

	// settled early, it is no longer VOTING
	suite.endVoting(expedited, -1)
	_, err = suite.settle(expedited)
	suite.Require().NoError(err)
	suite.requireLiveStatus(expedited, false, 0, false)
}

// The flag follows min_expedite_delay as SettleProposal does: a proposal that
// meets the bars is not settleable until the delay after its submission.
func (suite *IntegrationTestSuite) TestQueryProposal_ExpediteSettleableAfterDelay() {
	suite.setExpediteDelay(time.Hour)
	suite.createStaticDao(noConfigChange)
	id, err := suite.submit(memberA, 1, true)
	suite.Require().NoError(err)
	suite.Require().NoError(suite.vote(memberA, id, types.VoteOption_YES))
	suite.Require().NoError(suite.vote(memberB, id, types.VoteOption_YES))
	suite.requireLiveStatus(id, false, 0, false)

	proposal, _ := suite.k.GetProposal(suite.ctx, id)
	suite.ctx = suite.ctx.WithBlockTime(time.Unix(proposal.SubmitTime+3599, 0))
	suite.requireLiveStatus(id, false, 0, false)
	suite.ctx = suite.ctx.WithBlockTime(time.Unix(proposal.SubmitTime+3600, 0))
	suite.requireLiveStatus(id, false, 0, true)
}

// The flag uses the exact expedite bars SettleProposal checks: yes exactly
// expedite_threshold × D (0.66 × 150 = 99) is settleable, one unit less is not.
func (suite *IntegrationTestSuite) TestQueryProposal_ExpediteSettleableBoundary() {
	for _, tc := range []struct {
		yes        int64
		settleable bool
	}{{99, true}, {98, false}} {
		suite.Run(strconv.FormatInt(tc.yes, 10), func() {
			suite.SetupTest()
			suite.setExpediteDelay(0)
			_, s := suite.expediteStakeDao(noConfigChange)
			s.joinSr("1", memberA, tc.yes)
			suite.atBlock(10)
			id, err := suite.submit(memberA, 1, true)
			suite.Require().NoError(err)
			suite.Require().NoError(suite.vote(memberA, id, types.VoteOption_YES))

			suite.requireLiveStatus(id, false, 0, tc.settleable)
			_, err = suite.settle(id)
			suite.Require().Equal(tc.settleable, err == nil)
		})
	}
}

// An unknown id, a cleaned-up proposal and a nil request are refused.
func (suite *IntegrationTestSuite) TestQueryProposal_NotFound() {
	_, err := suite.k.Proposal(suite.ctx, &types.QueryProposalRequest{Id: 1})
	suite.Require().Equal(codes.NotFound, status.Code(err))

	suite.createStaticDao(noConfigChange)
	id, err := suite.submit(memberA, 1, false)
	suite.Require().NoError(err)
	suite.Require().NoError(suite.cancel(memberA, id))
	res, err := suite.cleanup(id, 10)
	suite.Require().NoError(err)
	suite.Require().True(res.Completed)

	_, err = suite.k.Proposal(suite.ctx, &types.QueryProposalRequest{Id: id})
	suite.Require().Equal(codes.NotFound, status.Code(err))
	suite.Require().ErrorContains(err, "proposal 1 not found")

	_, err = suite.k.Proposal(suite.ctx, &types.QueryProposalRequest{Id: 0})
	suite.Require().Equal(codes.NotFound, status.Code(err))

	_, err = suite.k.Proposal(suite.ctx, nil)
	suite.Require().Equal(codes.InvalidArgument, status.Code(err))
}
