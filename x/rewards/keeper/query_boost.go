package keeper

import (
	"context"

	"github.com/bze-alphateam/bze/x/rewards/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Boost returns a single boost identified by (reward_id, denom).
func (k Keeper) Boost(goCtx context.Context, req *types.QueryBoostRequest) (*types.QueryBoostResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)

	val, found := k.GetBoost(ctx, req.RewardId, req.Denom)
	if !found {
		return nil, status.Error(codes.NotFound, "not found")
	}

	return &types.QueryBoostResponse{Boost: val}, nil
}

// RewardBoosts returns every boost of a single reward via a bounded prefix scan.
func (k Keeper) RewardBoosts(goCtx context.Context, req *types.QueryRewardBoostsRequest) (*types.QueryRewardBoostsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)

	list := k.GetRewardBoosts(ctx, req.RewardId)

	return &types.QueryRewardBoostsResponse{List: list}, nil
}

// AllBoosts returns a paginated list of every boost in the store.
func (k Keeper) AllBoosts(goCtx context.Context, req *types.QueryAllBoostsRequest) (*types.QueryAllBoostsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	var boosts []types.Boost
	ctx := sdk.UnwrapSDKContext(goCtx)

	boostStore := k.getPrefixedStore(ctx, types.KeyPrefix(types.BoostKeyPrefix))

	pageRes, err := query.Paginate(boostStore, req.Pagination, func(key []byte, value []byte) error {
		var boost types.Boost
		if err := k.cdc.Unmarshal(value, &boost); err != nil {
			return err
		}

		boosts = append(boosts, boost)
		return nil
	})

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryAllBoostsResponse{List: boosts, Pagination: pageRes}, nil
}
