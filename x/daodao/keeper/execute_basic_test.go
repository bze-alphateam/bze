package keeper_test

import (
	"errors"

	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/bze-alphateam/bze/x/daodao/types"
)

// stagePassedProposal creates a proposal with the supplied msgs[] and
// drives it to PASSED via a single-member YES vote with revote disabled
// (so early-close fires PASSED inline). Returns the proposal id; the
// caller then triggers MsgExecuteProposal.
func (suite *IntegrationTestSuite) stagePassedProposal(daoID uint64, proposer string, msgs []*cdctypes.Any) uint64 {
	// Refund-on-pass mock for the early-close terminal disbursement.
	suite.expectRefundOnTerminal(daoID, proposer, sdk.NewInt64Coin("ubze", 1))
	// Bank send for the initial deposit.
	suite.expectInitialDepositSend(daoID, proposer, sdk.NewInt64Coin("ubze", 1))

	resp, err := suite.msgServer.CreateProposal(suite.ctx, &types.MsgCreateProposal{
		Proposer:       proposer,
		DaoId:          daoID,
		Title:          "stage",
		InitialDeposit: sdk.NewInt64Coin("ubze", 1),
		Msgs:           msgs,
	})
	suite.Require().NoError(err)

	_, err = suite.msgServer.Vote(suite.ctx, &types.MsgVote{
		Voter:      proposer,
		DaoId:      daoID,
		ProposalId: resp.ProposalId,
		Option:     types.VoteOption_VOTE_OPTION_YES,
	})
	suite.Require().NoError(err)

	p, _ := suite.k.GetProposal(suite.ctx, daoID, resp.ProposalId)
	suite.Require().Equal(types.ProposalStatus_PROPOSAL_STATUS_PASSED, p.Status)
	return resp.ProposalId
}

// daoWithExecuteGov creates a DAO whose governance config triggers
// early-close PASS on a single voter's YES (revote disabled, 50%
// threshold, single member). Returns (daoID, admin/member).
func (suite *IntegrationTestSuite) daoWithExecuteGov(name string) (uint64, string) {
	gov := validGovernance()
	gov.AllowRevote = false // enable early-close
	gov.ThresholdBps = 5_000
	creator := freshAddr()
	suite.expectAccountCreated(suite.k.GetDaoIDCounter(suite.ctx))
	resp, err := suite.msgServer.CreateDao(suite.ctx, &types.MsgCreateDao{
		Creator:      creator,
		Metadata:     sampleMetadata(name),
		VotingConfig: staticConfig(creator),
		Governance:   gov,
		Deposit:      validDeposit(),
	})
	suite.Require().NoError(err)
	return resp.DaoId, creator
}

// TestExecute_HappyPath_SingleMsg: one msg in the bundle dispatches
// successfully → status flips PASSED → EXECUTED.
func (suite *IntegrationTestSuite) TestExecute_HappyPath_SingleMsg() {
	daoID, admin := suite.daoWithExecuteGov("exec-happy")
	daoAddr := types.DaoAccountAddress(daoID).String()

	// Use MsgUpdateDaoMetadata as the bundle msg. Signer = DAO; fake
	// router returns success.
	bundleMsg := &types.MsgUpdateDaoMetadata{
		Authority: daoAddr,
		DaoId:     daoID,
		Metadata:  sampleMetadata("renamed-on-execute"),
	}
	pid := suite.stagePassedProposal(daoID, admin, []*cdctypes.Any{suite.mustPackAny(bundleMsg)})

	router := newFakeMsgRouter().withHandler(bundleMsg,
		func(ctx sdk.Context, req sdk.Msg) (*sdk.Result, error) {
			return &sdk.Result{}, nil
		})
	suite.installRouter(router)

	_, err := suite.msgServer.ExecuteProposal(suite.ctx, &types.MsgExecuteProposal{
		Executor:   freshAddr(),
		DaoId:      daoID,
		ProposalId: pid,
	})
	suite.Require().NoError(err)

	p, _ := suite.k.GetProposal(suite.ctx, daoID, pid)
	suite.Require().Equal(types.ProposalStatus_PROPOSAL_STATUS_EXECUTED, p.Status)
	suite.Require().Len(router.invocations, 1)
}

// TestExecute_MultiMsg_AllSucceed: bundle of two msgs both succeed →
// EXECUTED.
func (suite *IntegrationTestSuite) TestExecute_MultiMsg_AllSucceed() {
	daoID, admin := suite.daoWithExecuteGov("exec-multi")
	daoAddr := types.DaoAccountAddress(daoID).String()

	msg1 := &types.MsgUpdateDaoMetadata{
		Authority: daoAddr, DaoId: daoID, Metadata: sampleMetadata("step1"),
	}
	msg2 := &types.MsgUpdateDaoMetadata{
		Authority: daoAddr, DaoId: daoID, Metadata: sampleMetadata("step2"),
	}
	pid := suite.stagePassedProposal(daoID, admin,
		[]*cdctypes.Any{suite.mustPackAny(msg1), suite.mustPackAny(msg2)})

	router := newFakeMsgRouter().withHandler(msg1,
		func(ctx sdk.Context, req sdk.Msg) (*sdk.Result, error) {
			return &sdk.Result{}, nil
		})
	suite.installRouter(router)

	_, err := suite.msgServer.ExecuteProposal(suite.ctx, &types.MsgExecuteProposal{
		Executor:   freshAddr(),
		DaoId:      daoID,
		ProposalId: pid,
	})
	suite.Require().NoError(err)

	p, _ := suite.k.GetProposal(suite.ctx, daoID, pid)
	suite.Require().Equal(types.ProposalStatus_PROPOSAL_STATUS_EXECUTED, p.Status)
	suite.Require().Len(router.invocations, 2, "both bundle msgs dispatched in order")
}

// TestExecute_AtomicRollback_OnInnerFailure: bundle [A, B]; A succeeds,
// B fails. Status stays PASSED (retryable), and only A's invocation is
// recorded (B errored before completing).
func (suite *IntegrationTestSuite) TestExecute_AtomicRollback_OnInnerFailure() {
	daoID, admin := suite.daoWithExecuteGov("exec-rollback")
	daoAddr := types.DaoAccountAddress(daoID).String()

	msgA := &types.MsgUpdateDaoMetadata{
		Authority: daoAddr, DaoId: daoID, Metadata: sampleMetadata("A"),
	}
	msgB := &types.MsgUpdateDaoMetadata{
		Authority: daoAddr, DaoId: daoID, Metadata: sampleMetadata("B"),
	}
	pid := suite.stagePassedProposal(daoID, admin,
		[]*cdctypes.Any{suite.mustPackAny(msgA), suite.mustPackAny(msgB)})

	// Single registered handler; we differentiate A vs B by the metadata
	// name in the dispatched msg.
	innerErr := errors.New("simulated inner failure")
	router := newFakeMsgRouter().withHandler(msgA,
		func(ctx sdk.Context, req sdk.Msg) (*sdk.Result, error) {
			um := req.(*types.MsgUpdateDaoMetadata)
			if um.Metadata.Name == "B" {
				return nil, innerErr
			}
			return &sdk.Result{}, nil
		})
	suite.installRouter(router)

	_, err := suite.msgServer.ExecuteProposal(suite.ctx, &types.MsgExecuteProposal{
		Executor:   freshAddr(),
		DaoId:      daoID,
		ProposalId: pid,
	})
	suite.Require().Error(err)
	suite.Require().ErrorIs(err, innerErr)

	// Both A and B were invoked (B is what failed). Status MUST stay PASSED.
	suite.Require().Len(router.invocations, 2,
		"failing message also counts as an invocation; rollback drops its writes, not its observation")

	p, _ := suite.k.GetProposal(suite.ctx, daoID, pid)
	suite.Require().Equal(types.ProposalStatus_PROPOSAL_STATUS_PASSED, p.Status,
		"failing inner msg must leave proposal at PASSED (retryable)")
}

// TestExecute_RetryAfterFix: a failed execution can be retried after the
// underlying precondition is fixed. We simulate "fix" by re-installing a
// router whose handler now returns success.
func (suite *IntegrationTestSuite) TestExecute_RetryAfterFix() {
	daoID, admin := suite.daoWithExecuteGov("exec-retry")
	daoAddr := types.DaoAccountAddress(daoID).String()

	bundleMsg := &types.MsgUpdateDaoMetadata{
		Authority: daoAddr, DaoId: daoID, Metadata: sampleMetadata("retry"),
	}
	pid := suite.stagePassedProposal(daoID, admin, []*cdctypes.Any{suite.mustPackAny(bundleMsg)})

	// First execute: handler errors.
	failingRouter := newFakeMsgRouter().withHandler(bundleMsg,
		func(ctx sdk.Context, req sdk.Msg) (*sdk.Result, error) {
			return nil, errors.New("first attempt fails")
		})
	suite.installRouter(failingRouter)
	_, err := suite.msgServer.ExecuteProposal(suite.ctx, &types.MsgExecuteProposal{
		Executor: freshAddr(), DaoId: daoID, ProposalId: pid,
	})
	suite.Require().Error(err)

	p, _ := suite.k.GetProposal(suite.ctx, daoID, pid)
	suite.Require().Equal(types.ProposalStatus_PROPOSAL_STATUS_PASSED, p.Status)

	// Fix: swap in a successful router and retry.
	successRouter := newFakeMsgRouter().withHandler(bundleMsg,
		func(ctx sdk.Context, req sdk.Msg) (*sdk.Result, error) {
			return &sdk.Result{}, nil
		})
	suite.installRouter(successRouter)
	_, err = suite.msgServer.ExecuteProposal(suite.ctx, &types.MsgExecuteProposal{
		Executor: freshAddr(), DaoId: daoID, ProposalId: pid,
	})
	suite.Require().NoError(err)

	p, _ = suite.k.GetProposal(suite.ctx, daoID, pid)
	suite.Require().Equal(types.ProposalStatus_PROPOSAL_STATUS_EXECUTED, p.Status)
}

// TestExecute_NonPassedRejected: MsgExecuteProposal on a VOTING-status
// proposal is rejected.
func (suite *IntegrationTestSuite) TestExecute_NonPassedRejected() {
	daoID, admin := suite.createSampleDao("exec-not-passed")
	pid := suite.createTestProposal(daoID, admin) // stays VOTING

	_, err := suite.msgServer.ExecuteProposal(suite.ctx, &types.MsgExecuteProposal{
		Executor: freshAddr(), DaoId: daoID, ProposalId: pid,
	})
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "not in passed status")
}

// TestExecute_NoRouterWired: if SetMsgRouter was never called, execution
// fails with a clear "not wired" error rather than panicking.
func (suite *IntegrationTestSuite) TestExecute_NoRouterWired() {
	daoID, admin := suite.daoWithExecuteGov("exec-no-router")
	daoAddr := types.DaoAccountAddress(daoID).String()
	bundleMsg := &types.MsgUpdateDaoMetadata{
		Authority: daoAddr, DaoId: daoID, Metadata: sampleMetadata("no-router"),
	}
	pid := suite.stagePassedProposal(daoID, admin, []*cdctypes.Any{suite.mustPackAny(bundleMsg)})

	// Deliberately DON'T install a router.
	_, err := suite.msgServer.ExecuteProposal(suite.ctx, &types.MsgExecuteProposal{
		Executor: freshAddr(), DaoId: daoID, ProposalId: pid,
	})
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "msgRouter is not wired")
}
