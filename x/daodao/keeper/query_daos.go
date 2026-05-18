package keeper

import (
	"context"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/bze-alphateam/bze/x/daodao/types"
)

// Daos returns a paginated list of all DAOs.
func (k Keeper) Daos(ctx context.Context, req *types.QueryDaosRequest) (*types.QueryDaosResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}
	daos, pageRes, err := k.PaginatedDaos(ctx, req.Pagination)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &types.QueryDaosResponse{Daos: daos, Pagination: pageRes}, nil
}

// DaosByCreator returns DAOs created by a given address.
func (k Keeper) DaosByCreator(ctx context.Context, req *types.QueryDaosByCreatorRequest) (*types.QueryDaosResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}
	creator, err := sdk.AccAddressFromBech32(req.Creator)
	if err != nil {
		return nil, errorsmod.Wrap(types.ErrInvalidAddress, err.Error())
	}
	daos, pageRes, err := k.PaginatedDaosByCreator(ctx, creator, req.Pagination)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &types.QueryDaosResponse{Daos: daos, Pagination: pageRes}, nil
}
