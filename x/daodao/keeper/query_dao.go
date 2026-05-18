package keeper

import (
	"context"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/bze-alphateam/bze/x/daodao/types"
)

// Dao returns a single DAO by id.
func (k Keeper) Dao(ctx context.Context, req *types.QueryDaoRequest) (*types.QueryDaoResponse, error) {
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
	return &types.QueryDaoResponse{Dao: dao}, nil
}

// DaoByAddress returns a single DAO by its on-chain account address.
func (k Keeper) DaoByAddress(ctx context.Context, req *types.QueryDaoByAddressRequest) (*types.QueryDaoResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}
	addr, err := sdk.AccAddressFromBech32(req.Address)
	if err != nil {
		return nil, errorsmod.Wrap(types.ErrInvalidAddress, err.Error())
	}
	dao, found := k.GetDaoByAddress(ctx, addr)
	if !found {
		return nil, status.Errorf(codes.NotFound, "no DAO registered at address %s", req.Address)
	}
	return &types.QueryDaoResponse{Dao: dao}, nil
}
