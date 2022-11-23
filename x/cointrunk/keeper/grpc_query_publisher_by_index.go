package keeper

import (
	"context"

	"github.com/bze-alphateam/bze/x/cointrunk/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) PublisherByIndex(goCtx context.Context, req *types.QueryPublisherByIndexRequest) (*types.QueryPublisherByIndexResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	publisher, found := k.GetPublisher(ctx, req.Index)
	if !found {
		return nil, status.Error(codes.InvalidArgument, "not found")
	}

	_ = ctx

	return &types.QueryPublisherByIndexResponse{Publisher: publisher}, nil
}
