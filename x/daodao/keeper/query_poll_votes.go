package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/bze-alphateam/bze/x/daodao/types"
)

// PollVotes returns paginated votes on a single poll.
func (k Keeper) PollVotes(goCtx context.Context, req *types.QueryPollVotesRequest) (*types.QueryPollVotesResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)

	votes, pageRes, err := k.PaginatedPollVotes(ctx, req.DaoId, req.PollId, req.Pagination)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &types.QueryPollVotesResponse{Votes: votes, Pagination: pageRes}, nil
}
