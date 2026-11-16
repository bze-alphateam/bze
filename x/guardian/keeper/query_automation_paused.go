package keeper

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/bze-alphateam/bze/x/guardian/types"
)

// AutomationPaused tells whether the protocol automation is paused.
func (k Keeper) AutomationPaused(ctx context.Context, req *types.QueryAutomationPausedRequest) (*types.QueryAutomationPausedResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	return &types.QueryAutomationPausedResponse{Paused: k.IsAutomationPaused(ctx)}, nil
}
