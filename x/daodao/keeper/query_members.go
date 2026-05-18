package keeper

import (
	"context"

	errorsmod "cosmossdk.io/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/bze-alphateam/bze/x/daodao/types"
)

// Members lists the (address, weight) entries of a STATIC DAO. Returns an
// error for non-STATIC DAOs (their membership lives in rewards, not here).
func (k Keeper) Members(ctx context.Context, req *types.QueryMembersRequest) (*types.QueryMembersResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}
	if req.DaoId == 0 {
		return nil, status.Error(codes.InvalidArgument, "dao_id must be non-zero")
	}
	dao, found := k.GetDao(ctx, req.DaoId)
	if !found {
		return nil, status.Errorf(codes.NotFound, "dao id=%d not found", req.DaoId)
	}
	if dao.VotingBackend != types.VotingBackendType_VOTING_BACKEND_STATIC {
		return nil, errorsmod.Wrapf(types.ErrNotStaticBackend,
			"DAO %d is %s; the Members query is STATIC-only", dao.Id, dao.VotingBackend)
	}

	members, pageRes, err := k.PaginatedStaticMembers(ctx, dao.Id, req.Pagination)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &types.QueryMembersResponse{Members: members, Pagination: pageRes}, nil
}
