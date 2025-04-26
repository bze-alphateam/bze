package burner

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/bze-alphateam/bze/x/burner/keeper"
	"github.com/bze-alphateam/bze/x/burner/types"
)

// InitGenesis initializes the module's state from a provided genesis state.
func InitGenesis(ctx sdk.Context, k keeper.Keeper, genState types.GenesisState) {
	// this line is used by starport scaffolding # genesis/module/init
	if err := k.SetParams(ctx, genState.Params); err != nil {
		panic(err)
	}

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

	k.SetParticipantCounter(ctx, genState.RaffleParticipantCounter)
	k.InitGenesis(ctx)
}

// ExportGenesis returns the module's exported genesis.
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
	genesis.RaffleParticipantCounter = k.GetParticipantCounter(ctx)

	// this line is used by starport scaffolding # genesis/module/export

	return genesis
}
