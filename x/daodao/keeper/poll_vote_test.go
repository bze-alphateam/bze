package keeper_test

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/bze-alphateam/bze/x/daodao/types"
)

// makePollDaoMultiMember creates a DAO with the supplied member weights
// and a permissive governance config that lets every member vote on a
// poll. Returns (daoID, members in input order).
func (suite *IntegrationTestSuite) makePollDaoMultiMember(name string, members []types.StaticMember) (uint64, []string) {
	gov := validGovernance()
	gov.AllowRevote = true // poll revote
	creator := members[0].Address
	suite.expectAccountCreated(suite.k.GetDaoIDCounter(suite.ctx))
	resp, err := suite.msgServer.CreateDao(suite.ctx, &types.MsgCreateDao{
		Creator:      creator,
		Metadata:     sampleMetadata(name),
		VotingConfig: staticConfigWithMembers(members),
		Governance:   gov,
		Deposit:      validDeposit(),
	})
	suite.Require().NoError(err)

	addrs := make([]string, len(members))
	for i, m := range members {
		addrs[i] = m.Address
	}
	return resp.DaoId, addrs
}

// stagePollInVoting helper: creates a poll with the given choices /
// max_selections / nota, puts it directly into VOTING via a full
// initial deposit (1ubze).
func (suite *IntegrationTestSuite) stagePollInVoting(daoID uint64, proposer string, opts pollOpts) uint64 {
	opts.deposit = sdk.NewInt64Coin("ubze", 1)
	return suite.createPollMember(daoID, proposer, opts)
}

// TestPollVote_SingleSelection_MaxSelectionsOne: standard plurality
// behaviour. One voter; one chosen index gets the full power.
func (suite *IntegrationTestSuite) TestPollVote_SingleSelection_MaxSelectionsOne() {
	daoID, members := suite.makePollDaoMultiMember("poll-single", []types.StaticMember{
		{Address: freshAddr(), Weight: 5},
	})
	pid := suite.stagePollInVoting(daoID, members[0], defaultPollOpts())

	_, err := suite.msgServer.VoteOnPoll(suite.ctx, &types.MsgVoteOnPoll{
		Voter:         members[0],
		DaoId:         daoID,
		PollId:        pid,
		ChoiceIndices: []uint32{1},
	})
	suite.Require().NoError(err)

	p, _ := suite.k.GetPoll(suite.ctx, daoID, pid)
	suite.Require().Equal(uint64(0), p.Tally.ChoicePower[0])
	suite.Require().Equal(uint64(5), p.Tally.ChoicePower[1])
	suite.Require().Equal(uint64(0), p.Tally.ChoicePower[2])
	suite.Require().Equal(uint64(5), p.Tally.TotalVotedPower)
}

// TestPollVote_ApprovalMultiSelect: voter picks 2 of 3 choices; each
// chosen choice gets full power; total_voted_power increments by power
// (not 2*power) — single distinct voter.
func (suite *IntegrationTestSuite) TestPollVote_ApprovalMultiSelect() {
	daoID, members := suite.makePollDaoMultiMember("poll-multi", []types.StaticMember{
		{Address: freshAddr(), Weight: 7},
	})
	opts := defaultPollOpts()
	opts.maxSelections = 3
	pid := suite.stagePollInVoting(daoID, members[0], opts)

	_, err := suite.msgServer.VoteOnPoll(suite.ctx, &types.MsgVoteOnPoll{
		Voter:         members[0],
		DaoId:         daoID,
		PollId:        pid,
		ChoiceIndices: []uint32{0, 2},
	})
	suite.Require().NoError(err)

	p, _ := suite.k.GetPoll(suite.ctx, daoID, pid)
	suite.Require().Equal(uint64(7), p.Tally.ChoicePower[0])
	suite.Require().Equal(uint64(0), p.Tally.ChoicePower[1])
	suite.Require().Equal(uint64(7), p.Tally.ChoicePower[2])
	suite.Require().Equal(uint64(7), p.Tally.TotalVotedPower,
		"approval-style: one voter contributes once to total_voted_power")
}

// TestPollVote_AllOptions_MaxSelectionsN: max_selections = N means a
// voter can pick all N user choices in one go.
func (suite *IntegrationTestSuite) TestPollVote_AllOptions_MaxSelectionsN() {
	daoID, members := suite.makePollDaoMultiMember("poll-all", []types.StaticMember{
		{Address: freshAddr(), Weight: 3},
	})
	opts := defaultPollOpts()
	opts.maxSelections = 3
	pid := suite.stagePollInVoting(daoID, members[0], opts)

	_, err := suite.msgServer.VoteOnPoll(suite.ctx, &types.MsgVoteOnPoll{
		Voter:         members[0],
		DaoId:         daoID,
		PollId:        pid,
		ChoiceIndices: []uint32{0, 1, 2},
	})
	suite.Require().NoError(err)

	p, _ := suite.k.GetPoll(suite.ctx, daoID, pid)
	for i := range p.Tally.ChoicePower {
		suite.Require().Equal(uint64(3), p.Tally.ChoicePower[i],
			"each picked choice receives full power; index %d", i)
	}
	suite.Require().Equal(uint64(3), p.Tally.TotalVotedPower)
}

// TestPollVote_DuplicateIndex: duplicate indices in choice_indices are
// rejected.
func (suite *IntegrationTestSuite) TestPollVote_DuplicateIndex() {
	daoID, members := suite.makePollDaoMultiMember("poll-dup-idx", []types.StaticMember{
		{Address: freshAddr(), Weight: 1},
	})
	opts := defaultPollOpts()
	opts.maxSelections = 3
	pid := suite.stagePollInVoting(daoID, members[0], opts)

	_, err := suite.msgServer.VoteOnPoll(suite.ctx, &types.MsgVoteOnPoll{
		Voter:         members[0],
		DaoId:         daoID,
		PollId:        pid,
		ChoiceIndices: []uint32{0, 0},
	})
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "duplicate")
}

// TestPollVote_IndexOutOfRange: too-large index rejected.
func (suite *IntegrationTestSuite) TestPollVote_IndexOutOfRange() {
	daoID, members := suite.makePollDaoMultiMember("poll-oob", []types.StaticMember{
		{Address: freshAddr(), Weight: 1},
	})
	pid := suite.stagePollInVoting(daoID, members[0], defaultPollOpts())

	_, err := suite.msgServer.VoteOnPoll(suite.ctx, &types.MsgVoteOnPoll{
		Voter:         members[0],
		DaoId:         daoID,
		PollId:        pid,
		ChoiceIndices: []uint32{99},
	})
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "out of range")
}

// TestPollVote_OverMaxSelections: selecting more than max_selections
// (without NOTA) is rejected.
func (suite *IntegrationTestSuite) TestPollVote_OverMaxSelections() {
	daoID, members := suite.makePollDaoMultiMember("poll-over-max", []types.StaticMember{
		{Address: freshAddr(), Weight: 1},
	})
	opts := defaultPollOpts() // max_selections defaults to 1
	pid := suite.stagePollInVoting(daoID, members[0], opts)

	_, err := suite.msgServer.VoteOnPoll(suite.ctx, &types.MsgVoteOnPoll{
		Voter:         members[0],
		DaoId:         daoID,
		PollId:        pid,
		ChoiceIndices: []uint32{0, 1},
	})
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "max_selections")
}

// TestPollVote_NonVoterRejected: a voter not in the snapshot is rejected.
func (suite *IntegrationTestSuite) TestPollVote_NonVoterRejected() {
	daoID, members := suite.makePollDaoMultiMember("poll-nonvoter", []types.StaticMember{
		{Address: freshAddr(), Weight: 1},
	})
	pid := suite.stagePollInVoting(daoID, members[0], defaultPollOpts())

	_, err := suite.msgServer.VoteOnPoll(suite.ctx, &types.MsgVoteOnPoll{
		Voter:         freshAddr(),
		DaoId:         daoID,
		PollId:        pid,
		ChoiceIndices: []uint32{0},
	})
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "no snapshot power")
}
