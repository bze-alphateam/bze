package keeper

import (
	"context"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/bze-alphateam/bze/x/dao/types"
)

func (k Keeper) Proposal(goCtx context.Context, req *types.QueryProposalRequest) (*types.QueryProposalResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	proposal, found := k.GetProposal(ctx, req.Id)
	if !found {
		return nil, status.Errorf(codes.NotFound, "proposal %d not found", req.Id)
	}

	// the same conditions SettleProposal checks, so a flag set here means the
	// settle it announces is accepted in this block
	now := ctx.BlockTime().Unix()
	voting := proposal.Status == types.ProposalStatus_VOTING
	res := &types.QueryProposalResponse{
		Proposal:    proposal,
		VotingEnded: voting && now >= proposal.VotingEnd,
		ExpediteSettleable: voting && proposal.Expedited && now < proposal.VotingEnd &&
			now >= k.earliestExpediteSettle(ctx, proposal) && proposal.ExpediteTallyPasses(),
	}

	if proposal.Status == types.ProposalStatus_EXECUTION_FAILED {
		res.RetryDeadline = k.executionDeadline(ctx, proposal)
	}

	return res, nil
}

// earliestExpediteSettle is the first block time (unix seconds) an expedited
// proposal can settle early: submit_time plus the current min_expedite_delay
// param.
func (k Keeper) earliestExpediteSettle(ctx context.Context, proposal types.Proposal) int64 {
	return proposal.SubmitTime + int64(k.GetParams(ctx).MinExpediteDelay/time.Second)
}

// executionDeadline is the end of the window in which a failed execution can
// be retried: voting_end plus the current execution_window param.
func (k Keeper) executionDeadline(ctx context.Context, proposal types.Proposal) int64 {
	window := k.GetParams(ctx).ExecutionWindow

	return time.Unix(proposal.VotingEnd, 0).Add(window).Unix()
}
