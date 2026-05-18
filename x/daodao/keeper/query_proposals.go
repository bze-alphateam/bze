package keeper

import (
	"context"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/bze-alphateam/bze/x/daodao/types"
)

// Proposals returns paginated proposals for a DAO, optionally filtered by
// status. PROPOSAL_STATUS_UNSPECIFIED in the request means "no filter."
func (k Keeper) Proposals(goCtx context.Context, req *types.QueryProposalsRequest) (*types.QueryProposalsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Verify the DAO exists so an empty result on a typo is distinguishable
	// from an empty result on a real DAO.
	if _, ok := k.GetDao(ctx, req.DaoId); !ok {
		return nil, errorsmod.Wrapf(types.ErrDaoNotFound, "dao_id=%d", req.DaoId)
	}

	proposals, pageRes, err := k.PaginatedProposals(ctx, req.DaoId, req.StatusFilter, req.Pagination)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &types.QueryProposalsResponse{Proposals: proposals, Pagination: pageRes}, nil
}
