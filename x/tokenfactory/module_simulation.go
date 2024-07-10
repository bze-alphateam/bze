package tokenfactory

import (
	"math/rand"

	"github.com/bze-alphateam/bze/testutil/sample"
	tokenfactorysimulation "github.com/bze-alphateam/bze/x/tokenfactory/simulation"
	"github.com/bze-alphateam/bze/x/tokenfactory/types"
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
	_ = tokenfactorysimulation.FindAccount
	_ = simappparams.StakePerAccount
	_ = simulation.MsgEntryKind
	_ = baseapp.Paramspace
)

const (
	opWeightMsgCreateDenom = "op_weight_msg_create_denom"
	// TODO: Determine the simulation weight value
	defaultWeightMsgCreateDenom int = 100

	opWeightMsgMint = "op_weight_msg_mint"
	// TODO: Determine the simulation weight value
	defaultWeightMsgMint int = 100

	opWeightMsgBurn = "op_weight_msg_burn"
	// TODO: Determine the simulation weight value
	defaultWeightMsgBurn int = 100

	opWeightMsgChangeAdmin = "op_weight_msg_change_admin"
	// TODO: Determine the simulation weight value
	defaultWeightMsgChangeAdmin int = 100

	opWeightMsgSetDenomMetadata = "op_weight_msg_set_denom_metadata"
	// TODO: Determine the simulation weight value
	defaultWeightMsgSetDenomMetadata int = 100

	// this line is used by starport scaffolding # simapp/module/const
)

// GenerateGenesisState creates a randomized GenState of the module
func (AppModule) GenerateGenesisState(simState *module.SimulationState) {
	accs := make([]string, len(simState.Accounts))
	for i, acc := range simState.Accounts {
		accs[i] = acc.Address.String()
	}
	tokenfactoryGenesis := types.GenesisState{
		Params: types.DefaultParams(),
		// this line is used by starport scaffolding # simapp/module/genesisState
	}
	simState.GenState[types.ModuleName] = simState.Cdc.MustMarshalJSON(&tokenfactoryGenesis)
}

// ProposalContents doesn't return any content functions for governance proposals
func (AppModule) ProposalContents(_ module.SimulationState) []simtypes.WeightedProposalContent {
	return nil
}

// RandomizedParams creates randomized  param changes for the simulator
func (am AppModule) RandomizedParams(_ *rand.Rand) []simtypes.ParamChange {
	tokenfactoryParams := types.DefaultParams()
	return []simtypes.ParamChange{
		simulation.NewSimParamChange(types.ModuleName, string(types.KeyCreateDenomFee), func(r *rand.Rand) string {
			return string(types.Amino.MustMarshalJSON(tokenfactoryParams.CreateDenomFee))
		}),
	}
}

// RegisterStoreDecoder registers a decoder
func (am AppModule) RegisterStoreDecoder(_ sdk.StoreDecoderRegistry) {}

// WeightedOperations returns the all the gov module operations with their respective weights.
func (am AppModule) WeightedOperations(simState module.SimulationState) []simtypes.WeightedOperation {
	operations := make([]simtypes.WeightedOperation, 0)

	var weightMsgCreateDenom int
	simState.AppParams.GetOrGenerate(simState.Cdc, opWeightMsgCreateDenom, &weightMsgCreateDenom, nil,
		func(_ *rand.Rand) {
			weightMsgCreateDenom = defaultWeightMsgCreateDenom
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgCreateDenom,
		tokenfactorysimulation.SimulateMsgCreateDenom(am.accountKeeper, am.bankKeeper, am.keeper),
	))

	var weightMsgMint int
	simState.AppParams.GetOrGenerate(simState.Cdc, opWeightMsgMint, &weightMsgMint, nil,
		func(_ *rand.Rand) {
			weightMsgMint = defaultWeightMsgMint
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgMint,
		tokenfactorysimulation.SimulateMsgMint(am.accountKeeper, am.bankKeeper, am.keeper),
	))

	var weightMsgBurn int
	simState.AppParams.GetOrGenerate(simState.Cdc, opWeightMsgBurn, &weightMsgBurn, nil,
		func(_ *rand.Rand) {
			weightMsgBurn = defaultWeightMsgBurn
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgBurn,
		tokenfactorysimulation.SimulateMsgBurn(am.accountKeeper, am.bankKeeper, am.keeper),
	))

	var weightMsgChangeAdmin int
	simState.AppParams.GetOrGenerate(simState.Cdc, opWeightMsgChangeAdmin, &weightMsgChangeAdmin, nil,
		func(_ *rand.Rand) {
			weightMsgChangeAdmin = defaultWeightMsgChangeAdmin
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgChangeAdmin,
		tokenfactorysimulation.SimulateMsgChangeAdmin(am.accountKeeper, am.bankKeeper, am.keeper),
	))

	var weightMsgSetDenomMetadata int
	simState.AppParams.GetOrGenerate(simState.Cdc, opWeightMsgSetDenomMetadata, &weightMsgSetDenomMetadata, nil,
		func(_ *rand.Rand) {
			weightMsgSetDenomMetadata = defaultWeightMsgSetDenomMetadata
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgSetDenomMetadata,
		tokenfactorysimulation.SimulateMsgSetDenomMetadata(am.accountKeeper, am.bankKeeper, am.keeper),
	))

	// this line is used by starport scaffolding # simapp/module/operation

	return operations
}
