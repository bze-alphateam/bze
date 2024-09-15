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

	orders, pageRes, err := k.getMarketAggregatedOrdersPaginated(ctx, req.Market, req.OrderType, req.Pagination)

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryMarketAggregatedOrdersResponse{List: orders, Pagination: pageRes}, nil
}

func (k Keeper) getMarketAggregatedOrdersPaginated(
	ctx sdk.Context,
	market string,
	orderType string,
	pageReq *query.PageRequest,
) (orders []types.AggregatedOrder, response *query.PageResponse, err error) {

	aggOrderStore := k.getAggregatedOrderByMarketAndTypeStore(ctx, market, orderType)
	response, err = query.Paginate(aggOrderStore, pageReq, func(key []byte, value []byte) error {
		var order types.AggregatedOrder
		if err := k.cdc.Unmarshal(value, &order); err != nil {
			return err
		}

		orders = append(orders, order)
		return nil
	})

	if err != nil {
		return nil, nil, err
	}

	return orders, response, nil
}
