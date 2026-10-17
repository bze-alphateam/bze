package keeper_test

import (
	"strconv"
	"time"

	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"go.uber.org/mock/gomock"

	"github.com/bze-alphateam/bze/x/dao/types"
)

var settlerAddr = sdk.AccAddress("dao_settler_________")

func (suite *IntegrationTestSuite) settle(proposalId uint64) (types.ProposalStatus, error) {
	res, err := suite.msgServer.SettleProposal(suite.ctx, types.NewMsgSettleProposal(settlerAddr.String(), proposalId))
	if err != nil {
		return 0, err
	}

	return res.Status, nil
}

// endVoting moves the block time to the proposal's voting end, plus offset seconds.
func (suite *IntegrationTestSuite) endVoting(proposalId uint64, offset int64) {
	proposal, found := suite.k.GetProposal(suite.ctx, proposalId)
	suite.Require().True(found)
	suite.ctx = suite.ctx.WithBlockTime(time.Unix(proposal.VotingEnd+offset, 0))
}

// requireStatus checks the stored status and the last EventProposalSettled.
func (suite *IntegrationTestSuite) requireStatus(proposalId uint64, status types.ProposalStatus) {
	proposal, found := suite.k.GetProposal(suite.ctx, proposalId)
	suite.Require().True(found)
	suite.Require().Equal(status, proposal.Status)
	suite.Require().False(suite.k.IsOpenProposal(suite.ctx, proposal.DaoId, proposalId))

	attrs := suite.lastEventAttrs(&types.EventProposalSettled{})
	suite.Require().Equal(`"`+strconv.FormatUint(proposalId, 10)+`"`, attrs["id"])
	suite.Require().Equal(`"`+status.String()+`"`, attrs["status"])
	suite.Require().Equal("false", attrs["expedited_early"])
}

// requireExecuted checks the last EventProposalExecuted: success, or the fixed reason.
func (suite *IntegrationTestSuite) requireExecuted(reason string) {
	attrs := suite.lastEventAttrs(&types.EventProposalExecuted{})
	suite.Require().NotNil(attrs)
	suite.Require().Equal(strconv.FormatBool(reason == ""), attrs["success"])
	suite.Require().Equal(`"`+reason+`"`, attrs["error"])
}

// passedProposal submits msgs to the static DAO 1 and has both members vote yes.
func (suite *IntegrationTestSuite) passedProposal(msgs ...sdk.Msg) uint64 {
	id, err := suite.submit(memberA, 1, false, msgs...)
	suite.Require().NoError(err)
	suite.Require().NoError(suite.vote(memberA, id, types.VoteOption_YES))
	suite.Require().NoError(suite.vote(memberB, id, types.VoteOption_YES))

	return id
}

// A passed proposal spends the treasury as the DAO: EXECUTED, the funds and
// the messages' events are there, and the proposal left voting.
func (suite *IntegrationTestSuite) TestSettleProposal_PassedExecutes() {
	suite.createStaticDao(noConfigChange)
	suite.ledger.fund(daoAddress(1), ubze(100))
	id := suite.passedProposal(treasurySend(memberB, 30))
	suite.Require().True(suite.k.IsOpenProposal(suite.ctx, 1, id))

	suite.endVoting(id, 0)
	status, err := suite.settle(id)
	suite.Require().NoError(err)
	suite.Require().Equal(types.ProposalStatus_EXECUTED, status)

	suite.requireStatus(id, types.ProposalStatus_EXECUTED)
	suite.requireExecuted("")
	suite.Require().Equal(ubze(70), suite.ledger.balance(daoAddress(1)))
	suite.Require().Equal(ubze(30), suite.ledger.balance(memberB))
	suite.Require().True(suite.hasEvent(fakeSendEvent))

	attrs := suite.lastEventAttrs(&types.EventProposalSettled{})
	suite.Require().Equal(`"2"`, attrs["yes"])
	suite.Require().Equal(`"0"`, attrs["no"])
	suite.Require().Equal(`"0"`, attrs["abstain"])
	suite.Require().Equal(`"2"`, attrs["denominator"])
}

// A passed proposal with no message ends EXECUTED.
func (suite *IntegrationTestSuite) TestSettleProposal_SignalProposalExecuted() {
	suite.createStaticDao(noConfigChange)
	id := suite.passedProposal()

	suite.endVoting(id, 0)
	status, err := suite.settle(id)
	suite.Require().NoError(err)
	suite.Require().Equal(types.ProposalStatus_EXECUTED, status)
	suite.requireExecuted("")
}

// A rejected proposal runs nothing.
func (suite *IntegrationTestSuite) TestSettleProposal_RejectedRunsNothing() {
	suite.createStaticDao(noConfigChange)
	suite.ledger.fund(daoAddress(1), ubze(100))
	id, err := suite.submit(memberA, 1, false, treasurySend(memberB, 30))
	suite.Require().NoError(err)
	suite.Require().NoError(suite.vote(memberA, id, types.VoteOption_NO))

	suite.endVoting(id, 0)
	status, err := suite.settle(id)
	suite.Require().NoError(err)
	suite.Require().Equal(types.ProposalStatus_REJECTED, status)
	suite.requireStatus(id, types.ProposalStatus_REJECTED)
	suite.Require().Nil(suite.lastEventAttrs(&types.EventProposalExecuted{}))
	suite.Require().Equal(ubze(100), suite.ledger.balance(daoAddress(1)))
}

// D16: a proposal only abstained on, or not voted on at all, is rejected.
func (suite *IntegrationTestSuite) TestSettleProposal_D16_NoDecidedVotes() {
	suite.createStaticDao(func(c *types.DaoConfig) { c.MinQuorum = dec("0.1") })
	abstained, err := suite.submit(memberA, 1, false)
	suite.Require().NoError(err)
	suite.Require().NoError(suite.vote(memberA, abstained, types.VoteOption_ABSTAIN))
	suite.Require().NoError(suite.vote(memberB, abstained, types.VoteOption_ABSTAIN))
	empty, err := suite.submit(memberA, 1, false)
	suite.Require().NoError(err)

	suite.endVoting(empty, 0)
	for _, id := range []uint64{abstained, empty} {
		status, err := suite.settle(id)
		suite.Require().NoError(err)
		suite.Require().Equal(types.ProposalStatus_REJECTED, status)
	}
}

// D18: voting closes at voting_end; a settle one second earlier is refused.
func (suite *IntegrationTestSuite) TestSettleProposal_D18_AtVotingEnd() {
	suite.createStaticDao(noConfigChange)
	id := suite.passedProposal()

	suite.endVoting(id, -1)
	_, err := suite.settle(id)
	suite.Require().ErrorIs(err, types.ErrVotingNotEnded)
	proposal, _ := suite.k.GetProposal(suite.ctx, id)
	suite.Require().Equal(types.ProposalStatus_VOTING, proposal.Status)
	suite.Require().True(suite.k.IsOpenProposal(suite.ctx, 1, id))

	suite.endVoting(id, 0)
	_, err = suite.settle(id)
	suite.Require().NoError(err)
}

// Unknown proposals and invalid senders are refused; a finished proposal
// cannot be settled again.
func (suite *IntegrationTestSuite) TestSettleProposal_Refusals() {
	suite.createStaticDao(noConfigChange)

	_, err := suite.settle(1)
	suite.Require().ErrorIs(err, types.ErrProposalNotFound)
	_, err = suite.msgServer.SettleProposal(suite.ctx, &types.MsgSettleProposal{Sender: "bze1nope", ProposalId: 1})
	suite.Require().ErrorIs(err, sdkerrors.ErrInvalidAddress)

	executed := suite.passedProposal()
	rejected, err := suite.submit(memberA, 1, false)
	suite.Require().NoError(err)
	cancelled, err := suite.submit(memberA, 1, false)
	suite.Require().NoError(err)
	suite.Require().NoError(suite.cancel(memberA, cancelled))

	suite.endVoting(executed, 0)
	for _, id := range []uint64{executed, rejected} {
		_, err := suite.settle(id)
		suite.Require().NoError(err)
	}
	for _, id := range []uint64{executed, rejected, cancelled} {
		_, err := suite.settle(id)
		suite.Require().ErrorIs(err, types.ErrProposalFinished)
	}
}

// Deposit matrix (Business Logic §8), with I1: a passed proposal refunds unless
// the policy is NEVER, also when its execution fails (D13); a rejected one
// refunds only under ALWAYS. After the settle the escrow is empty.
func (suite *IntegrationTestSuite) TestSettleProposal_DepositMatrix() {
	type outcome int
	const (
		passed outcome = iota
		failed
		rejected
	)
	burner := authtypes.NewModuleAddress(burnerFeeCollector)
	for _, tc := range []struct {
		desc     string
		policy   types.DepositRefundPolicy
		outcome  outcome
		refunded bool
		status   types.ProposalStatus
	}{
		{"always/passed", types.DepositRefundPolicy_ALWAYS, passed, true, types.ProposalStatus_EXECUTED},
		{"always/failed", types.DepositRefundPolicy_ALWAYS, failed, true, types.ProposalStatus_EXECUTION_FAILED},
		{"always/rejected", types.DepositRefundPolicy_ALWAYS, rejected, true, types.ProposalStatus_REJECTED},
		{"only passed/passed", types.DepositRefundPolicy_ONLY_PASSED, passed, true, types.ProposalStatus_EXECUTED},
		{"only passed/failed (D13)", types.DepositRefundPolicy_ONLY_PASSED, failed, true, types.ProposalStatus_EXECUTION_FAILED},
		{"only passed/rejected", types.DepositRefundPolicy_ONLY_PASSED, rejected, false, types.ProposalStatus_REJECTED},
		{"never/passed", types.DepositRefundPolicy_NEVER, passed, false, types.ProposalStatus_EXECUTED},
		{"never/failed", types.DepositRefundPolicy_NEVER, failed, false, types.ProposalStatus_EXECUTION_FAILED},
		{"never/rejected", types.DepositRefundPolicy_NEVER, rejected, false, types.ProposalStatus_REJECTED},
	} {
		suite.Run(tc.desc, func() {
			suite.SetupTest()
			suite.createStaticDao(func(c *types.DaoConfig) {
				c.ProposalDeposit = coinPtr(40)
				c.DepositRefundPolicy = tc.policy
			})
			escrow := sdk.NewCoins()
			suite.bank.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), memberA, types.ModuleName, ubze(40)).
				DoAndReturn(func(_ any, _ sdk.AccAddress, _ string, amt sdk.Coins) error {
					escrow = escrow.Add(amt...)
					return nil
				})
			release := func(_ any, _ string, _ any, amt sdk.Coins) error {
				escrow = escrow.Sub(amt...)
				return nil
			}
			if tc.refunded {
				suite.bank.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, memberA, ubze(40)).DoAndReturn(release)
			} else {
				suite.bank.EXPECT().SendCoinsFromModuleToModule(gomock.Any(), types.ModuleName, burnerFeeCollector, ubze(40)).DoAndReturn(release)
			}

			// the treasury is empty, so the spend fails
			id, err := suite.submit(memberA, 1, false, treasurySend(memberB, 5))
			suite.Require().NoError(err)
			if tc.outcome == passed {
				suite.ledger.fund(daoAddress(1), ubze(5))
			}
			option := types.VoteOption_YES
			if tc.outcome == rejected {
				option = types.VoteOption_NO
			}
			suite.Require().NoError(suite.vote(memberA, id, option))
			suite.Require().NoError(suite.vote(memberB, id, option))

			suite.endVoting(id, 0)
			status, err := suite.settle(id)
			suite.Require().NoError(err)
			suite.Require().Equal(tc.status, status)
			suite.requireStatus(id, tc.status)
			suite.Require().True(escrow.IsZero(), "I1")

			to := burner
			if tc.refunded {
				to = memberA
			}
			attrs := suite.lastEventAttrs(&types.EventDepositReleased{})
			suite.Require().Equal(strconv.FormatBool(tc.refunded), attrs["refunded"])
			suite.Require().Equal(`"`+to.String()+`"`, attrs["destination"])
		})
	}
}

// A failed release fails the settle: the proposal stays in voting.
func (suite *IntegrationTestSuite) TestSettleProposal_ReleaseFailureKeepsVoting() {
	suite.createStaticDao(func(c *types.DaoConfig) { c.ProposalDeposit = coinPtr(40) })
	suite.bank.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), memberA, types.ModuleName, ubze(40)).Return(nil)
	id := suite.passedProposal()

	suite.bank.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, memberA, ubze(40)).Return(sdkerrors.ErrInsufficientFunds)
	suite.endVoting(id, 0)
	_, err := suite.settle(id)
	suite.Require().ErrorIs(err, sdkerrors.ErrInsufficientFunds)
}

// Atomicity: when the second message fails, the first one's effects are gone
// (here a proposal the DAO submitted), and so are its events.
func (suite *IntegrationTestSuite) TestSettleProposal_FailureDiscardsEveryMessage() {
	suite.createStaticDao(noConfigChange)
	suite.ledger.fund(daoAddress(1), ubze(10))
	inner, err := types.NewMsgSubmitProposal(daoAddress(1).String(), 1, "inner", "", nil, false)
	suite.Require().NoError(err)
	id := suite.passedProposal(inner, treasurySend(memberB, 500))

	suite.endVoting(id, 0)
	suite.ctx = suite.ctx.WithEventManager(sdk.NewEventManager())
	status, err := suite.settle(id)
	suite.Require().NoError(err)
	suite.Require().Equal(types.ProposalStatus_EXECUTION_FAILED, status)
	suite.requireStatus(id, types.ProposalStatus_EXECUTION_FAILED)
	suite.requireExecuted("message failed")

	suite.Require().Equal(id, suite.k.GetProposalCounter(suite.ctx))
	suite.Require().Nil(suite.lastEventAttrs(&types.EventProposalSubmitted{}))
	suite.Require().Equal(ubze(10), suite.ledger.balance(daoAddress(1)))
}

// D1 (execution part): a message that passes the submit check can still fail
// the signer check when it runs. DAO 1 administers DAO 2 and proposes an admin
// execute on it whose inner send spends DAO 1's own treasury: only the
// executor of DAO 2 sees the wrong signer.
func (suite *IntegrationTestSuite) TestSettleProposal_D1_SignerCheckedAtExecution() {
	suite.createStaticDao(noConfigChange)
	suite.createDao(2, func(msg *types.MsgCreateDao) { msg.Admin = daoAddress(1).String() })
	suite.ledger.fund(daoAddress(1), ubze(10))
	nested, err := types.NewMsgAdminExecute(daoAddress(1).String(), 2, []sdk.Msg{treasurySend(memberB, 10)})
	suite.Require().NoError(err)
	id := suite.passedProposal(nested)

	suite.endVoting(id, 0)
	status, err := suite.settle(id)
	suite.Require().NoError(err)
	suite.Require().Equal(types.ProposalStatus_EXECUTION_FAILED, status)
	suite.requireExecuted("unauthorized signer")
	suite.Require().Equal(ubze(10), suite.ledger.balance(daoAddress(1)))
}

// D8: a proposal that settles itself finds itself EXECUTED and fails cleanly;
// the outer settle ends EXECUTION_FAILED with one deposit release.
func (suite *IntegrationTestSuite) TestSettleProposal_D8_SelfSettle() {
	suite.createStaticDao(func(c *types.DaoConfig) { c.ProposalDeposit = coinPtr(40) })
	suite.bank.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), memberA, types.ModuleName, ubze(40)).Return(nil)
	suite.bank.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, memberA, ubze(40)).Return(nil).Times(1)
	id := suite.passedProposal(types.NewMsgSettleProposal(daoAddress(1).String(), 1))
	suite.Require().Equal(uint64(1), id)

	suite.endVoting(id, 0)
	status, err := suite.settle(id)
	suite.Require().NoError(err)
	suite.Require().Equal(types.ProposalStatus_EXECUTION_FAILED, status)
	suite.requireStatus(id, types.ProposalStatus_EXECUTION_FAILED)
	suite.requireExecuted("message failed")
}

// A failed execution is retried by any later settle within the window: it stays
// EXECUTION_FAILED while the cause remains and ends EXECUTED once the treasury
// is funded; the deposit is not released again.
func (suite *IntegrationTestSuite) TestSettleProposal_RetrySucceedsAfterFix() {
	suite.createStaticDao(func(c *types.DaoConfig) { c.ProposalDeposit = coinPtr(40) })
	suite.bank.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), memberA, types.ModuleName, ubze(40)).Return(nil)
	suite.bank.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, memberA, ubze(40)).Return(nil).Times(1)
	id := suite.passedProposal(treasurySend(memberB, 30))

	suite.endVoting(id, 0)
	status, err := suite.settle(id)
	suite.Require().NoError(err)
	suite.Require().Equal(types.ProposalStatus_EXECUTION_FAILED, status)

	suite.endVoting(id, 3600)
	status, err = suite.settle(id)
	suite.Require().NoError(err)
	suite.Require().Equal(types.ProposalStatus_EXECUTION_FAILED, status)
	suite.requireStatus(id, types.ProposalStatus_EXECUTION_FAILED)
	suite.requireExecuted("message failed")

	suite.ledger.fund(daoAddress(1), ubze(30))
	status, err = suite.settle(id)
	suite.Require().NoError(err)
	suite.Require().Equal(types.ProposalStatus_EXECUTED, status)
	suite.requireStatus(id, types.ProposalStatus_EXECUTED)
	suite.requireExecuted("")
	suite.Require().Equal(ubze(30), suite.ledger.balance(memberB))

	_, err = suite.settle(id)
	suite.Require().ErrorIs(err, types.ErrProposalFinished)
}

// The retry window ends at voting_end + execution_window: a settle one second
// before it still retries; from it on the proposal expires without running.
func (suite *IntegrationTestSuite) TestSettleProposal_ExpiresAfterWindow() {
	suite.createStaticDao(noConfigChange)
	id := suite.passedProposal(treasurySend(memberB, 30))
	window := int64(types.DefaultExecutionWindow.Seconds())

	suite.endVoting(id, 0)
	_, err := suite.settle(id)
	suite.Require().NoError(err)
	suite.endVoting(id, window-1)
	status, err := suite.settle(id)
	suite.Require().NoError(err)
	suite.Require().Equal(types.ProposalStatus_EXECUTION_FAILED, status)

	suite.ledger.fund(daoAddress(1), ubze(30))
	suite.endVoting(id, window)
	suite.ctx = suite.ctx.WithEventManager(sdk.NewEventManager())
	status, err = suite.settle(id)
	suite.Require().NoError(err)
	suite.Require().Equal(types.ProposalStatus_EXPIRED, status)
	suite.requireStatus(id, types.ProposalStatus_EXPIRED)
	suite.Require().Nil(suite.lastEventAttrs(&types.EventProposalExecuted{}))
	suite.Require().Equal(ubze(30), suite.ledger.balance(daoAddress(1)))

	_, err = suite.settle(id)
	suite.Require().ErrorIs(err, types.ErrProposalFinished)
}

// The window is read from the params at the time of the settle.
func (suite *IntegrationTestSuite) TestSettleProposal_ZeroWindowExpiresAtOnce() {
	suite.createStaticDao(noConfigChange)
	params := suite.k.GetParams(suite.ctx)
	params.ExecutionWindow = 0
	suite.Require().NoError(suite.k.SetParams(suite.ctx, params))
	id := suite.passedProposal(treasurySend(memberB, 30))

	suite.endVoting(id, 0)
	_, err := suite.settle(id)
	suite.Require().NoError(err)
	status, err := suite.settle(id)
	suite.Require().NoError(err)
	suite.Require().Equal(types.ProposalStatus_EXPIRED, status)
}

func (suite *IntegrationTestSuite) panickingMultiSend() *banktypes.MsgMultiSend {
	coins := ubze(5)
	return banktypes.NewMsgMultiSend(banktypes.NewInput(daoAddress(1), coins), []banktypes.Output{banktypes.NewOutput(memberB, coins)})
}

// A panicking message fails the execution like any error: EXECUTION_FAILED,
// the deposit handled as for a failure, and the earlier messages undone.
func (suite *IntegrationTestSuite) TestSettleProposal_PanicBecomesExecutionFailure() {
	suite.createStaticDao(func(c *types.DaoConfig) {
		c.ProposalDeposit = coinPtr(40)
		c.DepositRefundPolicy = types.DepositRefundPolicy_ONLY_PASSED
	})
	suite.bank.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), memberA, types.ModuleName, ubze(40)).Return(nil)
	suite.bank.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, memberA, ubze(40)).Return(nil)
	// the inner proposal's deposit, paid inside the discarded cache
	suite.bank.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), daoAddress(1), types.ModuleName, ubze(40)).Return(nil)
	inner, err := types.NewMsgSubmitProposal(daoAddress(1).String(), 1, "inner", "", nil, false)
	suite.Require().NoError(err)
	id := suite.passedProposal(inner, suite.panickingMultiSend())

	suite.endVoting(id, 0)
	status, err := suite.settle(id)
	suite.Require().NoError(err)
	suite.Require().Equal(types.ProposalStatus_EXECUTION_FAILED, status)
	suite.requireStatus(id, types.ProposalStatus_EXECUTION_FAILED)
	suite.requireExecuted("message panicked")
	suite.Require().Equal(id, suite.k.GetProposalCounter(suite.ctx))
}

// Out of gas is not swallowed: the settle panics, so its transaction fails and
// writes nothing.
func (suite *IntegrationTestSuite) TestSettleProposal_OutOfGasPropagates() {
	suite.createStaticDao(noConfigChange)
	id := suite.passedProposal(suite.panickingMultiSend())

	for _, value := range []any{storetypes.ErrorOutOfGas{Descriptor: "test"}, storetypes.ErrorGasOverflow{Descriptor: "test"}} {
		suite.ledger.panicWith = value
		suite.endVoting(id, 0)
		// the transaction's cache, written only on success as baseapp does
		txCtx, _ := suite.ctx.CacheContext()
		suite.Require().PanicsWithValue(value, func() {
			_, _ = suite.msgServer.SettleProposal(txCtx, types.NewMsgSettleProposal(settlerAddr.String(), id))
		})

		proposal, _ := suite.k.GetProposal(suite.ctx, id)
		suite.Require().Equal(types.ProposalStatus_VOTING, proposal.Status)
		suite.Require().True(suite.k.IsOpenProposal(suite.ctx, 1, id))
	}
}

// MsgAdminExecute shares the executor: a panicking message returns an error.
func (suite *IntegrationTestSuite) TestAdminExecute_PanicReturnsError() {
	suite.createAdminDao()

	var err error
	suite.Require().NotPanics(func() { err = suite.adminExecute(adminAddr, 1, suite.panickingMultiSend()) })
	suite.Require().ErrorIs(err, types.ErrExecutionPanic)
}

// D6 and I4 on a stake DAO: a whale joining mid-vote neither votes nor moves
// the pinned denominator, so the proposal still passes; the first settle drops
// the proposal's pins on its sources.
func (suite *IntegrationTestSuite) TestSettleProposal_D6_WhaleCannotMoveDenominator() {
	state := defaultRewardsState()
	suite.stubRewards(state)
	s := suite.stubStakes()
	suite.expectCreation(1)
	_, err := suite.msgServer.CreateDao(suite.ctx, validStakeDaoMsg([]string{"1"}, stakeDenom))
	suite.Require().NoError(err)
	s.joinSr("1", memberA, 70)
	s.joinDr(stakeDenom, memberA, 10)

	suite.atBlock(10)
	id, err := suite.submit(memberA, 1, false)
	suite.Require().NoError(err)
	suite.Require().NoError(suite.vote(memberA, id, types.VoteOption_YES))
	suite.Require().Equal(uint64(2), suite.k.GetSourcePins(suite.ctx, srSource))
	suite.Require().Equal(uint64(2), suite.k.GetSourcePins(suite.ctx, drSource))

	suite.atBlock(11)
	state.srs["1"] = stakingReward("1", stakeDenom, 5, "1000100")
	stakeSources[0].join(suite, s, memberC, 1_000_000)
	suite.Require().ErrorIs(suite.vote(memberC, id, types.VoteOption_NO), types.ErrNoVotingPower)

	suite.endVoting(id, 0)
	status, err := suite.settle(id)
	suite.Require().NoError(err)
	suite.Require().Equal(types.ProposalStatus_EXECUTED, status)
	proposal, _ := suite.k.GetProposal(suite.ctx, id)
	suite.Require().Equal(math.NewInt(150), proposal.PinnedDenominator)
	suite.Require().Equal(uint64(1), suite.k.GetSourcePins(suite.ctx, srSource))
	suite.Require().Equal(uint64(1), suite.k.GetSourcePins(suite.ctx, drSource))
}

// D19 (settle part): the settle reads the running tally and never the ballots,
// so its gas is the same with 10,000 ballots as with one of the same total.
func (suite *IntegrationTestSuite) TestSettleProposal_D19_GasIndependentOfBallots() {
	settleGas := func(voters int, weight int64) storetypes.Gas {
		suite.SetupTest()
		state := defaultRewardsState()
		state.srs["1"] = stakingReward("1", stakeDenom, 5, "20000")
		suite.stubRewards(state)
		s := suite.stubStakes()
		suite.expectCreation(1)
		_, err := suite.msgServer.CreateDao(suite.ctx, validStakeDaoMsg([]string{"1"}, ""))
		suite.Require().NoError(err)
		for i := 1; i <= voters; i++ {
			s.joinSr("1", memberAddress(i), weight)
		}

		suite.atBlock(10)
		id, err := suite.submit(memberAddress(1), 1, false)
		suite.Require().NoError(err)
		for i := 1; i <= voters; i++ {
			suite.Require().NoError(suite.vote(memberAddress(i), id, types.VoteOption_YES))
		}
		suite.requireTally(id, 10_000, 0, 0)

		suite.endVoting(id, 0)
		suite.ctx = suite.ctx.WithGasMeter(storetypes.NewInfiniteGasMeter())
		status, err := suite.settle(id)
		suite.Require().NoError(err)
		suite.Require().Equal(types.ProposalStatus_EXECUTED, status)

		return suite.ctx.GasMeter().GasConsumed()
	}

	one := settleGas(1, 10_000)
	many := settleGas(10_000, 1)
	suite.Require().Positive(one)
	suite.Require().Equal(one, many)
}
