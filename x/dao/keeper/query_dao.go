package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/bze-alphateam/bze/x/dao/types"
)

func (k Keeper) Dao(goCtx context.Context, req *types.QueryDaoRequest) (*types.QueryDaoResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	dao, found := k.GetDao(goCtx, req.Id)
	if !found {
		return nil, status.Errorf(codes.NotFound, "dao %d not found", req.Id)
	}

	return &types.QueryDaoResponse{Dao: dao}, nil
}

func (k Keeper) DaoByAddress(goCtx context.Context, req *types.QueryDaoByAddressRequest) (*types.QueryDaoByAddressResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	addr, err := sdk.AccAddressFromBech32(req.Address)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid address: %s", err)
	}

	daoId, found := k.GetDaoIdByAddress(goCtx, addr)
	if !found {
		return nil, status.Errorf(codes.NotFound, "no dao owns %s", req.Address)
	}

	dao, found := k.GetDao(goCtx, daoId)
	if !found {
		return nil, status.Errorf(codes.Internal, "dao %d of %s not found", daoId, req.Address)
	}

	return &types.QueryDaoByAddressResponse{Dao: dao}, nil
}
