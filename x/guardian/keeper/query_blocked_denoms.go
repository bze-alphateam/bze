package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/bze-alphateam/bze/x/guardian/types"
)

// BlockedDenoms passes through the rate limit module's blacklist, the module
// has no CLI command for it.
func (k Keeper) BlockedDenoms(goCtx context.Context, req *types.QueryBlockedDenomsRequest) (*types.QueryBlockedDenomsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	if k.rateLimitKeeper == nil {
		return nil, status.Error(codes.Unavailable, types.ErrRateLimitKeeperNotSet.Error())
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	return &types.QueryBlockedDenomsResponse{Denoms: k.rateLimitKeeper.GetAllBlacklistedDenoms(ctx)}, nil
}
