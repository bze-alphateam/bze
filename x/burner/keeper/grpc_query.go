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

	return &types.QueryAllBurnedCoinsResponse{BurnedCoins: burnedCoins, Pagination: pageRes}, nil
}

func (k Keeper) Raffles(goCtx context.Context, req *types.QueryRafflesRequest) (*types.QueryRafflesResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	var items []types.Raffle
	store := ctx.KVStore(k.storeKey)
	raffleStore := prefix.NewStore(store, types.KeyPrefix(types.RaffleKeyPrefix))
	pageRes, err := query.Paginate(raffleStore, req.Pagination, func(key []byte, value []byte) error {
		var entry types.Raffle
		if err := k.cdc.Unmarshal(value, &entry); err != nil {

			return err
		}
		items = append(items, entry)

		return nil
	})

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryRafflesResponse{List: items, Pagination: pageRes}, nil
}

func (k Keeper) RaffleWinners(goCtx context.Context, req *types.QueryRaffleWinnersRequest) (*types.QueryRaffleWinnersResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	var items []types.RaffleWinner
	store := ctx.KVStore(k.storeKey)
	raffleStore := prefix.NewStore(store, types.KeyPrefix(string(types.GetRaffleWinnerKeyPrefix(req.Denom))))
	pageRes, err := query.Paginate(raffleStore, req.Pagination, func(key []byte, value []byte) error {
		var entry types.RaffleWinner
		if err := k.cdc.Unmarshal(value, &entry); err != nil {

			return err
		}
		items = append(items, entry)

		return nil
	})

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryRaffleWinnersResponse{List: items, Pagination: pageRes}, nil
}
