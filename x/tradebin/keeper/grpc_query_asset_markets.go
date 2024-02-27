package keeper

import (
	"context"

	"github.com/bze-alphateam/bze/x/tradebin/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) AssetMarkets(goCtx context.Context, req *types.QueryAssetMarketsRequest) (*types.QueryAssetMarketsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
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
