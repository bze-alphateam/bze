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
