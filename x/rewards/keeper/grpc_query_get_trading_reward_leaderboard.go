package keeper

import (
	"context"

	"github.com/bze-alphateam/bze/x/rewards/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

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
