package keeper_test

import (
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/bze-alphateam/bze/x/daodao/types"
)

// TestPollNota_AloneAccepted: [NOTA] alone is a valid selection set
// when NOTA is enabled.
func (suite *IntegrationTestSuite) TestPollNota_AloneAccepted() {
	daoID, members := suite.makePollDaoMultiMember("poll-nota-alone", []types.StaticMember{
		{Address: freshAddr(), Weight: 1},
	})
	opts := defaultPollOpts()
	opts.includeNota = true
	opts.maxSelections = 2
	pid := suite.stagePollInVoting(daoID, members[0], opts)

	p, _ := suite.k.GetPoll(suite.ctx, daoID, pid)
	notaIdx := uint32(len(p.Choices) - 1) // 3 user + NOTA = 4; nota idx 3

	_, err := suite.msgServer.VoteOnPoll(suite.ctx, &types.MsgVoteOnPoll{
		Voter:         members[0],
		DaoId:         daoID,
		PollId:        pid,
		ChoiceIndices: []uint32{notaIdx},
	})
	suite.Require().NoError(err)

	p, _ = suite.k.GetPoll(suite.ctx, daoID, pid)
	suite.Require().Equal(uint64(1), p.Tally.ChoicePower[notaIdx])
	suite.Require().Equal(uint64(1), p.Tally.TotalVotedPower)
}

// TestPollNota_MixedRejected: combining NOTA with any other choice is
// rejected (NOTA exclusivity).
func (suite *IntegrationTestSuite) TestPollNota_MixedRejected() {
	daoID, members := suite.makePollDaoMultiMember("poll-nota-mixed", []types.StaticMember{
		{Address: freshAddr(), Weight: 1},
	})
	opts := defaultPollOpts()
	opts.includeNota = true
	opts.maxSelections = 2
	pid := suite.stagePollInVoting(daoID, members[0], opts)

	p, _ := suite.k.GetPoll(suite.ctx, daoID, pid)
	notaIdx := uint32(len(p.Choices) - 1)

	_, err := suite.msgServer.VoteOnPoll(suite.ctx, &types.MsgVoteOnPoll{
		Voter:         members[0],
		DaoId:         daoID,
		PollId:        pid,
		ChoiceIndices: []uint32{0, notaIdx},
	})
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "NOTA must be the sole selection")
}

// TestPollNota_WinsRejectsPoll: a poll where NOTA is the plurality
// winner finalizes as REJECTED (not CONCLUDED).
func (suite *IntegrationTestSuite) TestPollNota_WinsRejectsPoll() {
	a := freshAddr()
	b := freshAddr()
	daoID, _ := suite.makePollDaoMultiMember("poll-nota-wins", []types.StaticMember{
		{Address: a, Weight: 3}, // votes NOTA
		{Address: b, Weight: 1}, // votes choice 0
	})
	opts := defaultPollOpts()
	opts.includeNota = true
	opts.maxSelections = 1
	pid := suite.stagePollInVoting(daoID, a, opts)

	p, _ := suite.k.GetPoll(suite.ctx, daoID, pid)
	notaIdx := uint32(len(p.Choices) - 1)

	_, err := suite.msgServer.VoteOnPoll(suite.ctx, &types.MsgVoteOnPoll{
		Voter: a, DaoId: daoID, PollId: pid, ChoiceIndices: []uint32{notaIdx},
	})
	suite.Require().NoError(err)
	_, err = suite.msgServer.VoteOnPoll(suite.ctx, &types.MsgVoteOnPoll{
		Voter: b, DaoId: daoID, PollId: pid, ChoiceIndices: []uint32{0},
	})
	suite.Require().NoError(err)

	// On-pass refund (validDeposit default) means PASSED would refund;
	// REJECTED forfeits to treasury. NOTA-wins → REJECTED → forfeit.
	suite.expectTreasuryForfeit(daoID, sdk.NewInt64Coin("ubze", 1))
	suite.advanceTime(48 * time.Hour)
	suite.Require().NoError(suite.k.EndBlock(suite.ctx))

	p, _ = suite.k.GetPoll(suite.ctx, daoID, pid)
	suite.Require().Equal(types.PollStatus_POLL_STATUS_REJECTED, p.Status,
		"NOTA winning must reject the poll")
}

// TestPollNota_TiesRejectPoll: a tie at the top (including a NOTA tie)
// rejects.
func (suite *IntegrationTestSuite) TestPollNota_TiesRejectPoll() {
	a := freshAddr()
	b := freshAddr()
	daoID, _ := suite.makePollDaoMultiMember("poll-tie", []types.StaticMember{
		{Address: a, Weight: 1},
		{Address: b, Weight: 1},
	})
	pid := suite.stagePollInVoting(daoID, a, defaultPollOpts())

	_, err := suite.msgServer.VoteOnPoll(suite.ctx, &types.MsgVoteOnPoll{
		Voter: a, DaoId: daoID, PollId: pid, ChoiceIndices: []uint32{0},
	})
	suite.Require().NoError(err)
	_, err = suite.msgServer.VoteOnPoll(suite.ctx, &types.MsgVoteOnPoll{
		Voter: b, DaoId: daoID, PollId: pid, ChoiceIndices: []uint32{1},
	})
	suite.Require().NoError(err)

	suite.expectTreasuryForfeit(daoID, sdk.NewInt64Coin("ubze", 1))
	suite.advanceTime(48 * time.Hour)
	suite.Require().NoError(suite.k.EndBlock(suite.ctx))

	p, _ := suite.k.GetPoll(suite.ctx, daoID, pid)
	suite.Require().Equal(types.PollStatus_POLL_STATUS_REJECTED, p.Status,
		"top-tie must reject the poll")
}

// TestPollNota_UniqueWinnerConcludes: a clean plurality winner with no
// NOTA-collision concludes the poll.
func (suite *IntegrationTestSuite) TestPollNota_UniqueWinnerConcludes() {
	a := freshAddr()
	b := freshAddr()
	daoID, _ := suite.makePollDaoMultiMember("poll-winner", []types.StaticMember{
		{Address: a, Weight: 3},
		{Address: b, Weight: 1},
	})
	pid := suite.stagePollInVoting(daoID, a, defaultPollOpts())

	_, err := suite.msgServer.VoteOnPoll(suite.ctx, &types.MsgVoteOnPoll{
		Voter: a, DaoId: daoID, PollId: pid, ChoiceIndices: []uint32{2},
	})
	suite.Require().NoError(err)
	_, err = suite.msgServer.VoteOnPoll(suite.ctx, &types.MsgVoteOnPoll{
		Voter: b, DaoId: daoID, PollId: pid, ChoiceIndices: []uint32{0},
	})
	suite.Require().NoError(err)

	// CONCLUDED + ON_PASS policy → refund to proposer.
	suite.expectRefundOnTerminal(daoID, a, sdk.NewInt64Coin("ubze", 1))
	suite.advanceTime(48 * time.Hour)
	suite.Require().NoError(suite.k.EndBlock(suite.ctx))

	p, _ := suite.k.GetPoll(suite.ctx, daoID, pid)
	suite.Require().Equal(types.PollStatus_POLL_STATUS_CONCLUDED, p.Status)
	suite.Require().Equal(uint32(2), p.WinningChoiceIndex)
}
