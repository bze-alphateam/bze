package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/bze-alphateam/bze/x/tradebin/types"
)

// HaltedDenoms lists the halted denoms, paginated, in store (byte) order.
func (k Keeper) HaltedDenoms(goCtx context.Context, req *types.QueryHaltedDenomsRequest) (*types.QueryHaltedDenomsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	store := k.getHaltedDenomStore(ctx)

	var denoms []string
	pageRes, err := query.Paginate(store, req.Pagination, func(key []byte, _ []byte) error {
		denoms = append(denoms, string(key))

		return nil
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryHaltedDenomsResponse{Denoms: denoms, Pagination: pageRes}, nil
}

// DenomHalted answers whether one denom is halted.
func (k Keeper) DenomHalted(goCtx context.Context, req *types.QueryDenomHaltedRequest) (*types.QueryDenomHaltedResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	if req.Denom == "" {
		return nil, status.Error(codes.InvalidArgument, "denom is required")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	return &types.QueryDenomHaltedResponse{Halted: k.IsDenomHalted(ctx, req.Denom)}, nil
}
