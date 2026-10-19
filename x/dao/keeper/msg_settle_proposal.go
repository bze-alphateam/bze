package keeper

import (
	"context"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/bze-alphateam/bze/x/dao/types"
)

// Fixed EventProposalExecuted reasons: the raw error text is never put in an
// event, so events stay deterministic.
const (
	executionErrSigner  = "unauthorized signer"
	executionErrPanic   = "message panicked"
	executionErrRouting = "invalid message"
	executionErrFailed  = "message failed"
)

func (k msgServer) SettleProposal(goCtx context.Context, msg *types.MsgSettleProposal) (*types.MsgSettleProposalResponse, error) {
	// the router runs it too; repeated so the server is safe when called directly
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	proposal, found := k.GetProposal(ctx, msg.ProposalId)
	if !found {
		return nil, errorsmod.Wrapf(types.ErrProposalNotFound, "proposal %d", msg.ProposalId)
	}

	now := ctx.BlockTime().Unix()
	expeditedEarly := false
	var err error
	switch {
	case proposal.Status == types.ProposalStatus_VOTING && now >= proposal.VotingEnd:
		err = k.firstSettle(ctx, &proposal, proposal.RegularTallyPasses())
	case proposal.Status == types.ProposalStatus_VOTING && !proposal.Expedited:
		return nil, errorsmod.Wrapf(types.ErrVotingNotEnded, "proposal %d votes until %d", proposal.Id, proposal.VotingEnd)
	case proposal.Status == types.ProposalStatus_VOTING:
		// voters always get min_expedite_delay to react before an early settle
		if earliest := k.earliestExpediteSettle(ctx, proposal); now < earliest {
			return nil, errorsmod.Wrapf(types.ErrExpediteTooEarly, "proposal %d can settle early from %d", proposal.Id, earliest)
		}

		// Business Logic §7: an expedited proposal settles early only as passed
		if !proposal.ExpediteTallyPasses() {
			return nil, errorsmod.Wrapf(types.ErrExpediteBarNotMet, "proposal %d votes until %d", proposal.Id, proposal.VotingEnd)
		}
		expeditedEarly = true
		err = k.firstSettle(ctx, &proposal, true)
	case proposal.Status == types.ProposalStatus_EXECUTION_FAILED && now < k.executionDeadline(ctx, proposal):
		// a retry: the deposit was released by the first settle
		k.execute(ctx, &proposal)
	case proposal.Status == types.ProposalStatus_EXECUTION_FAILED:
		proposal.Status = types.ProposalStatus_EXPIRED
		k.setProposal(ctx, proposal)
	default:
		return nil, errorsmod.Wrapf(types.ErrProposalFinished, "proposal %d is %s", proposal.Id, proposal.Status)
	}
	if err != nil {
		return nil, err
	}

	if err := ctx.EventManager().EmitTypedEvent(&types.EventProposalSettled{
		Id:             proposal.Id,
		Status:         proposal.Status,
		Yes:            proposal.Yes.String(),
		No:             proposal.No.String(),
		Abstain:        proposal.Abstain.String(),
		Denominator:    proposal.PinnedDenominator.String(),
		ExpeditedEarly: expeditedEarly,
	}); err != nil {
		return nil, err
	}

	return &types.MsgSettleProposalResponse{Status: proposal.Status}, nil
}

// firstSettle takes a decided proposal out of voting, releases its deposit and
// runs it when it passed.
func (k msgServer) firstSettle(ctx sdk.Context, proposal *types.Proposal, passed bool) error {
	// Business Logic §8: a passed proposal refunds unless the policy is NEVER,
	// even when its execution fails (D13); a rejected one only under ALWAYS
	policy := proposal.PinnedConfig.DepositRefundPolicy
	refund := policy == types.DepositRefundPolicy_ALWAYS || (passed && policy == types.DepositRefundPolicy_ONLY_PASSED)
	if err := k.releaseDeposit(ctx, *proposal, refund); err != nil {
		return err
	}

	k.leaveVoting(ctx, *proposal)

	if !passed {
		proposal.Status = types.ProposalStatus_REJECTED
		k.setProposal(ctx, *proposal)

		return nil
	}

	k.execute(ctx, proposal)

	return nil
}

// execute runs a passed proposal's messages as the DAO and stores the outcome:
// EXECUTED, or EXECUTION_FAILED with every effect of the messages discarded.
// The status is stored as EXECUTED before the messages run, so a message that
// settles this proposal again finds it finished (D8). Only the messages run on
// a cached context: the status and the deposit release stay on ctx.
func (k msgServer) execute(ctx sdk.Context, proposal *types.Proposal) {
	proposal.Status = types.ProposalStatus_EXECUTED
	k.setProposal(ctx, *proposal)

	reason := k.runMsgs(ctx, *proposal)
	if reason != "" {
		proposal.Status = types.ProposalStatus_EXECUTION_FAILED
		k.setProposal(ctx, *proposal)
	}

	// a typed event of plain fields always marshals
	_ = ctx.EventManager().EmitTypedEvent(&types.EventProposalExecuted{
		Id:      proposal.Id,
		Success: reason == "",
		Error:   reason,
	})
}

// runMsgs runs the proposal's messages on a cached context and writes it only
// when every message succeeded. It returns the failure reason, or "" on success.
func (k msgServer) runMsgs(ctx sdk.Context, proposal types.Proposal) string {
	msgs, err := proposal.UnpackMsgs()
	if err != nil {
		return executionErrRouting
	}

	cachedCtx, write := ctx.CacheContext()
	if err := k.executeAsDao(cachedCtx, types.DaoAddress(proposal.DaoId), msgs); err != nil {
		return executionReason(err)
	}
	write()

	return ""
}

func executionReason(err error) string {
	switch {
	case errorsmod.IsOf(err, types.ErrUnauthorizedSigner):
		return executionErrSigner
	case errorsmod.IsOf(err, types.ErrExecutionPanic):
		return executionErrPanic
	case errorsmod.IsOf(err, types.ErrInvalidMsgs):
		return executionErrRouting
	default:
		return executionErrFailed
	}
}
