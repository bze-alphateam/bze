package keeper

import (
	"context"
	"github.com/cosmos/cosmos-sdk/types/query"

	"github.com/bze-alphateam/bze/x/tradebin/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) LiquidityPool(goCtx context.Context, req *types.QueryLiquidityPoolRequest) (*types.QueryLiquidityPoolResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	lp, found := k.GetLiquidityPool(ctx, req.GetPoolId())
	if !found {
		return nil, status.Error(codes.NotFound, "not found")
	}

	return &types.QueryLiquidityPoolResponse{Pool: &lp}, nil
}

func (k Keeper) AllLiquidityPools(goCtx context.Context, req *types.QueryAllLiquidityPoolsRequest) (*types.QueryAllLiquidityPoolsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	lpStore := k.getLpStore(ctx)

	var lps []types.LiquidityPool
	pageRes, err := query.Paginate(lpStore, req.Pagination, func(key []byte, value []byte) error {
		var lp types.LiquidityPool
		if err := k.cdc.Unmarshal(value, &lp); err != nil {
			return err
		}

		lps = append(lps, lp)

		return nil
	})

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryAllLiquidityPoolsResponse{
		List:       lps,
		Pagination: pageRes,
	}, nil
}
