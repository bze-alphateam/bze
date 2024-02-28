package tradebin

import (
	"github.com/bze-alphateam/bze/x/tradebin/keeper"
	"github.com/bze-alphateam/bze/x/tradebin/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// InitGenesis initializes the capability module's state from a provided genesis
// state.
func InitGenesis(ctx sdk.Context, k keeper.Keeper, genState types.GenesisState) {
	// Set all the market
	for _, elem := range genState.MarketList {
		k.SetMarket(ctx, elem)
	}

	for _, elem := range genState.QueueMessageList {
		k.SetQueueMessage(ctx, elem)
	}

	// this line is used by starport scaffolding # genesis/module/init
	k.SetParams(ctx, genState.Params)
	k.SetQueueMessageCounter(ctx, 0)
}

// ExportGenesis returns the capability module's exported genesis.
func ExportGenesis(ctx sdk.Context, k keeper.Keeper) *types.GenesisState {
	genesis := types.DefaultGenesis()
	genesis.Params = k.GetParams(ctx)

	genesis.MarketList = k.GetAllMarket(ctx)
	genesis.QueueMessageList = k.GetAllQueueMessage(ctx)
	// this line is used by starport scaffolding # genesis/module/export

	return genesis
}
