package keeper

import (
	"context"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/bze-alphateam/bze/x/dao/types"
)

func (k msgServer) CancelProposal(goCtx context.Context, msg *types.MsgCancelProposal) (*types.MsgCancelProposalResponse, error) {
	// the router runs it too; repeated so the server is safe when called directly
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	proposal, found := k.GetProposal(ctx, msg.ProposalId)
	if !found {
		return nil, errorsmod.Wrapf(types.ErrProposalNotFound, "proposal %d", msg.ProposalId)
	}

	if proposal.Status != types.ProposalStatus_VOTING {
		return nil, errorsmod.Wrapf(types.ErrVotingClosed, "proposal %d is %s", proposal.Id, proposal.Status)
	}

	// from voting_end on the outcome is decided: only the settle closes it
	if proposal.VotingEnded(ctx.BlockTime()) {
		return nil, errorsmod.Wrapf(types.ErrVotingClosed, "proposal %d voted until %d", proposal.Id, proposal.VotingEnd)
	}

	// compared in canonical form, as every handler stores addresses
	if sdk.MustAccAddressFromBech32(msg.Proposer).String() != proposal.Proposer {
		return nil, errorsmod.Wrapf(types.ErrNotProposalOwner, "%s did not propose proposal %d", msg.Proposer, proposal.Id)
	}

	// cancelling refunds only under ALWAYS; ONLY_PASSED and NEVER keep the deposit
	refund := proposal.PinnedConfig.DepositRefundPolicy == types.DepositRefundPolicy_ALWAYS
	if err := k.releaseDeposit(ctx, proposal, refund); err != nil {
		return nil, err
	}

	proposal.Status = types.ProposalStatus_CANCELLED
	k.leaveVoting(ctx, proposal)
	// the ballots stay until the cleanup of finished proposals sweeps them
	k.setProposal(ctx, proposal)

	if err := ctx.EventManager().EmitTypedEvent(&types.EventProposalCancelled{Id: proposal.Id}); err != nil {
		return nil, err
	}

	return &types.MsgCancelProposalResponse{}, nil
}
