package keeper

import (
	"context"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/bze-alphateam/bze/x/burner/types"
)

var _ types.QueryServer = Keeper{}

func (k Keeper) Raffles(goCtx context.Context, request *types.QueryRafflesRequest) (*types.QueryRafflesResponse, error) {
	if request == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)

	var items []types.Raffle
	raffleStore := k.getPrefixedStore(ctx, types.KeyPrefix(types.RaffleKeyPrefix))
	pageRes, err := query.Paginate(raffleStore, request.Pagination, func(key []byte, value []byte) error {
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

func (k Keeper) RaffleWinners(goCtx context.Context, request *types.QueryRaffleWinnersRequest) (*types.QueryRaffleWinnersResponse, error) {
	if request == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	if request.Denom == "" {
		return nil, status.Error(codes.InvalidArgument, "denom required")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	var items []types.RaffleWinner
	raffleStore := k.getPrefixedStore(ctx, types.KeyPrefix(string(types.GetRaffleWinnerKeyPrefix(request.Denom))))
	pageRes, err := query.Paginate(raffleStore, request.Pagination, func(key []byte, value []byte) error {
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

func (k Keeper) AllBurnedCoins(goCtx context.Context, request *types.QueryAllBurnedCoinsRequest) (*types.QueryAllBurnedCoinsResponse, error) {
	if request == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	var burnedCoins []types.BurnedCoins
	store := k.getPrefixedStore(ctx, types.KeyPrefix(types.BurnedCoinsKeyPrefix))
	pageRes, err := query.Paginate(store, request.Pagination, func(key []byte, value []byte) error {
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
