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

func (k Keeper) Publisher(goCtx context.Context, req *types.QueryPublisherRequest) (*types.QueryPublisherResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	var publishers []types.Publisher
	store := ctx.KVStore(k.storeKey)
	publisherStore := prefix.NewStore(store, types.KeyPrefix(types.PublisherKeyPrefix))
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

	return &types.QueryPublisherResponse{Publisher: publishers, Pagination: pageRes}, nil
}
