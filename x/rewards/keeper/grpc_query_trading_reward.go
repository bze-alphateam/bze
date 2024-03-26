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

func (k msgServer) TradingRewardAll(c context.Context, req *types.QueryAllTradingRewardRequest) (*types.QueryAllTradingRewardResponse, error) {
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

func (k msgServer) TradingReward(c context.Context, req *types.QueryGetTradingRewardRequest) (*types.QueryGetTradingRewardResponse, error) {
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

func (k msgServer) getTradingRewardStore(ctx sdk.Context, state string) prefix.Store {
	s := k.getActiveTradingRewardStore(ctx)
	if state == "pending" {
		s = k.getPendingTradingRewardStore(ctx)
	}

	return s
}
