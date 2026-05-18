package keeper

import (
	"context"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/bze-alphateam/bze/x/daodao/types"
)

// DepositConfig returns a DAO's current deposit-period configuration.
// In-flight proposals carry their own frozen Proposal.deposit_snapshot;
// this reflects the value that will apply to FUTURE proposals.
func (k Keeper) DepositConfig(goCtx context.Context, req *types.QueryDepositConfigRequest) (*types.QueryDepositConfigResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)

	dao, ok := k.GetDao(ctx, req.DaoId)
	if !ok {
		return nil, errorsmod.Wrapf(types.ErrDaoNotFound, "dao_id=%d", req.DaoId)
	}
	return &types.QueryDepositConfigResponse{Deposit: dao.Deposit}, nil
}
