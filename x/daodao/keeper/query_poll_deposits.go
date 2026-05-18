package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/bze-alphateam/bze/x/daodao/types"
)

// PollDeposits returns paginated per-(poll, depositor) deposit records.
func (k Keeper) PollDeposits(goCtx context.Context, req *types.QueryPollDepositsRequest) (*types.QueryPollDepositsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)

	deposits, pageRes, err := k.PaginatedPollDepositRecords(ctx, req.DaoId, req.PollId, req.Pagination)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &types.QueryPollDepositsResponse{Deposits: deposits, Pagination: pageRes}, nil
}
