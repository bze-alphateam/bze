package keeper

import (
	"context"
	types2 "github.com/bze-alphateam/bze/x/burner/types"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	"github.com/cosmos/cosmos-sdk/types/query"

	"github.com/bze-alphateam/bze/x/cointrunk/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) AllBurnedCoins(goCtx context.Context, req *types.QueryAllBurnedCoinsRequest) (*types.QueryAllBurnedCoinsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	var burnedCoins []types.BurnedCoins
	store := ctx.KVStore(k.storeKey)
	burnedCoinsStore := prefix.NewStore(store, types.KeyPrefix(types2.BurnedCoinsKeyPrefix))
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
