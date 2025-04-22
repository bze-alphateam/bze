package keeper

import (
	"context"
	"github.com/bze-alphateam/bze/x/epochs/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var _ types.QueryServer = Keeper{}

func (k Keeper) EpochInfos(ctx context.Context, _ *types.QueryEpochsInfoRequest) (*types.QueryEpochsInfoResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	return &types.QueryEpochsInfoResponse{
		Epochs: k.AllEpochInfos(sdkCtx),
	}, nil
}

func (k Keeper) CurrentEpoch(ctx context.Context, req *types.QueryCurrentEpochRequest) (*types.QueryCurrentEpochResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}
	if req.Identifier == "" {
		return nil, status.Error(codes.InvalidArgument, "identifier is empty")
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)

	info := k.GetEpochInfo(sdkCtx, req.Identifier)
	if info.Identifier != req.Identifier {
		return nil, status.Error(codes.NotFound, "identifier not found")
	}

	return &types.QueryCurrentEpochResponse{
		CurrentEpoch: info.CurrentEpoch,
	}, nil
}
