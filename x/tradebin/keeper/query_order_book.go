package keeper

import (
	"context"
	"cosmossdk.io/store/prefix"
	"fmt"
	"github.com/bze-alphateam/bze/x/tradebin/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

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

func (k Keeper) AllMarkets(goCtx context.Context, req *types.QueryAllMarketsRequest) (*types.QueryAllMarketsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	var markets []types.Market
	ctx := sdk.UnwrapSDKContext(goCtx)
	marketStore := k.getMarketStore(ctx)

	pageRes, err := query.Paginate(marketStore, req.Pagination, func(key []byte, value []byte) error {
		var market types.Market
		if err := k.cdc.Unmarshal(value, &market); err != nil {
			return err
		}

		markets = append(markets, market)
		return nil
	})

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryAllMarketsResponse{Market: markets, Pagination: pageRes}, nil
}

func (k Keeper) AllUserDust(goCtx context.Context, req *types.QueryAllUserDustRequest) (*types.QueryAllUserDustResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	_, err := sdk.AccAddressFromBech32(req.Address)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid address")
	}

	list := k.GetUserDustByOwner(ctx, req.Address)

	return &types.QueryAllUserDustResponse{List: list}, nil
}

func (k Keeper) AssetMarkets(goCtx context.Context, req *types.QueryAssetMarketsRequest) (*types.QueryAssetMarketsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	if req.Asset == "" {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("invalid asset [%s] provided", req.Asset))
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	base := k.GetAllAssetMarkets(ctx, req.Asset)
	quote := k.GetAllAssetMarketAliases(ctx, req.Asset)
	_ = ctx

	return &types.QueryAssetMarketsResponse{
		Base:  base,
		Quote: quote,
	}, nil
}

func (k Keeper) Market(goCtx context.Context, req *types.QueryMarketRequest) (*types.QueryMarketResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	if req.Base == "" || req.Quote == "" {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("invalid base [%s] and quote [%s] params provided", req.Base, req.Quote))
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	val, found := k.GetMarket(
		ctx,
		req.Base,
		req.Quote,
	)
	if found {
		return &types.QueryMarketResponse{Market: val}, nil
	}

	//try finding the alias in case the user requested the market with assets in wrong order
	val, found = k.GetMarketAlias(
		ctx,
		req.Base,
		req.Quote,
	)
	if found {
		return &types.QueryMarketResponse{Market: val}, nil
	}

	return nil, status.Error(codes.NotFound, "not found")
}

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

func (k Keeper) MarketHistory(goCtx context.Context, req *types.QueryMarketHistoryRequest) (*types.QueryMarketHistoryResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	_, found := k.GetMarketById(ctx, req.Market)
	if !found {
		return nil, status.Error(codes.InvalidArgument, "invalid market")
	}

	var orders []types.HistoryOrder
	historyOrderStore := k.getHistoryOrderByMarketStore(ctx, req.Market)

	pageRes, err := query.Paginate(historyOrderStore, req.Pagination, func(key []byte, value []byte) error {
		var order types.HistoryOrder
		if err := k.cdc.Unmarshal(value, &order); err != nil {
			return err
		}

		orders = append(orders, order)
		return nil
	})

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryMarketHistoryResponse{List: orders, Pagination: pageRes}, nil
}

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

func (k Keeper) UserMarketOrders(goCtx context.Context, req *types.QueryUserMarketOrdersRequest) (*types.QueryUserMarketOrdersResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	_, err := sdk.AccAddressFromBech32(req.Address)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid address")
	}

	var userOrderStore prefix.Store
	if req.Market != "" {
		_, found := k.GetMarketById(ctx, req.Market)
		if !found {
			return nil, status.Error(codes.InvalidArgument, "invalid market")
		}

		userOrderStore = k.getUserOrderByAddressAndMarketStore(ctx, req.Address, req.Market)
	} else {
		userOrderStore = k.getUserOrderByAddressStore(ctx, req.Address)
	}

	var orders []types.OrderReference
	pageRes, err := query.Paginate(userOrderStore, req.Pagination, func(key []byte, value []byte) error {
		var order types.OrderReference
		if err := k.cdc.Unmarshal(value, &order); err != nil {
			return err
		}

		orders = append(orders, order)
		return nil
	})

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryUserMarketOrdersResponse{List: orders, Pagination: pageRes}, nil
}
