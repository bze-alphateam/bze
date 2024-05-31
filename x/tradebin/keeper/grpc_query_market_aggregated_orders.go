package keeper

import (
	"context"
	"github.com/cosmos/cosmos-sdk/types/query"

	"github.com/bze-alphateam/bze/x/tradebin/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) MarketAggregatedOrders(goCtx context.Context, req *types.QueryMarketAggregatedOrdersRequest) (*types.QueryMarketAggregatedOrdersResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	_, found := k.GetMarketById(ctx, req.Market)
	if !found {
		return nil, status.Error(codes.InvalidArgument, "invalid market")
	}

	if req.OrderType != types.OrderTypeBuy && req.OrderType != types.OrderTypeSell {
		return nil, status.Error(codes.InvalidArgument, "invalid order type")
	}

	var orders []types.AggregatedOrder
	aggOrderStore := k.getAggregatedOrderByMarketAndTypeStore(ctx, req.Market, req.OrderType)

	pageRes, err := query.Paginate(aggOrderStore, req.Pagination, func(key []byte, value []byte) error {
		var order types.AggregatedOrder
		if err := k.cdc.Unmarshal(value, &order); err != nil {
			return err
		}

		orders = append(orders, order)
		return nil
	})

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryMarketAggregatedOrdersResponse{List: orders, Pagination: pageRes}, nil
}
