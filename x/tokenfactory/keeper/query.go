package keeper

import (
	"context"
	"github.com/bze-alphateam/bze/x/tokenfactory/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var _ types.QueryServer = Keeper{}

func (k Keeper) Params(goCtx context.Context, req *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)

	return &types.QueryParamsResponse{Params: k.GetParams(ctx)}, nil
}

func (k Keeper) DenomAuthority(goCtx context.Context, req *types.QueryDenomAuthorityRequest) (*types.QueryDenomAuthorityResponse, error) {
	if req == nil || req.Denom == "" {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	denom := req.GetDenom()
	ctx := sdk.UnwrapSDKContext(goCtx)
	dAuth, err := k.GetDenomAuthority(ctx, denom)
	if err != nil {
		return nil, err
	}

	return &types.QueryDenomAuthorityResponse{DenomAuthority: &dAuth}, nil
}
