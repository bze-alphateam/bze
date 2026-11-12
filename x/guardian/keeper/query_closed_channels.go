package keeper

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/bze-alphateam/bze/x/guardian/types"
)

// ClosedChannels lists the transfer channels whose ICS-20 transfers are stopped.
func (k Keeper) ClosedChannels(ctx context.Context, req *types.QueryClosedChannelsRequest) (*types.QueryClosedChannelsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	return &types.QueryClosedChannelsResponse{ChannelIds: k.GetAllClosedChannels(ctx)}, nil
}
