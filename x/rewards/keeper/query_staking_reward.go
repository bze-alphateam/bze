package keeper

import (
	"context"
	"github.com/cosmos/cosmos-sdk/types/query"

	"github.com/bze-alphateam/bze/x/rewards/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) StakingReward(goCtx context.Context, req *types.QueryGetStakingRewardRequest) (*types.QueryGetStakingRewardResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)

	val, found := k.GetStakingReward(
		ctx,
		req.RewardId,
	)
	if !found {
		return nil, status.Error(codes.NotFound, "not found")
	}

	return &types.QueryGetStakingRewardResponse{StakingReward: val}, nil
}

func (k Keeper) AllStakingRewards(goCtx context.Context, req *types.QueryAllStakingRewardsRequest) (*types.QueryAllStakingRewardsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	var stakingRewards []types.StakingReward
	ctx := sdk.UnwrapSDKContext(goCtx)

	stakingRewardStore := k.getPrefixedStore(ctx, types.KeyPrefix(types.StakingRewardKeyPrefix))

	pageRes, err := query.Paginate(stakingRewardStore, req.Pagination, func(key []byte, value []byte) error {
		var stakingReward types.StakingReward
		if err := k.cdc.Unmarshal(value, &stakingReward); err != nil {
			return err
		}

		stakingRewards = append(stakingRewards, stakingReward)
		return nil
	})

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryAllStakingRewardsResponse{List: stakingRewards, Pagination: pageRes}, nil
}

func (k Keeper) StakingRewardParticipant(goCtx context.Context, req *types.QueryStakingRewardParticipantRequest) (*types.QueryStakingRewardParticipantResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	var list []types.StakingRewardParticipant
	ctx := sdk.UnwrapSDKContext(goCtx)

	stakingRewardParticipantStore := k.getPrefixedStore(ctx, types.StakingRewardParticipantPrefix(req.Address))

	pageRes, err := query.Paginate(stakingRewardParticipantStore, req.Pagination, func(key []byte, value []byte) error {
		var stakingRewardParticipant types.StakingRewardParticipant
		if err := k.cdc.Unmarshal(value, &stakingRewardParticipant); err != nil {
			return err
		}

		list = append(list, stakingRewardParticipant)
		return nil
	})

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryStakingRewardParticipantResponse{List: list, Pagination: pageRes}, nil
}

func (k Keeper) AllStakingRewardParticipants(goCtx context.Context, req *types.QueryAllStakingRewardParticipantsRequest) (*types.QueryAllStakingRewardParticipantsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	var stakingRewardParticipants []types.StakingRewardParticipant
	ctx := sdk.UnwrapSDKContext(goCtx)

	stakingRewardParticipantStore := k.getPrefixedStore(ctx, types.KeyPrefix(types.StakingRewardParticipantKeyPrefix))

	pageRes, err := query.Paginate(stakingRewardParticipantStore, req.Pagination, func(key []byte, value []byte) error {
		var stakingRewardParticipant types.StakingRewardParticipant
		if err := k.cdc.Unmarshal(value, &stakingRewardParticipant); err != nil {
			return err
		}

		stakingRewardParticipants = append(stakingRewardParticipants, stakingRewardParticipant)
		return nil
	})

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryAllStakingRewardParticipantsResponse{List: stakingRewardParticipants, Pagination: pageRes}, nil
}

func (k Keeper) AllPendingUnlockParticipants(goCtx context.Context, req *types.QueryAllPendingUnlockParticipantsRequest) (*types.QueryAllPendingUnlockParticipantsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	store := k.getPrefixedStore(ctx, types.KeyPrefix(types.PendingUnlockParticipantKeyPrefix))

	var rewardParticipants []types.PendingUnlockParticipant
	pageRes, err := query.Paginate(store, req.Pagination, func(key []byte, value []byte) error {
		var part types.PendingUnlockParticipant
		if err := k.cdc.Unmarshal(value, &part); err != nil {
			return err
		}

		rewardParticipants = append(rewardParticipants, part)
		return nil
	})

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryAllPendingUnlockParticipantsResponse{List: rewardParticipants, Pagination: pageRes}, nil
}
