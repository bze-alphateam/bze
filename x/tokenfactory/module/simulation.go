package tokenfactory

import (
	"math/rand"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	"github.com/cosmos/cosmos-sdk/x/simulation"

	"github.com/bze-alphateam/bze/testutil/sample"
	tokenfactorysimulation "github.com/bze-alphateam/bze/x/tokenfactory/simulation"
	"github.com/bze-alphateam/bze/x/tokenfactory/types"
)

// avoid unused import issue
var (
	_ = tokenfactorysimulation.FindAccount
	_ = rand.Rand{}
	_ = sample.AccAddress
	_ = sdk.AccAddress{}
	_ = simulation.MsgEntryKind
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

// GenerateGenesisState creates a randomized GenState of the module.
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

// RegisterStoreDecoder registers a decoder.
func (am AppModule) RegisterStoreDecoder(_ simtypes.StoreDecoderRegistry) {}

// WeightedOperations returns the all the gov module operations with their respective weights.
func (am AppModule) WeightedOperations(simState module.SimulationState) []simtypes.WeightedOperation {
	operations := make([]simtypes.WeightedOperation, 0)

	var weightMsgCreateDenom int
	simState.AppParams.GetOrGenerate(opWeightMsgCreateDenom, &weightMsgCreateDenom, nil,
		func(_ *rand.Rand) {
			weightMsgCreateDenom = defaultWeightMsgCreateDenom
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgCreateDenom,
		tokenfactorysimulation.SimulateMsgCreateDenom(am.accountKeeper, am.bankKeeper, am.keeper),
	))

	var weightMsgMint int
	simState.AppParams.GetOrGenerate(opWeightMsgMint, &weightMsgMint, nil,
		func(_ *rand.Rand) {
			weightMsgMint = defaultWeightMsgMint
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgMint,
		tokenfactorysimulation.SimulateMsgMint(am.accountKeeper, am.bankKeeper, am.keeper),
	))

	var weightMsgBurn int
	simState.AppParams.GetOrGenerate(opWeightMsgBurn, &weightMsgBurn, nil,
		func(_ *rand.Rand) {
			weightMsgBurn = defaultWeightMsgBurn
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgBurn,
		tokenfactorysimulation.SimulateMsgBurn(am.accountKeeper, am.bankKeeper, am.keeper),
	))

	var weightMsgChangeAdmin int
	simState.AppParams.GetOrGenerate(opWeightMsgChangeAdmin, &weightMsgChangeAdmin, nil,
		func(_ *rand.Rand) {
			weightMsgChangeAdmin = defaultWeightMsgChangeAdmin
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgChangeAdmin,
		tokenfactorysimulation.SimulateMsgChangeAdmin(am.accountKeeper, am.bankKeeper, am.keeper),
	))

	var weightMsgSetDenomMetadata int
	simState.AppParams.GetOrGenerate(opWeightMsgSetDenomMetadata, &weightMsgSetDenomMetadata, nil,
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

// ProposalMsgs returns msgs used for governance proposals for simulations.
func (am AppModule) ProposalMsgs(simState module.SimulationState) []simtypes.WeightedProposalMsg {
	return []simtypes.WeightedProposalMsg{
		simulation.NewWeightedProposalMsg(
			opWeightMsgCreateDenom,
			defaultWeightMsgCreateDenom,
			func(r *rand.Rand, ctx sdk.Context, accs []simtypes.Account) sdk.Msg {
				tokenfactorysimulation.SimulateMsgCreateDenom(am.accountKeeper, am.bankKeeper, am.keeper)
				return nil
			},
		),
		simulation.NewWeightedProposalMsg(
			opWeightMsgMint,
			defaultWeightMsgMint,
			func(r *rand.Rand, ctx sdk.Context, accs []simtypes.Account) sdk.Msg {
				tokenfactorysimulation.SimulateMsgMint(am.accountKeeper, am.bankKeeper, am.keeper)
				return nil
			},
		),
		simulation.NewWeightedProposalMsg(
			opWeightMsgBurn,
			defaultWeightMsgBurn,
			func(r *rand.Rand, ctx sdk.Context, accs []simtypes.Account) sdk.Msg {
				tokenfactorysimulation.SimulateMsgBurn(am.accountKeeper, am.bankKeeper, am.keeper)
				return nil
			},
		),
		simulation.NewWeightedProposalMsg(
			opWeightMsgChangeAdmin,
			defaultWeightMsgChangeAdmin,
			func(r *rand.Rand, ctx sdk.Context, accs []simtypes.Account) sdk.Msg {
				tokenfactorysimulation.SimulateMsgChangeAdmin(am.accountKeeper, am.bankKeeper, am.keeper)
				return nil
			},
		),
		simulation.NewWeightedProposalMsg(
			opWeightMsgSetDenomMetadata,
			defaultWeightMsgSetDenomMetadata,
			func(r *rand.Rand, ctx sdk.Context, accs []simtypes.Account) sdk.Msg {
				tokenfactorysimulation.SimulateMsgSetDenomMetadata(am.accountKeeper, am.bankKeeper, am.keeper)
				return nil
			},
		),
		// this line is used by starport scaffolding # simapp/module/OpMsg
	}
}
