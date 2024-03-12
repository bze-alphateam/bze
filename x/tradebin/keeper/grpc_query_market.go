package keeper

import (
	"context"
	"fmt"

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

	if req.Base == "" || req.Quote == "" {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("invalid base [%s] and quote [%s] params provided", req.Base, req.Quote))
	}

	ctx := sdk.UnwrapSDKContext(c)

	val, found := k.GetMarket(
		ctx,
		req.Base,
		req.Quote,
	)
	if found {
		return &types.QueryGetMarketResponse{Market: val}, nil
	}

	//try finding the alias in case the user requested the market with assets in wrong order
	val, found = k.GetMarketAlias(
		ctx,
		req.Base,
		req.Quote,
	)
	if found {
		return &types.QueryGetMarketResponse{Market: val}, nil
	}

	return nil, status.Error(codes.NotFound, "not found")
}
