package keeper_test

import (
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/bze-alphateam/bze/x/daodao/types"
)

// TestEndBlock_FinalizesPastExpiry: at block time > voting_end, the
// end-blocker transitions a VOTING proposal to PASSED/REJECTED and removes
// it from the queue.
func (suite *IntegrationTestSuite) TestEndBlock_FinalizesPastExpiry() {
	a := freshAddr()
	b := freshAddr()
	gov := validGovernance()
	gov.AllowRevote = true // disable early-close so finalization is the only path to close
	gov.ThresholdBps = 5_000
	daoID, _ := suite.createDaoWithMembers("end-block",
		[]types.StaticMember{
			{Address: a, Weight: 1},
			{Address: b, Weight: 1},
		}, gov)

	pid := suite.createTestProposal(daoID, a)

	// 1/2 = 50% — strictly NOT > 50%. Reject expected.
	_, err := suite.msgServer.Vote(suite.ctx, &types.MsgVote{
		Voter: a, DaoId: daoID, ProposalId: pid, Option: types.VoteOption_VOTE_OPTION_YES,
	})
	suite.Require().NoError(err)

	// Before voting_end: still VOTING.
	p, _ := suite.k.GetProposal(suite.ctx, daoID, pid)
	suite.Require().Equal(types.ProposalStatus_PROPOSAL_STATUS_VOTING, p.Status)

	// PASSED at end-block → refund 1ubze to proposer.
	suite.expectRefundOnTerminal(daoID, a, sdk.NewInt64Coin("ubze", 1))

	suite.advanceTime(48 * time.Hour)
	suite.Require().NoError(suite.k.EndBlock(suite.ctx))

	p, _ = suite.k.GetProposal(suite.ctx, daoID, pid)
	// yes/total = 1/2 = 50%, NOT strictly >= 50% in bps math: 1*10000 = 10000;
	// threshold*total = 5000*2 = 10000. yes*10000 >= threshold*total is TRUE.
	// So this passes. Good — also exercises the "passes at boundary" case.
	suite.Require().Equal(types.ProposalStatus_PROPOSAL_STATUS_PASSED, p.Status)
}

// TestEndBlock_BeforeExpiry_NoChange: end-blocker doesn't touch proposals
// whose voting_end is in the future.
func (suite *IntegrationTestSuite) TestEndBlock_BeforeExpiry_NoChange() {
	a := freshAddr()
	gov := validGovernance()
	gov.AllowRevote = true
	daoID, _ := suite.createDaoWithMembers("end-block-early",
		[]types.StaticMember{{Address: a, Weight: 1}}, gov)
	pid := suite.createTestProposal(daoID, a)

	// Advance time but not past voting_end.
	suite.advanceTime(time.Minute)
	suite.Require().NoError(suite.k.EndBlock(suite.ctx))

	p, _ := suite.k.GetProposal(suite.ctx, daoID, pid)
	suite.Require().Equal(types.ProposalStatus_PROPOSAL_STATUS_VOTING, p.Status)
}

// TestEndBlock_MultipleProposalsSameBlock: all proposals whose voting_end
// is in the past finalize in a single end-blocker call.
func (suite *IntegrationTestSuite) TestEndBlock_MultipleProposalsSameBlock() {
	a := freshAddr()
	gov := validGovernance()
	gov.AllowRevote = true
	daoID, _ := suite.createDaoWithMembers("multi-end",
		[]types.StaticMember{{Address: a, Weight: 1}}, gov)

	p1 := suite.createTestProposal(daoID, a)
	p2 := suite.createTestProposal(daoID, a)
	p3 := suite.createTestProposal(daoID, a)

	// Vote yes on p2 only — p1/p3 will REJECT (no votes), p2 will PASS.
	_, err := suite.msgServer.Vote(suite.ctx, &types.MsgVote{
		Voter: a, DaoId: daoID, ProposalId: p2, Option: types.VoteOption_VOTE_OPTION_YES,
	})
	suite.Require().NoError(err)

	// p1 + p3 REJECTED → 2 forfeits. p2 PASSED → 1 refund.
	suite.expectTreasuryForfeit(daoID, sdk.NewInt64Coin("ubze", 1))
	suite.expectRefundOnTerminal(daoID, a, sdk.NewInt64Coin("ubze", 1))
	suite.expectTreasuryForfeit(daoID, sdk.NewInt64Coin("ubze", 1))

	suite.advanceTime(48 * time.Hour)
	suite.Require().NoError(suite.k.EndBlock(suite.ctx))

	got1, _ := suite.k.GetProposal(suite.ctx, daoID, p1)
	got2, _ := suite.k.GetProposal(suite.ctx, daoID, p2)
	got3, _ := suite.k.GetProposal(suite.ctx, daoID, p3)
	suite.Require().Equal(types.ProposalStatus_PROPOSAL_STATUS_REJECTED, got1.Status)
	suite.Require().Equal(types.ProposalStatus_PROPOSAL_STATUS_PASSED, got2.Status)
	suite.Require().Equal(types.ProposalStatus_PROPOSAL_STATUS_REJECTED, got3.Status)
}

// TestEndBlock_DequeuesAfterFinalize: a finalized proposal is removed from
// the ExpiringProposalKey queue, so a subsequent EndBlock is a no-op.
func (suite *IntegrationTestSuite) TestEndBlock_DequeuesAfterFinalize() {
	a := freshAddr()
	daoID, _ := suite.createDaoWithMembers("dequeue",
		[]types.StaticMember{{Address: a, Weight: 1}}, validGovernance())
	pid := suite.createTestProposal(daoID, a)

	// REJECTED at end-block → forfeit to treasury.
	suite.expectTreasuryForfeit(daoID, sdk.NewInt64Coin("ubze", 1))

	suite.advanceTime(48 * time.Hour)
	suite.Require().NoError(suite.k.EndBlock(suite.ctx))
	suite.Require().NoError(suite.k.EndBlock(suite.ctx)) // double-fire must be safe

	p, _ := suite.k.GetProposal(suite.ctx, daoID, pid)
	// Voting period elapsed without votes → REJECTED.
	suite.Require().Equal(types.ProposalStatus_PROPOSAL_STATUS_REJECTED, p.Status)
}
