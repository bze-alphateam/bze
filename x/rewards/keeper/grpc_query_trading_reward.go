package keeper

import (
	"context"

	"github.com/bze-alphateam/bze/x/rewards/types"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) TradingRewardAll(c context.Context, req *types.QueryAllTradingRewardRequest) (*types.QueryAllTradingRewardResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	var tradingRewards []types.TradingReward
	ctx := sdk.UnwrapSDKContext(c)

	tradingRewardStore := k.getTradingRewardStore(ctx, req.State)
	pageRes, err := query.Paginate(tradingRewardStore, req.Pagination, func(key []byte, value []byte) error {
		var tradingReward types.TradingReward
		if err := k.cdc.Unmarshal(value, &tradingReward); err != nil {
			return err
		}

		tradingRewards = append(tradingRewards, tradingReward)
		return nil
	})

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryAllTradingRewardResponse{List: tradingRewards, Pagination: pageRes}, nil
}

func (k Keeper) TradingReward(c context.Context, req *types.QueryGetTradingRewardRequest) (*types.QueryGetTradingRewardResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(c)

	val, found := k.GetPendingTradingReward(ctx, req.RewardId)
	if found {
		return &types.QueryGetTradingRewardResponse{TradingReward: val}, nil
	}

	val, found = k.GetActiveTradingReward(ctx, req.RewardId)
	if found {
		return &types.QueryGetTradingRewardResponse{TradingReward: val}, nil
	}

	return nil, status.Error(codes.NotFound, "not found")
}

func (k Keeper) getTradingRewardStore(ctx sdk.Context, state string) prefix.Store {
	s := k.getActiveTradingRewardStore(ctx)
	if state == "pending" {
		s = k.getPendingTradingRewardStore(ctx)
	}

	return s
}

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

func (k Keeper) GetTradingRewardLeaderboardHandler(goCtx context.Context, req *types.QueryGetTradingRewardLeaderboardRequest) (*types.QueryGetTradingRewardLeaderboardResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	trl, found := k.GetTradingRewardLeaderboard(ctx, req.RewardId)
	if !found {
		return nil, status.Error(codes.NotFound, "not found")
	}

	return &types.QueryGetTradingRewardLeaderboardResponse{Leaderboard: &trl}, nil
}
