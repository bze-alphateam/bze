package keeper

import (
	"context"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/bze-alphateam/bze/x/daodao/types"
)

// Proposal returns a single proposal by (dao_id, proposal_id).
func (k Keeper) Proposal(goCtx context.Context, req *types.QueryProposalRequest) (*types.QueryProposalResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)

	p, ok := k.GetProposal(ctx, req.DaoId, req.ProposalId)
	if !ok {
		return nil, errorsmod.Wrapf(types.ErrProposalNotFound,
			"dao_id=%d proposal_id=%d", req.DaoId, req.ProposalId)
	}
	return &types.QueryProposalResponse{Proposal: p}, nil
}
