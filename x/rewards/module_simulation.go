package rewards

import (
	"math/rand"

	"github.com/bze-alphateam/bze/testutil/sample"
	rewardssimulation "github.com/bze-alphateam/bze/x/rewards/simulation"
	"github.com/bze-alphateam/bze/x/rewards/types"
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
	_ = rewardssimulation.FindAccount
	_ = simappparams.StakePerAccount
	_ = simulation.MsgEntryKind
	_ = baseapp.Paramspace
)

const (
	opWeightMsgCreateStakingReward = "op_weight_msg_staking_reward"
	// TODO: Determine the simulation weight value
	defaultWeightMsgCreateStakingReward int = 100

	opWeightMsgUpdateStakingReward = "op_weight_msg_staking_reward"
	// TODO: Determine the simulation weight value
	defaultWeightMsgUpdateStakingReward int = 100

	opWeightMsgDeleteStakingReward = "op_weight_msg_staking_reward"
	// TODO: Determine the simulation weight value
	defaultWeightMsgDeleteStakingReward int = 100

	// this line is used by starport scaffolding # simapp/module/const
)

// GenerateGenesisState creates a randomized GenState of the module
func (AppModule) GenerateGenesisState(simState *module.SimulationState) {
	accs := make([]string, len(simState.Accounts))
	for i, acc := range simState.Accounts {
		accs[i] = acc.Address.String()
	}
	rewardsGenesis := types.GenesisState{
		Params: types.DefaultParams(),
		StakingRewardList: []types.StakingReward{
			{
				RewardId: "0",
			},
			{
				RewardId: "1",
			},
		},
		// this line is used by starport scaffolding # simapp/module/genesisState
	}
	simState.GenState[types.ModuleName] = simState.Cdc.MustMarshalJSON(&rewardsGenesis)
}

// ProposalContents doesn't return any content functions for governance proposals
func (AppModule) ProposalContents(_ module.SimulationState) []simtypes.WeightedProposalContent {
	return nil
}

// RandomizedParams creates randomized  param changes for the simulator
func (am AppModule) RandomizedParams(_ *rand.Rand) []simtypes.ParamChange {
	rewardsParams := types.DefaultParams()
	return []simtypes.ParamChange{
		simulation.NewSimParamChange(types.ModuleName, string(types.KeyCreateStakingRewardFee), func(r *rand.Rand) string {
			return string(types.Amino.MustMarshalJSON(rewardsParams.CreateStakingRewardFee))
		}),
		simulation.NewSimParamChange(types.ModuleName, string(types.KeyCreateTradingRewardFee), func(r *rand.Rand) string {
			return string(types.Amino.MustMarshalJSON(rewardsParams.CreateTradingRewardFee))
		}),
	}
}

// RegisterStoreDecoder registers a decoder
func (am AppModule) RegisterStoreDecoder(_ sdk.StoreDecoderRegistry) {}

// WeightedOperations returns the all the gov module operations with their respective weights.
func (am AppModule) WeightedOperations(simState module.SimulationState) []simtypes.WeightedOperation {
	operations := make([]simtypes.WeightedOperation, 0)

	var weightMsgCreateStakingReward int
	simState.AppParams.GetOrGenerate(simState.Cdc, opWeightMsgCreateStakingReward, &weightMsgCreateStakingReward, nil,
		func(_ *rand.Rand) {
			weightMsgCreateStakingReward = defaultWeightMsgCreateStakingReward
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgCreateStakingReward,
		rewardssimulation.SimulateMsgCreateStakingReward(am.accountKeeper, am.bankKeeper, am.keeper),
	))

	var weightMsgUpdateStakingReward int
	simState.AppParams.GetOrGenerate(simState.Cdc, opWeightMsgUpdateStakingReward, &weightMsgUpdateStakingReward, nil,
		func(_ *rand.Rand) {
			weightMsgUpdateStakingReward = defaultWeightMsgUpdateStakingReward
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgUpdateStakingReward,
		rewardssimulation.SimulateMsgUpdateStakingReward(am.accountKeeper, am.bankKeeper, am.keeper),
	))

	var weightMsgDeleteStakingReward int
	simState.AppParams.GetOrGenerate(simState.Cdc, opWeightMsgDeleteStakingReward, &weightMsgDeleteStakingReward, nil,
		func(_ *rand.Rand) {
			weightMsgDeleteStakingReward = defaultWeightMsgDeleteStakingReward
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgDeleteStakingReward,
		rewardssimulation.SimulateMsgDeleteStakingReward(am.accountKeeper, am.bankKeeper, am.keeper),
	))

	// this line is used by starport scaffolding # simapp/module/operation

	return operations
}
