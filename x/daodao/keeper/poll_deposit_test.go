package keeper_test

import (
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/bze-alphateam/bze/x/daodao/types"
)

// TestPollDeposit_TopUpTransitionsToVoting: a DEPOSIT_PERIOD poll
// reaches min_deposit via MsgDepositOnPoll and transitions to VOTING.
func (suite *IntegrationTestSuite) TestPollDeposit_TopUpTransitionsToVoting() {
	daoID, member := suite.createSampleDao("poll-topup")

	// Start in DEPOSIT_PERIOD (initial 0 deposit).
	pid := suite.createPollMember(daoID, member, defaultPollOpts())

	depositor := freshAddr()
	suite.expectInitialDepositSend(daoID, depositor, sdk.NewInt64Coin("ubze", 1))
	_, err := suite.msgServer.DepositOnPoll(suite.ctx, &types.MsgDepositOnPoll{
		Depositor: depositor,
		DaoId:     daoID,
		PollId:    pid,
		Amount:    sdk.NewInt64Coin("ubze", 1),
	})
	suite.Require().NoError(err)

	p, _ := suite.k.GetPoll(suite.ctx, daoID, pid)
	suite.Require().Equal(types.PollStatus_POLL_STATUS_VOTING, p.Status)
	suite.Require().Equal("1ubze", p.DepositCollected.String())
}

// TestPollDeposit_OnVotingRejected: depositing on a VOTING-status poll
// is rejected with the poll-specific error code.
func (suite *IntegrationTestSuite) TestPollDeposit_OnVotingRejected() {
	daoID, member := suite.createSampleDao("poll-notdeposit")
	opts := defaultPollOpts()
	opts.deposit = sdk.NewInt64Coin("ubze", 1) // direct to VOTING
	pid := suite.createPollMember(daoID, member, opts)

	_, err := suite.msgServer.DepositOnPoll(suite.ctx, &types.MsgDepositOnPoll{
		Depositor: freshAddr(),
		DaoId:     daoID,
		PollId:    pid,
		Amount:    sdk.NewInt64Coin("ubze", 1),
	})
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "deposit-period")
}

// TestPollDeposit_ExpiryForfeitToTreasury: a DEPOSIT_PERIOD poll that
// passes its deposit_deadline without funding gets forfeited to the
// DAO treasury.
func (suite *IntegrationTestSuite) TestPollDeposit_ExpiryForfeitToTreasury() {
	daoID, member := suite.createSampleDao("poll-expire-treasury")
	pid := suite.createPollMember(daoID, member, defaultPollOpts())

	// No depositors; default deposit_period = 7d. Advance > 7d.
	suite.advanceTime(8 * 24 * time.Hour)
	suite.Require().NoError(suite.k.EndBlock(suite.ctx))

	p, _ := suite.k.GetPoll(suite.ctx, daoID, pid)
	suite.Require().Equal(types.PollStatus_POLL_STATUS_REJECTED_NO_DEPOSIT, p.Status)
}

// TestPollDeposit_RefundOnConclude: a CONCLUDED poll (winning_choice
// selected, ON_PASS policy) refunds the proposer's initial deposit.
func (suite *IntegrationTestSuite) TestPollDeposit_RefundOnConclude() {
	daoID, members := suite.makePollDaoMultiMember("poll-refund", []types.StaticMember{
		{Address: freshAddr(), Weight: 5},
	})
	opts := defaultPollOpts()
	opts.deposit = sdk.NewInt64Coin("ubze", 1)
	pid := suite.stagePollInVoting(daoID, members[0], opts)

	_, err := suite.msgServer.VoteOnPoll(suite.ctx, &types.MsgVoteOnPoll{
		Voter: members[0], DaoId: daoID, PollId: pid, ChoiceIndices: []uint32{1},
	})
	suite.Require().NoError(err)

	suite.expectRefundOnTerminal(daoID, members[0], sdk.NewInt64Coin("ubze", 1))
	suite.advanceTime(48 * time.Hour)
	suite.Require().NoError(suite.k.EndBlock(suite.ctx))

	p, _ := suite.k.GetPoll(suite.ctx, daoID, pid)
	suite.Require().Equal(types.PollStatus_POLL_STATUS_CONCLUDED, p.Status)
}
