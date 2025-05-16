package rewards

import (
	"math/rand"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	"github.com/cosmos/cosmos-sdk/x/simulation"

	"github.com/bze-alphateam/bze/testutil/sample"
	rewardssimulation "github.com/bze-alphateam/bze/x/rewards/simulation"
	"github.com/bze-alphateam/bze/x/rewards/types"
)

// avoid unused import issue
var (
	_ = rewardssimulation.FindAccount
	_ = rand.Rand{}
	_ = sample.AccAddress
	_ = sdk.AccAddress{}
	_ = simulation.MsgEntryKind
)

const (
	opWeightMsgCreateStakingReward = "op_weight_msg_create_staking_reward"
	// TODO: Determine the simulation weight value
	defaultWeightMsgCreateStakingReward int = 100

	opWeightMsgUpdateStakingReward = "op_weight_msg_update_staking_reward"
	// TODO: Determine the simulation weight value
	defaultWeightMsgUpdateStakingReward int = 100

	opWeightMsgJoinStaking = "op_weight_msg_join_staking"
	// TODO: Determine the simulation weight value
	defaultWeightMsgJoinStaking int = 100

	opWeightMsgExitStaking = "op_weight_msg_exit_staking"
	// TODO: Determine the simulation weight value
	defaultWeightMsgExitStaking int = 100

	opWeightMsgClaimStakingRewards = "op_weight_msg_claim_staking_rewards"
	// TODO: Determine the simulation weight value
	defaultWeightMsgClaimStakingRewards int = 100

	opWeightMsgDistributeStakingRewards = "op_weight_msg_distribute_staking_rewards"
	// TODO: Determine the simulation weight value
	defaultWeightMsgDistributeStakingRewards int = 100

	opWeightMsgCreateTradingReward = "op_weight_msg_create_trading_reward"
	// TODO: Determine the simulation weight value
	defaultWeightMsgCreateTradingReward int = 100

	opWeightMsgActivateTradingReward = "op_weight_msg_activate_trading_reward"
	// TODO: Determine the simulation weight value
	defaultWeightMsgActivateTradingReward int = 100

	// this line is used by starport scaffolding # simapp/module/const
)

// GenerateGenesisState creates a randomized GenState of the module.
func (AppModule) GenerateGenesisState(simState *module.SimulationState) {
	accs := make([]string, len(simState.Accounts))
	for i, acc := range simState.Accounts {
		accs[i] = acc.Address.String()
	}
	rewardsGenesis := types.GenesisState{
		Params: types.DefaultParams(),
		// this line is used by starport scaffolding # simapp/module/genesisState
	}
	simState.GenState[types.ModuleName] = simState.Cdc.MustMarshalJSON(&rewardsGenesis)
}

// RegisterStoreDecoder registers a decoder.
func (am AppModule) RegisterStoreDecoder(_ simtypes.StoreDecoderRegistry) {}

// WeightedOperations returns the all the gov module operations with their respective weights.
func (am AppModule) WeightedOperations(simState module.SimulationState) []simtypes.WeightedOperation {
	operations := make([]simtypes.WeightedOperation, 0)

	var weightMsgCreateStakingReward int
	simState.AppParams.GetOrGenerate(opWeightMsgCreateStakingReward, &weightMsgCreateStakingReward, nil,
		func(_ *rand.Rand) {
			weightMsgCreateStakingReward = defaultWeightMsgCreateStakingReward
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgCreateStakingReward,
		rewardssimulation.SimulateMsgCreateStakingReward(am.accountKeeper, am.bankKeeper, am.keeper),
	))

	var weightMsgUpdateStakingReward int
	simState.AppParams.GetOrGenerate(opWeightMsgUpdateStakingReward, &weightMsgUpdateStakingReward, nil,
		func(_ *rand.Rand) {
			weightMsgUpdateStakingReward = defaultWeightMsgUpdateStakingReward
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgUpdateStakingReward,
		rewardssimulation.SimulateMsgUpdateStakingReward(am.accountKeeper, am.bankKeeper, am.keeper),
	))

	var weightMsgJoinStaking int
	simState.AppParams.GetOrGenerate(opWeightMsgJoinStaking, &weightMsgJoinStaking, nil,
		func(_ *rand.Rand) {
			weightMsgJoinStaking = defaultWeightMsgJoinStaking
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgJoinStaking,
		rewardssimulation.SimulateMsgJoinStaking(am.accountKeeper, am.bankKeeper, am.keeper),
	))

	var weightMsgExitStaking int
	simState.AppParams.GetOrGenerate(opWeightMsgExitStaking, &weightMsgExitStaking, nil,
		func(_ *rand.Rand) {
			weightMsgExitStaking = defaultWeightMsgExitStaking
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgExitStaking,
		rewardssimulation.SimulateMsgExitStaking(am.accountKeeper, am.bankKeeper, am.keeper),
	))

	var weightMsgClaimStakingRewards int
	simState.AppParams.GetOrGenerate(opWeightMsgClaimStakingRewards, &weightMsgClaimStakingRewards, nil,
		func(_ *rand.Rand) {
			weightMsgClaimStakingRewards = defaultWeightMsgClaimStakingRewards
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgClaimStakingRewards,
		rewardssimulation.SimulateMsgClaimStakingRewards(am.accountKeeper, am.bankKeeper, am.keeper),
	))

	var weightMsgDistributeStakingRewards int
	simState.AppParams.GetOrGenerate(opWeightMsgDistributeStakingRewards, &weightMsgDistributeStakingRewards, nil,
		func(_ *rand.Rand) {
			weightMsgDistributeStakingRewards = defaultWeightMsgDistributeStakingRewards
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgDistributeStakingRewards,
		rewardssimulation.SimulateMsgDistributeStakingRewards(am.accountKeeper, am.bankKeeper, am.keeper),
	))

	var weightMsgCreateTradingReward int
	simState.AppParams.GetOrGenerate(opWeightMsgCreateTradingReward, &weightMsgCreateTradingReward, nil,
		func(_ *rand.Rand) {
			weightMsgCreateTradingReward = defaultWeightMsgCreateTradingReward
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgCreateTradingReward,
		rewardssimulation.SimulateMsgCreateTradingReward(am.accountKeeper, am.bankKeeper, am.keeper),
	))

	var weightMsgActivateTradingReward int
	simState.AppParams.GetOrGenerate(opWeightMsgActivateTradingReward, &weightMsgActivateTradingReward, nil,
		func(_ *rand.Rand) {
			weightMsgActivateTradingReward = defaultWeightMsgActivateTradingReward
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgActivateTradingReward,
		rewardssimulation.SimulateMsgActivateTradingReward(am.accountKeeper, am.bankKeeper, am.keeper),
	))

	// this line is used by starport scaffolding # simapp/module/operation

	return operations
}

// ProposalMsgs returns msgs used for governance proposals for simulations.
func (am AppModule) ProposalMsgs(simState module.SimulationState) []simtypes.WeightedProposalMsg {
	return []simtypes.WeightedProposalMsg{
		simulation.NewWeightedProposalMsg(
			opWeightMsgCreateStakingReward,
			defaultWeightMsgCreateStakingReward,
			func(r *rand.Rand, ctx sdk.Context, accs []simtypes.Account) sdk.Msg {
				rewardssimulation.SimulateMsgCreateStakingReward(am.accountKeeper, am.bankKeeper, am.keeper)
				return nil
			},
		),
		simulation.NewWeightedProposalMsg(
			opWeightMsgUpdateStakingReward,
			defaultWeightMsgUpdateStakingReward,
			func(r *rand.Rand, ctx sdk.Context, accs []simtypes.Account) sdk.Msg {
				rewardssimulation.SimulateMsgUpdateStakingReward(am.accountKeeper, am.bankKeeper, am.keeper)
				return nil
			},
		),
		simulation.NewWeightedProposalMsg(
			opWeightMsgJoinStaking,
			defaultWeightMsgJoinStaking,
			func(r *rand.Rand, ctx sdk.Context, accs []simtypes.Account) sdk.Msg {
				rewardssimulation.SimulateMsgJoinStaking(am.accountKeeper, am.bankKeeper, am.keeper)
				return nil
			},
		),
		simulation.NewWeightedProposalMsg(
			opWeightMsgExitStaking,
			defaultWeightMsgExitStaking,
			func(r *rand.Rand, ctx sdk.Context, accs []simtypes.Account) sdk.Msg {
				rewardssimulation.SimulateMsgExitStaking(am.accountKeeper, am.bankKeeper, am.keeper)
				return nil
			},
		),
		simulation.NewWeightedProposalMsg(
			opWeightMsgClaimStakingRewards,
			defaultWeightMsgClaimStakingRewards,
			func(r *rand.Rand, ctx sdk.Context, accs []simtypes.Account) sdk.Msg {
				rewardssimulation.SimulateMsgClaimStakingRewards(am.accountKeeper, am.bankKeeper, am.keeper)
				return nil
			},
		),
		simulation.NewWeightedProposalMsg(
			opWeightMsgDistributeStakingRewards,
			defaultWeightMsgDistributeStakingRewards,
			func(r *rand.Rand, ctx sdk.Context, accs []simtypes.Account) sdk.Msg {
				rewardssimulation.SimulateMsgDistributeStakingRewards(am.accountKeeper, am.bankKeeper, am.keeper)
				return nil
			},
		),
		simulation.NewWeightedProposalMsg(
			opWeightMsgCreateTradingReward,
			defaultWeightMsgCreateTradingReward,
			func(r *rand.Rand, ctx sdk.Context, accs []simtypes.Account) sdk.Msg {
				rewardssimulation.SimulateMsgCreateTradingReward(am.accountKeeper, am.bankKeeper, am.keeper)
				return nil
			},
		),
		simulation.NewWeightedProposalMsg(
			opWeightMsgActivateTradingReward,
			defaultWeightMsgActivateTradingReward,
			func(r *rand.Rand, ctx sdk.Context, accs []simtypes.Account) sdk.Msg {
				rewardssimulation.SimulateMsgActivateTradingReward(am.accountKeeper, am.bankKeeper, am.keeper)
				return nil
			},
		),

		// this line is used by starport scaffolding # simapp/module/OpMsg
	}
}
