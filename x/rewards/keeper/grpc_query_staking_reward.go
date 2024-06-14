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

func (k Keeper) StakingRewardAll(c context.Context, req *types.QueryAllStakingRewardRequest) (*types.QueryAllStakingRewardResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	var stakingRewards []types.StakingReward
	ctx := sdk.UnwrapSDKContext(c)

	store := ctx.KVStore(k.storeKey)
	stakingRewardStore := prefix.NewStore(store, types.KeyPrefix(types.StakingRewardKeyPrefix))

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

	return &types.QueryAllStakingRewardResponse{List: stakingRewards, Pagination: pageRes}, nil
}

func (k Keeper) StakingReward(c context.Context, req *types.QueryGetStakingRewardRequest) (*types.QueryGetStakingRewardResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(c)

	val, found := k.GetStakingReward(
		ctx,
		req.RewardId,
	)
	if !found {
		return nil, status.Error(codes.NotFound, "not found")
	}

	return &types.QueryGetStakingRewardResponse{StakingReward: val}, nil
}

func (k Keeper) StakingRewardParticipantAll(c context.Context, req *types.QueryAllStakingRewardParticipantRequest) (*types.QueryAllStakingRewardParticipantResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	var stakingRewardParticipants []types.StakingRewardParticipant
	ctx := sdk.UnwrapSDKContext(c)

	store := ctx.KVStore(k.storeKey)
	stakingRewardParticipantStore := prefix.NewStore(store, types.KeyPrefix(types.StakingRewardParticipantKeyPrefix))

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

	return &types.QueryAllStakingRewardParticipantResponse{List: stakingRewardParticipants, Pagination: pageRes}, nil
}

func (k Keeper) StakingRewardParticipant(c context.Context, req *types.QueryGetStakingRewardParticipantRequest) (*types.QueryGetStakingRewardParticipantResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	var list []types.StakingRewardParticipant
	ctx := sdk.UnwrapSDKContext(c)

	store := ctx.KVStore(k.storeKey)
	stakingRewardParticipantStore := prefix.NewStore(store, types.StakingRewardParticipantPrefix(req.Address))

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

	return &types.QueryGetStakingRewardParticipantResponse{List: list, Pagination: pageRes}, nil
}

func (k Keeper) AllPendingUnlockParticipant(goCtx context.Context, req *types.QueryAllPendingUnlockParticipantRequest) (*types.QueryAllPendingUnlockParticipantResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.PendingUnlockParticipantKeyPrefix))

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

	return &types.QueryAllPendingUnlockParticipantResponse{List: rewardParticipants, Pagination: pageRes}, nil
}
