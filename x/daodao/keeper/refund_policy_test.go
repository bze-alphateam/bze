package keeper_test

import (
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"go.uber.org/mock/gomock"

	"github.com/bze-alphateam/bze/x/daodao/types"
)

// makeDaoWithPolicy is a helper that creates a single-member DAO with a
// chosen RefundPolicy / ForfeitDestination. min_deposit = 1ubze, voting
// period 1h, revote DISABLED so a sole YES early-closes as PASSED.
func (suite *IntegrationTestSuite) makeDaoWithPolicy(name string, refund types.RefundPolicy, forfeit types.ForfeitDestination) (daoID uint64, member string) {
	gov := validGovernance()
	gov.AllowRevote = false
	gov.VotingPeriod = time.Hour
	gov.ThresholdBps = 5_000

	dep := types.DepositConfig{
		MinDeposit:         sdk.NewInt64Coin("ubze", 1),
		DepositPeriod:      24 * time.Hour,
		ForfeitDestination: forfeit,
		VotingRefundPolicy: refund,
	}

	member = freshAddr()
	suite.expectAccountCreated(suite.k.GetDaoIDCounter(suite.ctx))
	resp, err := suite.msgServer.CreateDao(suite.ctx, &types.MsgCreateDao{
		Creator:      member,
		Metadata:     sampleMetadata(name),
		VotingConfig: staticConfig(member),
		Governance:   gov,
		Deposit:      dep,
	})
	suite.Require().NoError(err)
	return resp.DaoId, member
}

// TestRefundPolicy_AlwaysRefundsOnPass exercises ALWAYS + PASSED outcome.
func (suite *IntegrationTestSuite) TestRefundPolicy_AlwaysRefundsOnPass() {
	daoID, member := suite.makeDaoWithPolicy("always-pass",
		types.RefundPolicy_REFUND_POLICY_ALWAYS,
		types.ForfeitDestination_FORFEIT_DESTINATION_TREASURY,
	)
	memberAddr := suite.mustAcc(member)

	suite.expectInitialDepositSend(daoID, member, sdk.NewInt64Coin("ubze", 1))
	resp, err := suite.msgServer.CreateProposal(suite.ctx, &types.MsgCreateProposal{
		Proposer:       member,
		DaoId:          daoID,
		Title:          "always-pass",
		InitialDeposit: sdk.NewInt64Coin("ubze", 1),
	})
	suite.Require().NoError(err)

	// Expect the refund: escrow → member, 1ubze.
	escrow := types.DepositEscrowAddress(daoID)
	suite.bank.EXPECT().
		SendCoins(gomock.Any(), escrow, memberAddr, sdk.NewCoins(sdk.NewInt64Coin("ubze", 1))).
		Return(nil).
		Times(1)

	// YES vote → early-close PASS.
	_, err = suite.msgServer.Vote(suite.ctx, &types.MsgVote{
		Voter: member, DaoId: daoID, ProposalId: resp.ProposalId,
		Option: types.VoteOption_VOTE_OPTION_YES,
	})
	suite.Require().NoError(err)

	p, _ := suite.k.GetProposal(suite.ctx, daoID, resp.ProposalId)
	suite.Require().Equal(types.ProposalStatus_PROPOSAL_STATUS_PASSED, p.Status)
}

// TestRefundPolicy_OnPassForfeitsOnReject: ON_PASS + REJECTED outcome
// forfeits to treasury.
func (suite *IntegrationTestSuite) TestRefundPolicy_OnPassForfeitsOnReject() {
	daoID, member := suite.makeDaoWithPolicy("onpass-reject",
		types.RefundPolicy_REFUND_POLICY_ON_PASS,
		types.ForfeitDestination_FORFEIT_DESTINATION_TREASURY,
	)

	suite.expectInitialDepositSend(daoID, member, sdk.NewInt64Coin("ubze", 1))
	resp, err := suite.msgServer.CreateProposal(suite.ctx, &types.MsgCreateProposal{
		Proposer:       member,
		DaoId:          daoID,
		Title:          "onpass-reject",
		InitialDeposit: sdk.NewInt64Coin("ubze", 1),
	})
	suite.Require().NoError(err)

	// Expect treasury forfeit when the proposal rejects at end-block.
	dao, _ := suite.k.GetDao(suite.ctx, daoID)
	daoAddr := suite.mustAcc(dao.AccountAddress)
	escrow := types.DepositEscrowAddress(daoID)
	suite.bank.EXPECT().
		SendCoins(gomock.Any(), escrow, daoAddr, sdk.NewCoins(sdk.NewInt64Coin("ubze", 1))).
		Return(nil).
		Times(1)

	// NO vote → early-close REJECT (threshold 50%, single voter at 1; voting NO
	// is unreachable threshold for yes-half).
	_, err = suite.msgServer.Vote(suite.ctx, &types.MsgVote{
		Voter: member, DaoId: daoID, ProposalId: resp.ProposalId,
		Option: types.VoteOption_VOTE_OPTION_NO,
	})
	suite.Require().NoError(err)

	p, _ := suite.k.GetProposal(suite.ctx, daoID, resp.ProposalId)
	suite.Require().Equal(types.ProposalStatus_PROPOSAL_STATUS_REJECTED, p.Status)
}

// TestRefundPolicy_NeverForfeitsOnPass: NEVER + PASSED forfeits anyway.
func (suite *IntegrationTestSuite) TestRefundPolicy_NeverForfeitsOnPass() {
	daoID, member := suite.makeDaoWithPolicy("never-pass",
		types.RefundPolicy_REFUND_POLICY_NEVER,
		types.ForfeitDestination_FORFEIT_DESTINATION_TREASURY,
	)

	suite.expectInitialDepositSend(daoID, member, sdk.NewInt64Coin("ubze", 1))
	resp, err := suite.msgServer.CreateProposal(suite.ctx, &types.MsgCreateProposal{
		Proposer:       member,
		DaoId:          daoID,
		Title:          "never-pass",
		InitialDeposit: sdk.NewInt64Coin("ubze", 1),
	})
	suite.Require().NoError(err)

	dao, _ := suite.k.GetDao(suite.ctx, daoID)
	daoAddr := suite.mustAcc(dao.AccountAddress)
	escrow := types.DepositEscrowAddress(daoID)
	// Despite PASSED, the proposer's deposit is sent to treasury.
	suite.bank.EXPECT().
		SendCoins(gomock.Any(), escrow, daoAddr, sdk.NewCoins(sdk.NewInt64Coin("ubze", 1))).
		Return(nil).
		Times(1)

	_, err = suite.msgServer.Vote(suite.ctx, &types.MsgVote{
		Voter: member, DaoId: daoID, ProposalId: resp.ProposalId,
		Option: types.VoteOption_VOTE_OPTION_YES,
	})
	suite.Require().NoError(err)

	p, _ := suite.k.GetProposal(suite.ctx, daoID, resp.ProposalId)
	suite.Require().Equal(types.ProposalStatus_PROPOSAL_STATUS_PASSED, p.Status)
}
