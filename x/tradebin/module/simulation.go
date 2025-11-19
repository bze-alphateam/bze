package tradebin

import (
	"math/rand"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	"github.com/cosmos/cosmos-sdk/x/simulation"

	"github.com/bze-alphateam/bze/testutil/sample"
	tradebinsimulation "github.com/bze-alphateam/bze/x/tradebin/simulation"
	"github.com/bze-alphateam/bze/x/tradebin/types"
)

// avoid unused import issue
var (
	_ = tradebinsimulation.FindAccount
	_ = rand.Rand{}
	_ = sample.AccAddress
	_ = sdk.AccAddress{}
	_ = simulation.MsgEntryKind
)

const (
	opWeightMsgCreateMarket = "op_weight_msg_create_market"
	// TODO: Determine the simulation weight value
	defaultWeightMsgCreateMarket int = 100

	opWeightMsgCreateOrder = "op_weight_msg_create_order"
	// TODO: Determine the simulation weight value
	defaultWeightMsgCreateOrder int = 100

	opWeightMsgCancelOrder = "op_weight_msg_cancel_order"
	// TODO: Determine the simulation weight value
	defaultWeightMsgCancelOrder int = 100

	opWeightMsgFillOrders = "op_weight_msg_fill_orders"
	// TODO: Determine the simulation weight value
	defaultWeightMsgFillOrders int = 100

	opWeightMsgCreateLiquidityPool = "op_weight_msg_create_liquidity_pool"
	// TODO: Determine the simulation weight value
	defaultWeightMsgCreateLiquidityPool int = 100

	opWeightMsgAddLiquidity = "op_weight_msg_add_liquidity"
	// TODO: Determine the simulation weight value
	defaultWeightMsgAddLiquidity int = 100

	opWeightMsgRemoveLiquidity = "op_weight_msg_remove_liquidity"
	// TODO: Determine the simulation weight value
	defaultWeightMsgRemoveLiquidity int = 100

	opWeightMsgMultiSwap = "op_weight_msg_multi_swap"
	// TODO: Determine the simulation weight value
	defaultWeightMsgMultiSwap int = 100

	// this line is used by starport scaffolding # simapp/module/const
)

// GenerateGenesisState creates a randomized GenState of the module.
func (AppModule) GenerateGenesisState(simState *module.SimulationState) {
	accs := make([]string, len(simState.Accounts))
	for i, acc := range simState.Accounts {
		accs[i] = acc.Address.String()
	}
	tradebinGenesis := types.GenesisState{
		Params: types.DefaultParams(),
		// this line is used by starport scaffolding # simapp/module/genesisState
	}
	simState.GenState[types.ModuleName] = simState.Cdc.MustMarshalJSON(&tradebinGenesis)
}

// RegisterStoreDecoder registers a decoder.
func (am AppModule) RegisterStoreDecoder(_ simtypes.StoreDecoderRegistry) {}

// WeightedOperations returns the all the gov module operations with their respective weights.
func (am AppModule) WeightedOperations(simState module.SimulationState) []simtypes.WeightedOperation {
	operations := make([]simtypes.WeightedOperation, 0)

	var weightMsgCreateMarket int
	simState.AppParams.GetOrGenerate(opWeightMsgCreateMarket, &weightMsgCreateMarket, nil,
		func(_ *rand.Rand) {
			weightMsgCreateMarket = defaultWeightMsgCreateMarket
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgCreateMarket,
		tradebinsimulation.SimulateMsgCreateMarket(am.accountKeeper, am.bankKeeper, *am.keeper),
	))

	var weightMsgCreateOrder int
	simState.AppParams.GetOrGenerate(opWeightMsgCreateOrder, &weightMsgCreateOrder, nil,
		func(_ *rand.Rand) {
			weightMsgCreateOrder = defaultWeightMsgCreateOrder
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgCreateOrder,
		tradebinsimulation.SimulateMsgCreateOrder(am.accountKeeper, am.bankKeeper, *am.keeper),
	))

	var weightMsgCancelOrder int
	simState.AppParams.GetOrGenerate(opWeightMsgCancelOrder, &weightMsgCancelOrder, nil,
		func(_ *rand.Rand) {
			weightMsgCancelOrder = defaultWeightMsgCancelOrder
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgCancelOrder,
		tradebinsimulation.SimulateMsgCancelOrder(am.accountKeeper, am.bankKeeper, *am.keeper),
	))

	var weightMsgFillOrders int
	simState.AppParams.GetOrGenerate(opWeightMsgFillOrders, &weightMsgFillOrders, nil,
		func(_ *rand.Rand) {
			weightMsgFillOrders = defaultWeightMsgFillOrders
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgFillOrders,
		tradebinsimulation.SimulateMsgFillOrders(am.accountKeeper, am.bankKeeper, *am.keeper),
	))

	var weightMsgCreateLiquidityPool int
	simState.AppParams.GetOrGenerate(opWeightMsgCreateLiquidityPool, &weightMsgCreateLiquidityPool, nil,
		func(_ *rand.Rand) {
			weightMsgCreateLiquidityPool = defaultWeightMsgCreateLiquidityPool
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgCreateLiquidityPool,
		tradebinsimulation.SimulateMsgCreateLiquidityPool(am.accountKeeper, am.bankKeeper, *am.keeper),
	))

	var weightMsgAddLiquidity int
	simState.AppParams.GetOrGenerate(opWeightMsgAddLiquidity, &weightMsgAddLiquidity, nil,
		func(_ *rand.Rand) {
			weightMsgAddLiquidity = defaultWeightMsgAddLiquidity
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgAddLiquidity,
		tradebinsimulation.SimulateMsgAddLiquidity(am.accountKeeper, am.bankKeeper, *am.keeper),
	))

	var weightMsgRemoveLiquidity int
	simState.AppParams.GetOrGenerate(opWeightMsgRemoveLiquidity, &weightMsgRemoveLiquidity, nil,
		func(_ *rand.Rand) {
			weightMsgRemoveLiquidity = defaultWeightMsgRemoveLiquidity
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgRemoveLiquidity,
		tradebinsimulation.SimulateMsgRemoveLiquidity(am.accountKeeper, am.bankKeeper, *am.keeper),
	))

	var weightMsgMultiSwap int
	simState.AppParams.GetOrGenerate(opWeightMsgMultiSwap, &weightMsgMultiSwap, nil,
		func(_ *rand.Rand) {
			weightMsgMultiSwap = defaultWeightMsgMultiSwap
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgMultiSwap,
		tradebinsimulation.SimulateMsgMultiSwap(am.accountKeeper, am.bankKeeper, *am.keeper),
	))

	// this line is used by starport scaffolding # simapp/module/operation

	return operations
}

// ProposalMsgs returns msgs used for governance proposals for simulations.
func (am AppModule) ProposalMsgs(_ module.SimulationState) []simtypes.WeightedProposalMsg {
	return []simtypes.WeightedProposalMsg{
		simulation.NewWeightedProposalMsg(
			opWeightMsgCreateMarket,
			defaultWeightMsgCreateMarket,
			func(r *rand.Rand, ctx sdk.Context, accs []simtypes.Account) sdk.Msg {
				tradebinsimulation.SimulateMsgCreateMarket(am.accountKeeper, am.bankKeeper, *am.keeper)
				return nil
			},
		),
		simulation.NewWeightedProposalMsg(
			opWeightMsgCreateOrder,
			defaultWeightMsgCreateOrder,
			func(r *rand.Rand, ctx sdk.Context, accs []simtypes.Account) sdk.Msg {
				tradebinsimulation.SimulateMsgCreateOrder(am.accountKeeper, am.bankKeeper, *am.keeper)
				return nil
			},
		),
		simulation.NewWeightedProposalMsg(
			opWeightMsgCancelOrder,
			defaultWeightMsgCancelOrder,
			func(r *rand.Rand, ctx sdk.Context, accs []simtypes.Account) sdk.Msg {
				tradebinsimulation.SimulateMsgCancelOrder(am.accountKeeper, am.bankKeeper, *am.keeper)
				return nil
			},
		),
		simulation.NewWeightedProposalMsg(
			opWeightMsgFillOrders,
			defaultWeightMsgFillOrders,
			func(r *rand.Rand, ctx sdk.Context, accs []simtypes.Account) sdk.Msg {
				tradebinsimulation.SimulateMsgFillOrders(am.accountKeeper, am.bankKeeper, *am.keeper)
				return nil
			},
		),
		simulation.NewWeightedProposalMsg(
			opWeightMsgCreateLiquidityPool,
			defaultWeightMsgCreateLiquidityPool,
			func(r *rand.Rand, ctx sdk.Context, accs []simtypes.Account) sdk.Msg {
				tradebinsimulation.SimulateMsgCreateLiquidityPool(am.accountKeeper, am.bankKeeper, *am.keeper)
				return nil
			},
		),
		simulation.NewWeightedProposalMsg(
			opWeightMsgAddLiquidity,
			defaultWeightMsgAddLiquidity,
			func(r *rand.Rand, ctx sdk.Context, accs []simtypes.Account) sdk.Msg {
				tradebinsimulation.SimulateMsgAddLiquidity(am.accountKeeper, am.bankKeeper, *am.keeper)
				return nil
			},
		),
		simulation.NewWeightedProposalMsg(
			opWeightMsgRemoveLiquidity,
			defaultWeightMsgRemoveLiquidity,
			func(r *rand.Rand, ctx sdk.Context, accs []simtypes.Account) sdk.Msg {
				tradebinsimulation.SimulateMsgRemoveLiquidity(am.accountKeeper, am.bankKeeper, *am.keeper)
				return nil
			},
		),
		simulation.NewWeightedProposalMsg(
			opWeightMsgMultiSwap,
			defaultWeightMsgMultiSwap,
			func(r *rand.Rand, ctx sdk.Context, accs []simtypes.Account) sdk.Msg {
				tradebinsimulation.SimulateMsgMultiSwap(am.accountKeeper, am.bankKeeper, *am.keeper)
				return nil
			},
		),
		// this line is used by starport scaffolding # simapp/module/OpMsg
	}
}
