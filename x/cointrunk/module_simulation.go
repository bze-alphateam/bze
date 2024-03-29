package cointrunk

import (
	"math/rand"

	"github.com/bze-alphateam/bze/testutil/sample"
	cointrunksimulation "github.com/bze-alphateam/bze/x/cointrunk/simulation"
	"github.com/bze-alphateam/bze/x/cointrunk/types"
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
	_ = cointrunksimulation.FindAccount
	_ = simappparams.StakePerAccount
	_ = simulation.MsgEntryKind
	_ = baseapp.Paramspace
)

const (
	opWeightMsgAddArticle = "op_weight_msg_add_article"
	// TODO: Determine the simulation weight value
	defaultWeightMsgAddArticle int = 100

	opWeightMsgPayPublisherRespect = "op_weight_msg_pay_publisher_respect"
	// TODO: Determine the simulation weight value
	defaultWeightMsgPayPublisherRespect int = 100

	// this line is used by starport scaffolding # simapp/module/const
)

// GenerateGenesisState creates a randomized GenState of the module
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

// ProposalContents doesn't return any content functions for governance proposals
func (AppModule) ProposalContents(_ module.SimulationState) []simtypes.WeightedProposalContent {
	return nil
}

// RandomizedParams creates randomized  param changes for the simulator
func (am AppModule) RandomizedParams(_ *rand.Rand) []simtypes.ParamChange {
	cointrunkParams := types.DefaultParams()
	return []simtypes.ParamChange{
		simulation.NewSimParamChange(types.ModuleName, string(types.KeyAnonArticleLimit), func(r *rand.Rand) string {
			return string(types.Amino.MustMarshalJSON(cointrunkParams.AnonArticleLimit))
		}),
		simulation.NewSimParamChange(types.ModuleName, string(types.KeyAnonArticleCost), func(r *rand.Rand) string {
			return string(types.Amino.MustMarshalJSON(cointrunkParams.AnonArticleCost))
		}),
	}
}

// RegisterStoreDecoder registers a decoder
func (am AppModule) RegisterStoreDecoder(_ sdk.StoreDecoderRegistry) {}

// WeightedOperations returns the all the gov module operations with their respective weights.
func (am AppModule) WeightedOperations(simState module.SimulationState) []simtypes.WeightedOperation {
	operations := make([]simtypes.WeightedOperation, 0)

	var weightMsgAddArticle int
	simState.AppParams.GetOrGenerate(simState.Cdc, opWeightMsgAddArticle, &weightMsgAddArticle, nil,
		func(_ *rand.Rand) {
			weightMsgAddArticle = defaultWeightMsgAddArticle
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgAddArticle,
		cointrunksimulation.SimulateMsgAddArticle(am.accountKeeper, am.bankKeeper, am.keeper),
	))

	var weightMsgPayPublisherRespect int
	simState.AppParams.GetOrGenerate(simState.Cdc, opWeightMsgPayPublisherRespect, &weightMsgPayPublisherRespect, nil,
		func(_ *rand.Rand) {
			weightMsgPayPublisherRespect = defaultWeightMsgPayPublisherRespect
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgPayPublisherRespect,
		cointrunksimulation.SimulateMsgPayPublisherRespect(am.accountKeeper, am.bankKeeper, am.keeper),
	))

	// this line is used by starport scaffolding # simapp/module/operation

	return operations
}
