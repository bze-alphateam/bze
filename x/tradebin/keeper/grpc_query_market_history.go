package keeper

import (
	"context"
	"github.com/cosmos/cosmos-sdk/types/query"

	"github.com/bze-alphateam/bze/x/tradebin/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

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
