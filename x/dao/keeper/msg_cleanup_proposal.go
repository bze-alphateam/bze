package keeper

import (
	"context"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/bze-alphateam/bze/x/dao/types"
)

func (k msgServer) CleanupProposal(goCtx context.Context, msg *types.MsgCleanupProposal) (*types.MsgCleanupProposalResponse, error) {
	// the router runs it too; repeated so the server is safe when called directly
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	// a fully cleaned proposal is gone, so a repeated cleanup lands here too
	proposal, found := k.GetProposal(ctx, msg.ProposalId)
	if !found {
		return nil, errorsmod.Wrapf(types.ErrProposalNotFound, "proposal %d", msg.ProposalId)
	}

	if !proposal.Status.IsFinished() {
		return nil, errorsmod.Wrapf(types.ErrProposalNotFinished, "proposal %d is %s", proposal.Id, proposal.Status)
	}

	limit := k.GetParams(ctx).CleanupBatchSize
	if msg.Limit > 0 && msg.Limit < limit {
		limit = msg.Limit
	}

	// collected before deleting: the store must not change under an open iterator
	ballots, more, err := k.proposalBallots(ctx, proposal.Id, proposal.CleanupCursor, limit)
	if err != nil {
		return nil, errorsmod.Wrapf(err, "invalid cleanup cursor of proposal %d", proposal.Id)
	}

	for _, ballot := range ballots {
		k.removeBallot(ctx, ballot)
	}

	if more {
		// the cursor only moves forward: the next batch starts after the last
		// deleted ballot and never revisits the deleted range
		proposal.CleanupCursor = ballots[len(ballots)-1].Voter
		k.setProposal(ctx, proposal)
	} else {
		k.removeProposal(ctx, proposal)
	}

	processed := uint64(len(ballots))
	if err := ctx.EventManager().EmitTypedEvent(&types.EventProposalCleaned{
		Id:        proposal.Id,
		Processed: processed,
		Completed: !more,
	}); err != nil {
		return nil, err
	}

	return &types.MsgCleanupProposalResponse{Processed: processed, Completed: !more}, nil
}
