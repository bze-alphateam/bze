package keeper

import (
	"context"
	"github.com/bze-alphateam/bze/x/epochs/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var _ types.QueryServer = &Keeper{}

// EpochInfos provide running epochInfos.
func (k Keeper) EpochInfos(c context.Context, _ *types.QueryEpochsInfoRequest) (*types.QueryEpochsInfoResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)

	return &types.QueryEpochsInfoResponse{
		Epochs: k.AllEpochInfos(ctx),
	}, nil
}

// CurrentEpoch provides current epoch of specified identifier.
func (k Keeper) CurrentEpoch(c context.Context, req *types.QueryCurrentEpochRequest) (*types.QueryCurrentEpochResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}
	if req.Identifier == "" {
		return nil, status.Error(codes.InvalidArgument, "identifier is empty")
	}

	ctx := sdk.UnwrapSDKContext(c)

	info := k.GetEpochInfo(ctx, req.Identifier)
	if info.Identifier != req.Identifier {
		return nil, status.Error(codes.NotFound, "identifier not found")
	}

	return &types.QueryCurrentEpochResponse{
		CurrentEpoch: info.CurrentEpoch,
	}, nil
}
