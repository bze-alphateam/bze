package keeper

import (
	"context"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/bze-alphateam/bze/x/daodao/types"
)

// Polls returns paginated polls for a DAO, optionally status-filtered.
// POLL_STATUS_UNSPECIFIED → no filter (full per-DAO range).
func (k Keeper) Polls(goCtx context.Context, req *types.QueryPollsRequest) (*types.QueryPollsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)

	if _, ok := k.GetDao(ctx, req.DaoId); !ok {
		return nil, errorsmod.Wrapf(types.ErrDaoNotFound, "dao_id=%d", req.DaoId)
	}

	polls, pageRes, err := k.PaginatedPolls(ctx, req.DaoId, req.StatusFilter, req.Pagination)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &types.QueryPollsResponse{Polls: polls, Pagination: pageRes}, nil
}
