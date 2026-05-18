package keeper_test

import (
	"time"

	"github.com/bze-alphateam/bze/x/daodao/types"
)

// TestRevote_Enabled_ReplacesPreviousOption: with allow_revote = true, a
// second MsgVote replaces the first; the tally subtracts the old option's
// contribution and adds the new option's.
func (suite *IntegrationTestSuite) TestRevote_Enabled_ReplacesPreviousOption() {
	a := freshAddr()
	b := freshAddr()
	gov := validGovernance() // AllowRevote = true
	daoID, _ := suite.createDaoWithMembers("revote-on",
		[]types.StaticMember{
			{Address: a, Weight: 3},
			{Address: b, Weight: 1},
		}, gov)

	pid := suite.createTestProposal(daoID, a)

	// First vote: NO with weight 3.
	_, err := suite.msgServer.Vote(suite.ctx, &types.MsgVote{
		Voter: a, DaoId: daoID, ProposalId: pid, Option: types.VoteOption_VOTE_OPTION_NO,
	})
	suite.Require().NoError(err)

	p, _ := suite.k.GetProposal(suite.ctx, daoID, pid)
	suite.Require().Equal(uint64(3), p.Tally.NoPower)
	suite.Require().Equal(uint64(0), p.Tally.YesPower)

	// Revote: switch to YES. NoPower must go back to 0; YesPower to 3.
	_, err = suite.msgServer.Vote(suite.ctx, &types.MsgVote{
		Voter: a, DaoId: daoID, ProposalId: pid, Option: types.VoteOption_VOTE_OPTION_YES,
	})
	suite.Require().NoError(err)

	p, _ = suite.k.GetProposal(suite.ctx, daoID, pid)
	suite.Require().Equal(uint64(0), p.Tally.NoPower)
	suite.Require().Equal(uint64(3), p.Tally.YesPower)
}

// TestRevote_Disabled_SecondVoteRejected: with allow_revote = false, a
// second MsgVote from the same voter is rejected with ErrRevoteNotAllowed.
func (suite *IntegrationTestSuite) TestRevote_Disabled_SecondVoteRejected() {
	a := freshAddr()
	b := freshAddr()
	c := freshAddr()
	// Disable revote; use WITHOUT_QUORUM with a 99% threshold so the first
	// vote can't accidentally early-close as pass.
	gov := validGovernance()
	gov.AllowRevote = false
	gov.ThresholdBps = 9_900
	daoID, _ := suite.createDaoWithMembers("revote-off",
		[]types.StaticMember{
			{Address: a, Weight: 1},
			{Address: b, Weight: 1},
			{Address: c, Weight: 1},
		}, gov)

	pid := suite.createTestProposal(daoID, a)

	_, err := suite.msgServer.Vote(suite.ctx, &types.MsgVote{
		Voter: a, DaoId: daoID, ProposalId: pid, Option: types.VoteOption_VOTE_OPTION_YES,
	})
	suite.Require().NoError(err)

	_, err = suite.msgServer.Vote(suite.ctx, &types.MsgVote{
		Voter: a, DaoId: daoID, ProposalId: pid, Option: types.VoteOption_VOTE_OPTION_NO,
	})
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "revoting is disabled")

	// Tally should still reflect ONLY the first vote.
	p, _ := suite.k.GetProposal(suite.ctx, daoID, pid)
	suite.Require().Equal(uint64(1), p.Tally.YesPower)
	suite.Require().Equal(uint64(0), p.Tally.NoPower)
	_ = time.Hour
}
