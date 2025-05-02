package keeper

import (
	"context"
	"encoding/binary"

	"github.com/bze-alphateam/bze/x/cointrunk/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var _ types.QueryServer = Keeper{}

func (k Keeper) AcceptedDomain(goCtx context.Context, req *types.QueryAcceptedDomainRequest) (*types.QueryAcceptedDomainResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	var acceptedDomains []types.AcceptedDomain
	adStore := k.getPrefixedStore(ctx, types.KeyPrefix(types.AcceptedDomainKeyPrefix))

	pageRes, err := query.Paginate(adStore, req.Pagination, func(key []byte, value []byte) error {
		var acceptedDomain types.AcceptedDomain
		if err := k.cdc.Unmarshal(value, &acceptedDomain); err != nil {
			return err
		}
		acceptedDomains = append(acceptedDomains, acceptedDomain)
		return nil
	})

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryAcceptedDomainResponse{AcceptedDomain: acceptedDomains, Pagination: pageRes}, nil
}

func (k Keeper) AllAnonArticlesCounters(goCtx context.Context, req *types.QueryAllAnonArticlesCountersRequest) (*types.QueryAllAnonArticlesCountersResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	var counters []types.AnonArticlesCounter
	countersStore := k.getPrefixedStore(ctx, types.KeyPrefix(types.AnonArticlesCounterKeyPrefix))
	pageRes, err := query.Paginate(countersStore, req.Pagination, func(key []byte, value []byte) error {
		var counter = types.AnonArticlesCounter{
			Key:     string(key[:]),
			Counter: binary.BigEndian.Uint64(value),
		}

		counters = append(counters, counter)
		return nil
	})

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryAllAnonArticlesCountersResponse{AnonArticlesCounters: counters, Pagination: pageRes}, nil
}

func (k Keeper) AllArticles(goCtx context.Context, req *types.QueryAllArticlesRequest) (*types.QueryAllArticlesResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	var articles []types.Article
	articlesStore := k.getPrefixedStore(ctx, types.KeyPrefix(types.ArticleKeyPrefix))
	pageRes, err := query.Paginate(articlesStore, req.Pagination, func(key []byte, value []byte) error {
		var article types.Article
		if err := k.cdc.Unmarshal(value, &article); err != nil {
			return err
		}
		articles = append(articles, article)
		return nil
	})

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryAllArticlesResponse{Article: articles, Pagination: pageRes}, nil
}

func (k Keeper) Publisher(goCtx context.Context, req *types.QueryPublisherRequest) (*types.QueryPublisherResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	publisher, found := k.GetPublisher(ctx, req.Address)
	if !found {
		return nil, status.Error(codes.InvalidArgument, "not found")
	}

	return &types.QueryPublisherResponse{Publisher: &publisher}, nil
}

func (k Keeper) Publishers(goCtx context.Context, req *types.QueryPublishersRequest) (*types.QueryPublishersResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	var publishers []types.Publisher
	publisherStore := k.getPrefixedStore(ctx, types.KeyPrefix(types.PublisherKeyPrefix))
	pageRes, err := query.Paginate(publisherStore, req.Pagination, func(key []byte, value []byte) error {
		var publisher types.Publisher
		if err := k.cdc.Unmarshal(value, &publisher); err != nil {
			return err
		}
		publishers = append(publishers, publisher)
		return nil
	})

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryPublishersResponse{Publisher: publishers, Pagination: pageRes}, nil
}
