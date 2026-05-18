package keeper_test

import (
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/bze-alphateam/bze/x/daodao/types"
)

// TestPollQuorum_NotMetRejects: quorum > 0; voted/total < quorum → REJECTED.
func (suite *IntegrationTestSuite) TestPollQuorum_NotMetRejects() {
	a := freshAddr()
	b := freshAddr()
	c := freshAddr()
	daoID, _ := suite.makePollDaoMultiMember("poll-quorum-fail", []types.StaticMember{
		{Address: a, Weight: 1},
		{Address: b, Weight: 1},
		{Address: c, Weight: 8}, // huge, doesn't vote
	})
	opts := defaultPollOpts()
	opts.quorumBps = 6_000 // need 60% of total power = 6 of 10
	pid := suite.stagePollInVoting(daoID, a, opts)

	// a + b vote → voted=2/10=20% < 60%. REJECTED.
	_, err := suite.msgServer.VoteOnPoll(suite.ctx, &types.MsgVoteOnPoll{
		Voter: a, DaoId: daoID, PollId: pid, ChoiceIndices: []uint32{0},
	})
	suite.Require().NoError(err)
	_, err = suite.msgServer.VoteOnPoll(suite.ctx, &types.MsgVoteOnPoll{
		Voter: b, DaoId: daoID, PollId: pid, ChoiceIndices: []uint32{0},
	})
	suite.Require().NoError(err)

	suite.expectTreasuryForfeit(daoID, sdk.NewInt64Coin("ubze", 1))
	suite.advanceTime(48 * time.Hour)
	suite.Require().NoError(suite.k.EndBlock(suite.ctx))

	p, _ := suite.k.GetPoll(suite.ctx, daoID, pid)
	suite.Require().Equal(types.PollStatus_POLL_STATUS_REJECTED, p.Status,
		"quorum not met must reject")
}

// TestPollQuorum_MetConcludes: quorum met → CONCLUDED.
func (suite *IntegrationTestSuite) TestPollQuorum_MetConcludes() {
	a := freshAddr()
	b := freshAddr()
	daoID, _ := suite.makePollDaoMultiMember("poll-quorum-ok", []types.StaticMember{
		{Address: a, Weight: 7},
		{Address: b, Weight: 3},
	})
	opts := defaultPollOpts()
	opts.quorumBps = 5_000 // 50% of 10 = 5
	pid := suite.stagePollInVoting(daoID, a, opts)

	// a alone votes (7/10=70% > 50%). Choice 0 wins.
	_, err := suite.msgServer.VoteOnPoll(suite.ctx, &types.MsgVoteOnPoll{
		Voter: a, DaoId: daoID, PollId: pid, ChoiceIndices: []uint32{0},
	})
	suite.Require().NoError(err)

	suite.expectRefundOnTerminal(daoID, a, sdk.NewInt64Coin("ubze", 1))
	suite.advanceTime(48 * time.Hour)
	suite.Require().NoError(suite.k.EndBlock(suite.ctx))

	p, _ := suite.k.GetPoll(suite.ctx, daoID, pid)
	suite.Require().Equal(types.PollStatus_POLL_STATUS_CONCLUDED, p.Status)
	suite.Require().Equal(uint32(0), p.WinningChoiceIndex)
}

// TestPollQuorum_ZeroSkipsCheck: quorum_bps=0 means no quorum gate; even
// a tiny vote suffices to conclude.
func (suite *IntegrationTestSuite) TestPollQuorum_ZeroSkipsCheck() {
	daoID, members := suite.makePollDaoMultiMember("poll-no-quorum", []types.StaticMember{
		{Address: freshAddr(), Weight: 1},
		{Address: freshAddr(), Weight: 99}, // doesn't vote
	})
	// Quorum 0 (default).
	pid := suite.stagePollInVoting(daoID, members[0], defaultPollOpts())

	_, err := suite.msgServer.VoteOnPoll(suite.ctx, &types.MsgVoteOnPoll{
		Voter: members[0], DaoId: daoID, PollId: pid, ChoiceIndices: []uint32{1},
	})
	suite.Require().NoError(err)

	suite.expectRefundOnTerminal(daoID, members[0], sdk.NewInt64Coin("ubze", 1))
	suite.advanceTime(48 * time.Hour)
	suite.Require().NoError(suite.k.EndBlock(suite.ctx))

	p, _ := suite.k.GetPoll(suite.ctx, daoID, pid)
	suite.Require().Equal(types.PollStatus_POLL_STATUS_CONCLUDED, p.Status)
	suite.Require().Equal(uint32(1), p.WinningChoiceIndex)
}
