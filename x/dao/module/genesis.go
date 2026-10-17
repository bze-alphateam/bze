package dao

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/bze-alphateam/bze/x/dao/keeper"
	"github.com/bze-alphateam/bze/x/dao/types"
)

// InitGenesis initializes the module's state from a provided genesis state.
func InitGenesis(ctx sdk.Context, k *keeper.Keeper, genState types.GenesisState) {
	if err := k.SetParams(ctx, genState.Params); err != nil {
		panic(err)
	}

	k.SetAllDaos(ctx, genState.Daos)
	k.SetDaoCounter(ctx, genState.DaoCount)
	k.SetProposalCounter(ctx, genState.ProposalCount)
}

// ExportGenesis returns the module's exported genesis.
func ExportGenesis(ctx sdk.Context, k *keeper.Keeper) *types.GenesisState {
	genesis := types.DefaultGenesis()
	genesis.Params = k.GetParams(ctx)
	genesis.Daos = k.GetAllDaos(ctx)
	genesis.DaoCount = k.GetDaoCounter(ctx)
	genesis.ProposalCount = k.GetProposalCounter(ctx)

	return genesis
}
