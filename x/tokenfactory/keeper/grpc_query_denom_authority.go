package keeper

import (
	"context"
	"strings"

	"github.com/bze-alphateam/bze/x/tokenfactory/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) DenomAuthority(goCtx context.Context, req *types.QueryDenomAuthorityRequest) (*types.QueryDenomAuthorityResponse, error) {
	if req == nil || req.Denom == "" {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	//we accept | in denom instead of / to ensure this can be used in REST endpoint.
	//the issue is that the denomination {DENOM} is a parameter in the URL and the slashes within the denom
	//makes the web server try to find a route that doesn't exist
	denom := strings.ReplaceAll(req.GetDenom(), "|", "/")

	ctx := sdk.UnwrapSDKContext(goCtx)
	dAuth, err := k.GetDenomAuthority(ctx, denom)
	if err != nil {
		return nil, err
	}

	return &types.QueryDenomAuthorityResponse{DenomAuthority: &dAuth}, nil
}
