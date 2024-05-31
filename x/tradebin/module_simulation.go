package tradebin

import (
	"math/rand"

	"github.com/bze-alphateam/bze/testutil/sample"
	tradebinsimulation "github.com/bze-alphateam/bze/x/tradebin/simulation"
	"github.com/bze-alphateam/bze/x/tradebin/types"
	"github.com/cosmos/cosmos-sdk/baseapp"
	simappparams "github.com/cosmos/cosmos-sdk/simapp/params"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	"github.com/cosmos/cosmos-sdk/x/simulation"
)

// avoid unused import issue
var (
	_ = sample.AccAddress
	_ = tradebinsimulation.FindAccount
	_ = simappparams.StakePerAccount
	_ = simulation.MsgEntryKind
	_ = baseapp.Paramspace
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

	// this line is used by starport scaffolding # simapp/module/const
)

// GenerateGenesisState creates a randomized GenState of the module
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

// ProposalContents doesn't return any content functions for governance proposals
func (AppModule) ProposalContents(_ module.SimulationState) []simtypes.WeightedProposalContent {
	return nil
}

// RandomizedParams creates randomized  param changes for the simulator
func (am AppModule) RandomizedParams(_ *rand.Rand) []simtypes.ParamChange {
	tradebinParams := types.DefaultParams()
	return []simtypes.ParamChange{
		simulation.NewSimParamChange(types.ModuleName, string(types.KeyCreateMarketFee), func(r *rand.Rand) string {
			return string(types.Amino.MustMarshalJSON(tradebinParams.CreateMarketFee))
		}),
		simulation.NewSimParamChange(types.ModuleName, string(types.KeyMarketMakerFee), func(r *rand.Rand) string {
			return string(types.Amino.MustMarshalJSON(tradebinParams.MarketMakerFee))
		}),
		simulation.NewSimParamChange(types.ModuleName, string(types.KeyMarketTakerFee), func(r *rand.Rand) string {
			return string(types.Amino.MustMarshalJSON(tradebinParams.MarketTakerFee))
		}),
		simulation.NewSimParamChange(types.ModuleName, string(types.KeyMakerFeeDestination), func(r *rand.Rand) string {
			return string(types.Amino.MustMarshalJSON(tradebinParams.MakerFeeDestination))
		}),
		simulation.NewSimParamChange(types.ModuleName, string(types.KeyTakerFeeDestination), func(r *rand.Rand) string {
			return string(types.Amino.MustMarshalJSON(tradebinParams.TakerFeeDestination))
		}),
	}
}

// RegisterStoreDecoder registers a decoder
func (am AppModule) RegisterStoreDecoder(_ sdk.StoreDecoderRegistry) {}

// WeightedOperations returns the all the gov module operations with their respective weights.
func (am AppModule) WeightedOperations(simState module.SimulationState) []simtypes.WeightedOperation {
	operations := make([]simtypes.WeightedOperation, 0)

	var weightMsgCreateMarket int
	simState.AppParams.GetOrGenerate(simState.Cdc, opWeightMsgCreateMarket, &weightMsgCreateMarket, nil,
		func(_ *rand.Rand) {
			weightMsgCreateMarket = defaultWeightMsgCreateMarket
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgCreateMarket,
		tradebinsimulation.SimulateMsgCreateMarket(am.bankKeeper, am.keeper),
	))

	var weightMsgCreateOrder int
	simState.AppParams.GetOrGenerate(simState.Cdc, opWeightMsgCreateOrder, &weightMsgCreateOrder, nil,
		func(_ *rand.Rand) {
			weightMsgCreateOrder = defaultWeightMsgCreateOrder
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgCreateOrder,
		tradebinsimulation.SimulateMsgCreateOrder(am.bankKeeper, am.keeper),
	))

	var weightMsgCancelOrder int
	simState.AppParams.GetOrGenerate(simState.Cdc, opWeightMsgCancelOrder, &weightMsgCancelOrder, nil,
		func(_ *rand.Rand) {
			weightMsgCancelOrder = defaultWeightMsgCancelOrder
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgCancelOrder,
		tradebinsimulation.SimulateMsgCancelOrder(am.bankKeeper, am.keeper),
	))

	// this line is used by starport scaffolding # simapp/module/operation

	return operations
}
