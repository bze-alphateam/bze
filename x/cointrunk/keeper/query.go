package keeper

import (
	"context"
	"github.com/bze-alphateam/bze/x/cointrunk/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var _ types.QueryServer = Keeper{}

func (k Keeper) AcceptedDomain(goCtx context.Context, req *types.QueryAcceptedDomainRequest) (*types.QueryAcceptedDomainResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	// TODO: Process the query
	_ = ctx

	return &types.QueryAcceptedDomainResponse{}, nil
}

func (k Keeper) AllAnonArticlesCounters(goCtx context.Context, req *types.QueryAllAnonArticlesCountersRequest) (*types.QueryAllAnonArticlesCountersResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	// TODO: Process the query
	_ = ctx

	return &types.QueryAllAnonArticlesCountersResponse{}, nil
}

func (k Keeper) AllArticles(goCtx context.Context, req *types.QueryAllArticlesRequest) (*types.QueryAllArticlesResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	// TODO: Process the query
	_ = ctx

	return &types.QueryAllArticlesResponse{}, nil
}

func (k Keeper) Publisher(goCtx context.Context, req *types.QueryPublisherRequest) (*types.QueryPublisherResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	// TODO: Process the query
	_ = ctx

	return &types.QueryPublisherResponse{}, nil
}

func (k Keeper) Publishers(goCtx context.Context, req *types.QueryPublishersRequest) (*types.QueryPublishersResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	// TODO: Process the query
	_ = ctx

	return &types.QueryPublishersResponse{}, nil
}
