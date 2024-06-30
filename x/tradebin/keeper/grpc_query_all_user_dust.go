package keeper

import (
	"context"

	"github.com/bze-alphateam/bze/x/tradebin/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) AllUserDust(goCtx context.Context, req *types.QueryAllUserDustRequest) (*types.QueryAllUserDustResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	list := k.GetUserDustByOwner(ctx, req.Address)

	return &types.QueryAllUserDustResponse{List: list}, nil
}
