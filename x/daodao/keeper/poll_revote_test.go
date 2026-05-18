package keeper_test

import (
	"github.com/bze-alphateam/bze/x/daodao/types"
)

// TestPollRevote_EnabledReplacesPreviousSelection: with revote enabled,
// a second VoteOnPoll subtracts the old selections and adds the new
// ones; total_voted_power stays constant (same distinct voter).
func (suite *IntegrationTestSuite) TestPollRevote_EnabledReplacesPreviousSelection() {
	daoID, members := suite.makePollDaoMultiMember("poll-revote-on", []types.StaticMember{
		{Address: freshAddr(), Weight: 3},
	})
	opts := defaultPollOpts()
	opts.maxSelections = 2
	pid := suite.stagePollInVoting(daoID, members[0], opts)

	// First vote: [0, 1].
	_, err := suite.msgServer.VoteOnPoll(suite.ctx, &types.MsgVoteOnPoll{
		Voter: members[0], DaoId: daoID, PollId: pid, ChoiceIndices: []uint32{0, 1},
	})
	suite.Require().NoError(err)

	p, _ := suite.k.GetPoll(suite.ctx, daoID, pid)
	suite.Require().Equal(uint64(3), p.Tally.ChoicePower[0])
	suite.Require().Equal(uint64(3), p.Tally.ChoicePower[1])
	suite.Require().Equal(uint64(0), p.Tally.ChoicePower[2])
	suite.Require().Equal(uint64(3), p.Tally.TotalVotedPower)

	// Revote: switch to [2] alone. Choice 0 and 1 must drop to 0; 2 to 3.
	_, err = suite.msgServer.VoteOnPoll(suite.ctx, &types.MsgVoteOnPoll{
		Voter: members[0], DaoId: daoID, PollId: pid, ChoiceIndices: []uint32{2},
	})
	suite.Require().NoError(err)

	p, _ = suite.k.GetPoll(suite.ctx, daoID, pid)
	suite.Require().Equal(uint64(0), p.Tally.ChoicePower[0])
	suite.Require().Equal(uint64(0), p.Tally.ChoicePower[1])
	suite.Require().Equal(uint64(3), p.Tally.ChoicePower[2])
	suite.Require().Equal(uint64(3), p.Tally.TotalVotedPower,
		"revote preserves total_voted_power (same distinct voter)")
}

// TestPollRevote_DisabledSecondVoteRejected: with revote disabled, a
// second VoteOnPoll from the same voter is rejected. Tally reflects
// only the first vote.
func (suite *IntegrationTestSuite) TestPollRevote_DisabledSecondVoteRejected() {
	// Build the DAO with allow_revote = false so the poll inherits that.
	gov := validGovernance()
	gov.AllowRevote = false
	creator := freshAddr()
	suite.expectAccountCreated(suite.k.GetDaoIDCounter(suite.ctx))
	resp, err := suite.msgServer.CreateDao(suite.ctx, &types.MsgCreateDao{
		Creator:      creator,
		Metadata:     sampleMetadata("poll-revote-off"),
		VotingConfig: staticConfig(creator),
		Governance:   gov,
		Deposit:      validDeposit(),
	})
	suite.Require().NoError(err)
	daoID := resp.DaoId

	pid := suite.stagePollInVoting(daoID, creator, defaultPollOpts())

	_, err = suite.msgServer.VoteOnPoll(suite.ctx, &types.MsgVoteOnPoll{
		Voter: creator, DaoId: daoID, PollId: pid, ChoiceIndices: []uint32{0},
	})
	suite.Require().NoError(err)

	// Second vote rejected.
	_, err = suite.msgServer.VoteOnPoll(suite.ctx, &types.MsgVoteOnPoll{
		Voter: creator, DaoId: daoID, PollId: pid, ChoiceIndices: []uint32{1},
	})
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "revoting is disabled")

	// Tally reflects only the first vote.
	p, _ := suite.k.GetPoll(suite.ctx, daoID, pid)
	suite.Require().Equal(uint64(1), p.Tally.ChoicePower[0])
	suite.Require().Equal(uint64(0), p.Tally.ChoicePower[1])
}
