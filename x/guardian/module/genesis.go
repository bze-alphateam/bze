package guardian

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/bze-alphateam/bze/x/guardian/keeper"
	"github.com/bze-alphateam/bze/x/guardian/types"
)

// InitGenesis initializes the module's state from a provided genesis state.
func InitGenesis(ctx sdk.Context, k *keeper.Keeper, genState types.GenesisState) {
	if err := genState.Validate(); err != nil {
		panic(err)
	}

	if err := k.SetParams(ctx, genState.Params); err != nil {
		panic(err)
	}

	k.SetAutomationPaused(ctx, genState.AutomationPaused)
	for _, channelID := range genState.ClosedChannels {
		k.SetClosedChannel(ctx, channelID)
	}
}

// ExportGenesis returns the module's exported genesis.
func ExportGenesis(ctx sdk.Context, k *keeper.Keeper) *types.GenesisState {
	genesis := types.DefaultGenesis()
	genesis.Params = k.GetParams(ctx)
	genesis.AutomationPaused = k.IsAutomationPaused(ctx)
	genesis.ClosedChannels = k.GetAllClosedChannels(ctx)

	return genesis
}
