package keeper

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/bze-alphateam/bze/x/daodao/types"
)

// SubDaos returns the children of a parent DAO.
func (k Keeper) SubDaos(ctx context.Context, req *types.QuerySubDaosRequest) (*types.QueryDaosResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}
	if req.ParentDaoId == 0 {
		return nil, status.Error(codes.InvalidArgument, "parent_dao_id must be non-zero")
	}
	daos, pageRes, err := k.PaginatedSubDaos(ctx, req.ParentDaoId, req.Pagination)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &types.QueryDaosResponse{Daos: daos, Pagination: pageRes}, nil
}
