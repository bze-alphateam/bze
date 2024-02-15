package keeper

import (
	"context"

	"github.com/bze-alphateam/bze/x/tradebin/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) MarketAll(c context.Context, req *types.QueryAllMarketRequest) (*types.QueryAllMarketResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	var markets []types.Market
	ctx := sdk.UnwrapSDKContext(c)
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

	return &types.QueryAllMarketResponse{Market: markets, Pagination: pageRes}, nil
}

func (k Keeper) Market(c context.Context, req *types.QueryGetMarketRequest) (*types.QueryGetMarketResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(c)

	val, found := k.GetMarket(
		ctx,
		req.Asset1,
		req.Asset2,
	)
	if !found {
		return nil, status.Error(codes.NotFound, "not found")
	}

	return &types.QueryGetMarketResponse{Market: val}, nil
}
