package burner

import (
	"math/rand"

	"github.com/bze-alphateam/bze/testutil/sample"
	burnersimulation "github.com/bze-alphateam/bze/x/burner/simulation"
	"github.com/bze-alphateam/bze/x/burner/types"
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
	_ = burnersimulation.FindAccount
	_ = simappparams.StakePerAccount
	_ = simulation.MsgEntryKind
	_ = baseapp.Paramspace
)

const (
	opWeightMsgFundBurner = "op_weight_msg_fund_burner"
	// TODO: Determine the simulation weight value
	defaultWeightMsgFundBurner int = 100

	opWeightMsgStartRaffle = "op_weight_msg_start_raffle"
	// TODO: Determine the simulation weight value
	defaultWeightMsgStartRaffle int = 100

	// this line is used by starport scaffolding # simapp/module/const
)

// GenerateGenesisState creates a randomized GenState of the module
func (AppModule) GenerateGenesisState(simState *module.SimulationState) {
	accs := make([]string, len(simState.Accounts))
	for i, acc := range simState.Accounts {
		accs[i] = acc.Address.String()
	}
	burnerGenesis := types.GenesisState{
		Params: types.DefaultParams(),
		// this line is used by starport scaffolding # simapp/module/genesisState
	}
	simState.GenState[types.ModuleName] = simState.Cdc.MustMarshalJSON(&burnerGenesis)
}

// ProposalContents doesn't return any content functions for governance proposals
func (AppModule) ProposalContents(_ module.SimulationState) []simtypes.WeightedProposalContent {
	return nil
}

// RandomizedParams creates randomized  param changes for the simulator
func (am AppModule) RandomizedParams(_ *rand.Rand) []simtypes.ParamChange {
	return []simtypes.ParamChange{}
}

// RegisterStoreDecoder registers a decoder
func (am AppModule) RegisterStoreDecoder(_ sdk.StoreDecoderRegistry) {}

// WeightedOperations returns the all the gov module operations with their respective weights.
func (am AppModule) WeightedOperations(simState module.SimulationState) []simtypes.WeightedOperation {
	operations := make([]simtypes.WeightedOperation, 0)

	var weightMsgFundBurner int
	simState.AppParams.GetOrGenerate(simState.Cdc, opWeightMsgFundBurner, &weightMsgFundBurner, nil,
		func(_ *rand.Rand) {
			weightMsgFundBurner = defaultWeightMsgFundBurner
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgFundBurner,
		burnersimulation.SimulateMsgFundBurner(am.accountKeeper, am.bankKeeper, am.keeper),
	))

	var weightMsgStartRaffle int
	simState.AppParams.GetOrGenerate(simState.Cdc, opWeightMsgStartRaffle, &weightMsgStartRaffle, nil,
		func(_ *rand.Rand) {
			weightMsgStartRaffle = defaultWeightMsgStartRaffle
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgStartRaffle,
		burnersimulation.SimulateMsgStartRaffle(am.accountKeeper, am.bankKeeper, am.keeper),
	))

	// this line is used by starport scaffolding # simapp/module/operation

	return operations
}
