package keeper

import (
	"context"

	"github.com/bze-alphateam/bze/x/burner/types"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var _ types.QueryServer = Keeper{}

func (k Keeper) Params(c context.Context, req *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(c)

	return &types.QueryParamsResponse{Params: k.GetParams(ctx)}, nil
}

func (k Keeper) AllBurnedCoins(goCtx context.Context, req *types.QueryAllBurnedCoinsRequest) (*types.QueryAllBurnedCoinsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	var burnedCoins []types.BurnedCoins
	store := ctx.KVStore(k.storeKey)
	burnedCoinsStore := prefix.NewStore(store, types.KeyPrefix(types.BurnedCoinsKeyPrefix))
	pageRes, err := query.Paginate(burnedCoinsStore, req.Pagination, func(key []byte, value []byte) error {
		var entry types.BurnedCoins
		if err := k.cdc.Unmarshal(value, &entry); err != nil {
			return err
		}
		burnedCoins = append(burnedCoins, entry)
		return nil
	})

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	_ = ctx

	return &types.QueryAllBurnedCoinsResponse{BurnedCoins: burnedCoins, Pagination: pageRes}, nil
}
