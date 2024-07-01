package tradebin

import (
	"github.com/bze-alphateam/bze/x/tradebin/keeper"
	"github.com/bze-alphateam/bze/x/tradebin/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"strconv"
)

// InitGenesis initializes the capability module's state from a provided genesis
// state.
func InitGenesis(ctx sdk.Context, k keeper.Keeper, genState types.GenesisState) {
	// Set all the market
	for _, elem := range genState.MarketList {
		k.SetMarket(ctx, elem)
	}

	var qmCounter uint64
	for _, elem := range genState.QueueMessageList {
		qmCounter++
		k.SetQueueMessage(ctx, elem)
	}

	for _, elem := range genState.OrderList {
		k.SaveOrder(ctx, elem)
	}

	for _, elem := range genState.AggregatedOrderList {
		k.SetAggregatedOrder(ctx, elem)
	}

	for key, elem := range genState.HistoryOrderList {
		k.SetHistoryOrder(ctx, elem, strconv.Itoa(key))
	}

	for _, elem := range genState.AllUsersDust {
		k.SetUserDust(ctx, elem)
	}

	// this line is used by starport scaffolding # genesis/module/init
	k.SetParams(ctx, genState.Params)
	k.SetQueueMessageCounter(ctx, qmCounter)
	k.SetOrderCounter(ctx, uint64(genState.OrderCounter))
}

// ExportGenesis returns the capability module's exported genesis.
func ExportGenesis(ctx sdk.Context, k keeper.Keeper) *types.GenesisState {
	genesis := types.DefaultGenesis()
	genesis.Params = k.GetParams(ctx)

	genesis.MarketList = k.GetAllMarket(ctx)
	genesis.QueueMessageList = k.GetAllQueueMessage(ctx)

	genesis.OrderList = k.GetAllOrder(ctx)
	genesis.AggregatedOrderList = k.GetAllAggregatedOrder(ctx)
	genesis.HistoryOrderList = k.GetAllHistoryOrder(ctx)

	genesis.OrderCounter = int64(k.GetOrderCounter(ctx))
	genesis.AllUsersDust = k.GetAllUserDust(ctx)
	// this line is used by starport scaffolding # genesis/module/export

	return genesis
}
