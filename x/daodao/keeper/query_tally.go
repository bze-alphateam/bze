package keeper

import (
	"context"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/bze-alphateam/bze/x/daodao/types"
)

// Tally returns the running tally for a proposal plus its status. A
// dedicated endpoint (over reading the Proposal) keeps UI polling
// payloads minimal.
func (k Keeper) Tally(goCtx context.Context, req *types.QueryTallyRequest) (*types.QueryTallyResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)

	p, ok := k.GetProposal(ctx, req.DaoId, req.ProposalId)
	if !ok {
		return nil, errorsmod.Wrapf(types.ErrProposalNotFound,
			"dao_id=%d proposal_id=%d", req.DaoId, req.ProposalId)
	}
	return &types.QueryTallyResponse{Tally: p.Tally, Status: p.Status}, nil
}
