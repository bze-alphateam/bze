package keeper

import (
	"context"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/bze-alphateam/bze/x/daodao/types"
)

// Poll returns a single poll by (dao_id, poll_id).
func (k Keeper) Poll(goCtx context.Context, req *types.QueryPollRequest) (*types.QueryPollResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)

	p, ok := k.GetPoll(ctx, req.DaoId, req.PollId)
	if !ok {
		return nil, errorsmod.Wrapf(types.ErrPollNotFound,
			"dao_id=%d poll_id=%d", req.DaoId, req.PollId)
	}
	return &types.QueryPollResponse{Poll: p}, nil
}
