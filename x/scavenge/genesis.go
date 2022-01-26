package scavenge

import (
	"github.com/bze-alphateam/bze/x/scavenge/keeper"
	"github.com/bze-alphateam/bze/x/scavenge/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// InitGenesis initializes the capability module's state from a provided genesis
// state.
func InitGenesis(ctx sdk.Context, k keeper.Keeper, genState types.GenesisState) {
	// Set all the scavenge
	for _, elem := range genState.ScavengeList {
		k.SetScavenge(ctx, elem)
	}
	// Set all the commit
	for _, elem := range genState.CommitList {
		k.SetCommit(ctx, elem)
	}
	// this line is used by starport scaffolding # genesis/module/init
}

// ExportGenesis returns the capability module's exported genesis.
func ExportGenesis(ctx sdk.Context, k keeper.Keeper) *types.GenesisState {
	genesis := types.DefaultGenesis()

	genesis.ScavengeList = k.GetAllScavenge(ctx)
	genesis.CommitList = k.GetAllCommit(ctx)
	// this line is used by starport scaffolding # genesis/module/export

	return genesis
}
