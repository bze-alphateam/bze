package keeper

import (
	"context"
	"github.com/bze-alphateam/bze/x/burner/types"
)

var _ types.QueryServer = Keeper{}

func (k Keeper) Raffles(ctx context.Context, request *types.QueryRafflesRequest) (*types.QueryRafflesResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (k Keeper) RaffleWinners(ctx context.Context, request *types.QueryRaffleWinnersRequest) (*types.QueryRaffleWinnersResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (k Keeper) AllBurnedCoins(ctx context.Context, request *types.QueryAllBurnedCoinsRequest) (*types.QueryAllBurnedCoinsResponse, error) {
	//TODO implement me
	panic("implement me")
}
