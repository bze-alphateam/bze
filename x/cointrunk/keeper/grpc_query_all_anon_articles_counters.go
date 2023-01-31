package keeper

import (
	"context"
	"encoding/binary"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	"github.com/cosmos/cosmos-sdk/types/query"

	"github.com/bze-alphateam/bze/x/cointrunk/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) AllAnonArticlesCounters(goCtx context.Context, req *types.QueryAllAnonArticlesCountersRequest) (*types.QueryAllAnonArticlesCountersResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	var counters []types.AnonArticlesCounter
	store := ctx.KVStore(k.storeKey)
	countersStore := prefix.NewStore(store, types.KeyPrefix(types.AnonArticlesCounterKeyPrefix))
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
	_ = ctx

	return &types.QueryAllAnonArticlesCountersResponse{AnonArticlesCounters: counters, Pagination: pageRes}, nil
}
