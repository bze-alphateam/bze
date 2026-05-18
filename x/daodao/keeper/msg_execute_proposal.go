package keeper

import (
	"context"
	"fmt"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/bze-alphateam/bze/x/daodao/types"
)

// ExecuteProposal implements MsgExecuteProposal.
//
// Order of operations:
//  1. ValidateBasic (signer + ids).
//  2. Load Proposal; reject if not PASSED.
//  3. Re-validate signers on msgs[] (defense-in-depth — the storage is
//     immutable but proto definitions could change in a chain upgrade).
//  4. Dispatch via dispatchProposalMsgs (atomic, cached context).
//     On failure: emit ExecutionFailed event, surface the inner error,
//     leave the proposal at PASSED so a retry is possible.
//     On success: commit, flip status to EXECUTED, emit Executed event.
func (k msgServer) ExecuteProposal(goCtx context.Context, msg *types.MsgExecuteProposal) (*types.MsgExecuteProposalResponse, error) {
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}
	ctx := sdk.UnwrapSDKContext(goCtx)

	p, ok := k.GetProposal(ctx, msg.DaoId, msg.ProposalId)
	if !ok {
		return nil, errorsmod.Wrapf(types.ErrProposalNotFound,
			"dao=%d proposal=%d", msg.DaoId, msg.ProposalId)
	}
	if p.Status != types.ProposalStatus_PROPOSAL_STATUS_PASSED {
		return nil, errorsmod.Wrapf(types.ErrProposalNotPassed,
			"proposal %d/%d is %s", p.DaoId, p.ProposalId, p.Status)
	}

	if err := k.validateProposalMsgSigners(p.DaoId, p.Msgs); err != nil {
		return nil, err
	}

	failedIdx, err := k.dispatchProposalMsgs(ctx, p.Msgs)
	if err != nil {
		// Status stays PASSED. Emit a structured failure event so indexers
		// / UIs can render the retry surface.
		failedType := ""
		if failedIdx >= 0 && failedIdx < len(p.Msgs) {
			failedType = p.Msgs[failedIdx].TypeUrl
		}
		ctx.EventManager().EmitEvent(sdk.NewEvent(
			types.EventTypeExecutionFailed,
			sdk.NewAttribute(types.AttributeKeyDaoID, fmt.Sprintf("%d", p.DaoId)),
			sdk.NewAttribute(types.AttributeKeyProposalID, fmt.Sprintf("%d", p.ProposalId)),
			sdk.NewAttribute(types.AttributeKeyExecutor, msg.Executor),
			sdk.NewAttribute(types.AttributeKeyMsgIndex, fmt.Sprintf("%d", failedIdx)),
			sdk.NewAttribute(types.AttributeKeyMsgTypeURL, failedType),
			sdk.NewAttribute(types.AttributeKeyFailureMsg, err.Error()),
		))
		return nil, err
	}

	// Atomic success — commit the status transition.
	p.Status = types.ProposalStatus_PROPOSAL_STATUS_EXECUTED
	if err := k.UpdateProposal(ctx, p); err != nil {
		return nil, fmt.Errorf("update proposal to EXECUTED: %w", err)
	}

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeExecuteProposal,
		sdk.NewAttribute(types.AttributeKeyDaoID, fmt.Sprintf("%d", p.DaoId)),
		sdk.NewAttribute(types.AttributeKeyProposalID, fmt.Sprintf("%d", p.ProposalId)),
		sdk.NewAttribute(types.AttributeKeyExecutor, msg.Executor),
	))

	return &types.MsgExecuteProposalResponse{}, nil
}
