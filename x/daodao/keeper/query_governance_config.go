package keeper

import (
	"context"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/bze-alphateam/bze/x/daodao/types"
)

// GovernanceConfig returns a DAO's current proposal-track configuration.
// In-flight proposals carry their own frozen governance_snapshot — this
// endpoint reflects the value that will apply to FUTURE proposals.
func (k Keeper) GovernanceConfig(goCtx context.Context, req *types.QueryGovernanceConfigRequest) (*types.QueryGovernanceConfigResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)

	g, ok := k.GetGovernanceConfig(ctx, req.DaoId)
	if !ok {
		return nil, errorsmod.Wrapf(types.ErrDaoNotFound, "dao_id=%d", req.DaoId)
	}
	return &types.QueryGovernanceConfigResponse{Governance: g}, nil
}
