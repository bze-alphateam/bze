package keeper

import (
	"context"
	"github.com/bze-alphateam/bze/x/tokenfactory/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
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

	ctx := sdk.UnwrapSDKContext(goCtx)
	dAuth, err := k.GetDenomAuthority(ctx, req.GetDenom())
	if err != nil {
		return nil, err
	}

	return &types.QueryDenomAuthorityResponse{DenomAuthority: &dAuth}, nil
}

func (k Keeper) DenomBranding(goCtx context.Context, req *types.QueryDenomBrandingRequest) (*types.QueryDenomBrandingResponse, error) {
	if req == nil || req.Denom == "" {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	branding, found := k.GetDenomBranding(ctx, req.GetDenom())
	if !found {
		return nil, status.Error(codes.NotFound, "not found")
	}

	return &types.QueryDenomBrandingResponse{Branding: &branding}, nil
}

func (k Keeper) AllDenomBranding(goCtx context.Context, req *types.QueryAllDenomBrandingRequest) (*types.QueryAllDenomBrandingResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	var records []types.DenomBrandingRecord

	brandingStore := k.GetBrandingsPrefixStore(ctx)
	pageRes, err := query.Paginate(brandingStore, req.Pagination, func(key []byte, value []byte) error {
		var branding types.DenomBranding
		if err := k.cdc.Unmarshal(value, &branding); err != nil {
			return err
		}

		records = append(records, types.DenomBrandingRecord{
			Denom:    string(key),
			Branding: &branding,
		})
		return nil
	})

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryAllDenomBrandingResponse{DenomBrandings: records, Pagination: pageRes}, nil
}
