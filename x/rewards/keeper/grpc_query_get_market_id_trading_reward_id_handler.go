package keeper

import (
	"context"

	"github.com/bze-alphateam/bze/x/rewards/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) GetMarketIdTradingRewardIdHandler(goCtx context.Context, req *types.QueryGetMarketIdTradingRewardIdHandlerRequest) (*types.QueryGetMarketIdTradingRewardIdHandlerResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	mr, found := k.GetMarketIdRewardId(ctx, req.MarketId)
	if !found {
		return nil, status.Error(codes.NotFound, "not found")
	}

	return &types.QueryGetMarketIdTradingRewardIdHandlerResponse{MarketIdRewardId: &mr}, nil
}
