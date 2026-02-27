package burner

import (
	"math/rand"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	"github.com/cosmos/cosmos-sdk/x/simulation"

	"github.com/bze-alphateam/bze/testutil/sample"
	burnersimulation "github.com/bze-alphateam/bze/x/burner/simulation"
	"github.com/bze-alphateam/bze/x/burner/types"
)

// avoid unused import issue
var (
	_ = burnersimulation.FindAccount
	_ = rand.Rand{}
	_ = sample.AccAddress
	_ = sdk.AccAddress{}
	_ = simulation.MsgEntryKind
)

const (
	opWeightMsgMoveIbcLockedCoins = "op_weight_msg_burn_locked_coins"
	// TODO: Determine the simulation weight value
	defaultWeightMsgMoveIbcLockedCoins int = 100

	// this line is used by starport scaffolding # simapp/module/const
)

// GenerateGenesisState creates a randomized GenState of the module.
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

// RegisterStoreDecoder registers a decoder.
func (am AppModule) RegisterStoreDecoder(_ simtypes.StoreDecoderRegistry) {}

// WeightedOperations returns the all the gov module operations with their respective weights.
func (am AppModule) WeightedOperations(simState module.SimulationState) []simtypes.WeightedOperation {
	operations := make([]simtypes.WeightedOperation, 0)

	var weightMsgMoveIbcLockedCoins int
	simState.AppParams.GetOrGenerate(opWeightMsgMoveIbcLockedCoins, &weightMsgMoveIbcLockedCoins, nil,
		func(_ *rand.Rand) {
			weightMsgMoveIbcLockedCoins = defaultWeightMsgMoveIbcLockedCoins
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgMoveIbcLockedCoins,
		burnersimulation.SimulateMsgMoveIbcLockedCoins(am.accountKeeper, am.bankKeeper, am.keeper),
	))

	// this line is used by starport scaffolding # simapp/module/operation

	return operations
}

// ProposalMsgs returns msgs used for governance proposals for simulations.
func (am AppModule) ProposalMsgs(simState module.SimulationState) []simtypes.WeightedProposalMsg {
	return []simtypes.WeightedProposalMsg{
		simulation.NewWeightedProposalMsg(
			opWeightMsgMoveIbcLockedCoins,
			defaultWeightMsgMoveIbcLockedCoins,
			func(r *rand.Rand, ctx sdk.Context, accs []simtypes.Account) sdk.Msg {
				burnersimulation.SimulateMsgMoveIbcLockedCoins(am.accountKeeper, am.bankKeeper, am.keeper)
				return nil
			},
		),
		// this line is used by starport scaffolding # simapp/module/OpMsg
	}
}
