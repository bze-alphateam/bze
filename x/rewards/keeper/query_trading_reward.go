package keeper

import (
	"context"
	"cosmossdk.io/store/prefix"
	"github.com/cosmos/cosmos-sdk/types/query"

	"github.com/bze-alphateam/bze/x/rewards/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) TradingReward(goCtx context.Context, req *types.QueryTradingRewardRequest) (*types.QueryTradingRewardResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	val, found := k.GetPendingTradingReward(ctx, req.RewardId)
	if found {
		return &types.QueryTradingRewardResponse{TradingReward: val}, nil
	}

	val, found = k.GetActiveTradingReward(ctx, req.RewardId)
	if found {
		return &types.QueryTradingRewardResponse{TradingReward: val}, nil
	}

	return nil, status.Error(codes.NotFound, "not found")
}

func (k Keeper) AllTradingRewards(goCtx context.Context, req *types.QueryAllTradingRewardsRequest) (*types.QueryAllTradingRewardsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	var tradingRewards []types.TradingReward

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

	return &types.QueryAllTradingRewardsResponse{List: tradingRewards, Pagination: pageRes}, nil
}

func (k Keeper) TradingRewardLeaderboard(goCtx context.Context, req *types.QueryTradingRewardLeaderboardRequest) (*types.QueryTradingRewardLeaderboardResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	trl, found := k.GetTradingRewardLeaderboard(ctx, req.RewardId)
	if !found {
		return nil, status.Error(codes.NotFound, "not found")
	}

	return &types.QueryTradingRewardLeaderboardResponse{Leaderboard: &trl}, nil
}

func (k Keeper) MarketTradingReward(goCtx context.Context, req *types.QueryMarketTradingRewardRequest) (*types.QueryMarketTradingRewardResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	mr, found := k.GetMarketIdRewardId(ctx, req.MarketId)
	if !found {
		return nil, status.Error(codes.NotFound, "not found")
	}

	return &types.QueryMarketTradingRewardResponse{MarketReward: &mr}, nil
}

func (k Keeper) getTradingRewardStore(ctx sdk.Context, state string) prefix.Store {
	s := k.getActiveTradingRewardStore(ctx)
	if state == "pending" {
		s = k.getPendingTradingRewardStore(ctx)
	}

	return s
}
