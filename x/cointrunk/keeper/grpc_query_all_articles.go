package keeper

import (
	"context"

	"github.com/bze-alphateam/bze/x/cointrunk/types"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) AllArticles(goCtx context.Context, req *types.QueryAllArticlesRequest) (*types.QueryAllArticlesResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	var articles []types.Article
	store := ctx.KVStore(k.storeKey)
	articlesStore := prefix.NewStore(store, types.KeyPrefix(types.ArticleKeyPrefix))
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

	_ = ctx

	return &types.QueryAllArticlesResponse{Article: articles, Pagination: pageRes}, nil
}
