package keeper

import (
	"context"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/bze-alphateam/bze/x/daodao/types"
)

// PollTally returns the running tally + status + winning_choice_index
// for a poll. UI polls this endpoint instead of reading the full Poll.
func (k Keeper) PollTally(goCtx context.Context, req *types.QueryPollTallyRequest) (*types.QueryPollTallyResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)

	p, ok := k.GetPoll(ctx, req.DaoId, req.PollId)
	if !ok {
		return nil, errorsmod.Wrapf(types.ErrPollNotFound,
			"dao_id=%d poll_id=%d", req.DaoId, req.PollId)
	}
	return &types.QueryPollTallyResponse{
		Tally:              p.Tally,
		Status:             p.Status,
		WinningChoiceIndex: p.WinningChoiceIndex,
	}, nil
}
