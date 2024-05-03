package keeper

import (
	"context"

	"github.com/bze-alphateam/bze/x/tradebin/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) MarketOrder(goCtx context.Context, req *types.QueryMarketOrderRequest) (*types.QueryMarketOrderResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	if req.Market == "" || req.OrderId == "" || (req.OrderType != types.OrderTypeSell && req.OrderType != types.OrderTypeBuy) {
		return nil, status.Error(codes.InvalidArgument, "please provide a valid market, order_id and order_type")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)

	order, found := k.GetOrder(ctx, req.Market, req.OrderType, req.OrderId)
	if !found {
		return nil, status.Error(codes.NotFound, "not found")
	}

	return &types.QueryMarketOrderResponse{Order: order}, nil
}
