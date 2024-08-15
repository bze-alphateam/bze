package burner

import (
	"github.com/bze-alphateam/bze/x/burner/keeper"
	"github.com/bze-alphateam/bze/x/burner/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// InitGenesis initializes the capability module's state from a provided genesis
// state.
func InitGenesis(ctx sdk.Context, k keeper.Keeper, genState types.GenesisState) {
	// this line is used by starport scaffolding # genesis/module/init
	k.SetParams(ctx, genState.Params)

	for _, burnedCoins := range genState.BurnedCoinsList {
		k.SetBurnedCoins(ctx, burnedCoins)
	}

	for _, raffle := range genState.RaffleList {
		k.SetRaffle(ctx, raffle)
		k.SetRaffleDeleteHook(ctx, types.RaffleDeleteHook{
			Denom: raffle.Denom,
			EndAt: raffle.EndAt,
		})
	}

	for _, winner := range genState.RaffleWinnersList {
		k.SetRaffleWinner(ctx, winner)
	}

	for _, part := range genState.RaffleParticipantsList {
		k.SetRaffleParticipant(ctx, part)
	}
}

// ExportGenesis returns the capability module's exported genesis.
func ExportGenesis(ctx sdk.Context, k keeper.Keeper) *types.GenesisState {
	genesis := types.DefaultGenesis()
	genesis.Params = k.GetParams(ctx)
	genesis.BurnedCoinsList = k.GetAllBurnedCoins(ctx)
	genesis.RaffleList = k.GetAllRaffle(ctx)

	var winnersList []types.RaffleWinner
	for _, raffle := range genesis.RaffleList {
		w := k.GetRaffleWinners(ctx, raffle.Denom)
		winnersList = append(winnersList, w...)
	}

	genesis.RaffleWinnersList = winnersList
	genesis.RaffleParticipantsList = k.GetAllRaffleParticipants(ctx)
	// this line is used by starport scaffolding # genesis/module/export

	return genesis
}
