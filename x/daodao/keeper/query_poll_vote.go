package keeper

import (
	"context"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/bze-alphateam/bze/x/daodao/types"
)

// PollVote returns a single voter's selection on a poll. NOT_FOUND if
// the voter hasn't voted.
func (k Keeper) PollVote(goCtx context.Context, req *types.QueryPollVoteRequest) (*types.QueryPollVoteResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)

	voterAddr, err := sdk.AccAddressFromBech32(req.Voter)
	if err != nil {
		return nil, errorsmod.Wrap(types.ErrInvalidAddress, err.Error())
	}
	v, ok := k.GetPollVote(ctx, req.DaoId, req.PollId, voterAddr)
	if !ok {
		return nil, status.Errorf(codes.NotFound,
			"no poll vote from %s on dao=%d poll=%d", req.Voter, req.DaoId, req.PollId)
	}
	return &types.QueryPollVoteResponse{Vote: v}, nil
}
