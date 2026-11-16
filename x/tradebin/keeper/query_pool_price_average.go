package keeper

import (
	"context"

	"github.com/bze-alphateam/bze/x/tradebin/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// PoolPriceAverage returns the moving average of a pool's price, brought up to the current block, and the
// height at which the stored record was last written.
func (k Keeper) PoolPriceAverage(goCtx context.Context, req *types.QueryPoolPriceAverageRequest) (*types.QueryPoolPriceAverageResponse, error) {
	if req == nil || req.GetPoolId() == "" {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	record, found := k.GetPoolPriceAverageRecord(ctx, req.GetPoolId())
	if !found {
		return nil, status.Error(codes.NotFound, "not found")
	}

	avg, _ := k.GetPoolPriceAverage(ctx, req.GetPoolId())

	return &types.QueryPoolPriceAverageResponse{AvgPrice: avg, LastHeight: record.LastHeight}, nil
}
