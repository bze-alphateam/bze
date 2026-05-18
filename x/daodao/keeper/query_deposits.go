package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/bze-alphateam/bze/x/daodao/types"
)

// Deposits returns paginated DepositRecords for a (dao, proposal) pair.
func (k Keeper) Deposits(goCtx context.Context, req *types.QueryDepositsRequest) (*types.QueryDepositsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)

	deposits, pageRes, err := k.PaginatedDepositRecords(ctx, req.DaoId, req.ProposalId, req.Pagination)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &types.QueryDepositsResponse{Deposits: deposits, Pagination: pageRes}, nil
}
