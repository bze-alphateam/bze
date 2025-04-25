package cointrunk

import (
	"math/rand"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	"github.com/cosmos/cosmos-sdk/x/simulation"

	"github.com/bze-alphateam/bze/testutil/sample"
	cointrunksimulation "github.com/bze-alphateam/bze/x/cointrunk/simulation"
	"github.com/bze-alphateam/bze/x/cointrunk/types"
)

// avoid unused import issue
var (
	_ = cointrunksimulation.FindAccount
	_ = rand.Rand{}
	_ = sample.AccAddress
	_ = sdk.AccAddress{}
	_ = simulation.MsgEntryKind
)

const (
	opWeightMsgAddArticle = "op_weight_msg_add_article"
	// TODO: Determine the simulation weight value
	defaultWeightMsgAddArticle int = 100

	opWeightMsgPayPublisherRespect = "op_weight_msg_pay_publisher_respect"
	// TODO: Determine the simulation weight value
	defaultWeightMsgPayPublisherRespect int = 100

	opWeightMsgAcceptDomain = "op_weight_msg_accept_domain"
	// TODO: Determine the simulation weight value
	defaultWeightMsgAcceptDomain int = 100

	opWeightMsgSavePublisher = "op_weight_msg_save_publisher"
	// TODO: Determine the simulation weight value
	defaultWeightMsgSavePublisher int = 100

	// this line is used by starport scaffolding # simapp/module/const
)

// GenerateGenesisState creates a randomized GenState of the module.
func (AppModule) GenerateGenesisState(simState *module.SimulationState) {
	accs := make([]string, len(simState.Accounts))
	for i, acc := range simState.Accounts {
		accs[i] = acc.Address.String()
	}
	cointrunkGenesis := types.GenesisState{
		Params: types.DefaultParams(),
		// this line is used by starport scaffolding # simapp/module/genesisState
	}
	simState.GenState[types.ModuleName] = simState.Cdc.MustMarshalJSON(&cointrunkGenesis)
}

// RegisterStoreDecoder registers a decoder.
func (am AppModule) RegisterStoreDecoder(_ simtypes.StoreDecoderRegistry) {}

// WeightedOperations returns the all the gov module operations with their respective weights.
func (am AppModule) WeightedOperations(simState module.SimulationState) []simtypes.WeightedOperation {
	operations := make([]simtypes.WeightedOperation, 0)

	var weightMsgAddArticle int
	simState.AppParams.GetOrGenerate(opWeightMsgAddArticle, &weightMsgAddArticle, nil,
		func(_ *rand.Rand) {
			weightMsgAddArticle = defaultWeightMsgAddArticle
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgAddArticle,
		cointrunksimulation.SimulateMsgAddArticle(am.bankKeeper, am.keeper),
	))

	var weightMsgPayPublisherRespect int
	simState.AppParams.GetOrGenerate(opWeightMsgPayPublisherRespect, &weightMsgPayPublisherRespect, nil,
		func(_ *rand.Rand) {
			weightMsgPayPublisherRespect = defaultWeightMsgPayPublisherRespect
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgPayPublisherRespect,
		cointrunksimulation.SimulateMsgPayPublisherRespect(am.bankKeeper, am.keeper),
	))

	var weightMsgAcceptDomain int
	simState.AppParams.GetOrGenerate(opWeightMsgAcceptDomain, &weightMsgAcceptDomain, nil,
		func(_ *rand.Rand) {
			weightMsgAcceptDomain = defaultWeightMsgAcceptDomain
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgAcceptDomain,
		cointrunksimulation.SimulateMsgAcceptDomain(am.bankKeeper, am.keeper),
	))

	var weightMsgSavePublisher int
	simState.AppParams.GetOrGenerate(opWeightMsgSavePublisher, &weightMsgSavePublisher, nil,
		func(_ *rand.Rand) {
			weightMsgSavePublisher = defaultWeightMsgSavePublisher
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgSavePublisher,
		cointrunksimulation.SimulateMsgSavePublisher(am.bankKeeper, am.keeper),
	))

	// this line is used by starport scaffolding # simapp/module/operation

	return operations
}

// ProposalMsgs returns msgs used for governance proposals for simulations.
func (am AppModule) ProposalMsgs(simState module.SimulationState) []simtypes.WeightedProposalMsg {
	return []simtypes.WeightedProposalMsg{
		simulation.NewWeightedProposalMsg(
			opWeightMsgAddArticle,
			defaultWeightMsgAddArticle,
			func(r *rand.Rand, ctx sdk.Context, accs []simtypes.Account) sdk.Msg {
				cointrunksimulation.SimulateMsgAddArticle(am.bankKeeper, am.keeper)
				return nil
			},
		),
		simulation.NewWeightedProposalMsg(
			opWeightMsgPayPublisherRespect,
			defaultWeightMsgPayPublisherRespect,
			func(r *rand.Rand, ctx sdk.Context, accs []simtypes.Account) sdk.Msg {
				cointrunksimulation.SimulateMsgPayPublisherRespect(am.bankKeeper, am.keeper)
				return nil
			},
		),
		simulation.NewWeightedProposalMsg(
			opWeightMsgAcceptDomain,
			defaultWeightMsgAcceptDomain,
			func(r *rand.Rand, ctx sdk.Context, accs []simtypes.Account) sdk.Msg {
				cointrunksimulation.SimulateMsgAcceptDomain(am.bankKeeper, am.keeper)
				return nil
			},
		),
		simulation.NewWeightedProposalMsg(
			opWeightMsgSavePublisher,
			defaultWeightMsgSavePublisher,
			func(r *rand.Rand, ctx sdk.Context, accs []simtypes.Account) sdk.Msg {
				cointrunksimulation.SimulateMsgSavePublisher(am.bankKeeper, am.keeper)
				return nil
			},
		),
		// this line is used by starport scaffolding # simapp/module/OpMsg
	}
}
