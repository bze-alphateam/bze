package rewards

import (
	autocliv1 "cosmossdk.io/api/cosmos/autocli/v1"

	modulev1 "github.com/bze-alphateam/bze/api/bze/rewards"
)

// AutoCLIOptions implements the autocli.HasAutoCLIConfig interface.
func (am AppModule) AutoCLIOptions() *autocliv1.ModuleOptions {
	return &autocliv1.ModuleOptions{
		Query: &autocliv1.ServiceCommandDescriptor{
			Service: modulev1.Query_ServiceDesc.ServiceName,
			RpcCommandOptions: []*autocliv1.RpcCommandOptions{
				{
					RpcMethod: "Params",
					Use:       "params",
					Short:     "Shows the parameters of the module",
				},
				{
					RpcMethod:      "StakingReward",
					Use:            "get-staking-reward [reward-id]",
					Short:          "Query get-staking-reward",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "reward_id"}},
				},
				{
					RpcMethod:      "AllStakingRewards",
					Use:            "all-staking-rewards",
					Short:          "Query all-staking-rewards",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{},
				},
				{
					RpcMethod:      "TradingReward",
					Use:            "trading-reward [reward-id]",
					Short:          "Query trading-reward",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "reward_id"}},
				},
				{
					RpcMethod:      "AllTradingRewards",
					Use:            "all-trading-rewards",
					Short:          "Query all-trading-rewards",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{},
				},
				{
					RpcMethod:      "StakingRewardParticipant",
					Use:            "staking-reward-participant [address]",
					Short:          "Query staking-reward-participant",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "address"}},
				},
				{
					RpcMethod:      "AllStakingRewardParticipants",
					Use:            "all-staking-reward-participants",
					Short:          "Query all-staking-reward-participants",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{},
				},
				{
					RpcMethod:      "TradingRewardLeaderboard",
					Use:            "trading-reward-leaderboard [reward-id]",
					Short:          "Query trading-reward-leaderboard",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "reward_id"}},
				},
				{
					RpcMethod:      "MarketTradingReward",
					Use:            "market-trading-reward [market-id]",
					Short:          "Query market-trading-reward",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "market_id"}},
				},

				{
					RpcMethod: "AllPendingUnlockParticipants",
					Use: "all-pending-unlock-participants",
					Short: "Query all-pending-unlock-participants",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{},
				},

				// this line is used by ignite scaffolding # autocli/query
			},
		},
		Tx: &autocliv1.ServiceCommandDescriptor{
			Service:              modulev1.Msg_ServiceDesc.ServiceName,
			EnhanceCustomCommand: true, // only required if you want to use the custom command
			RpcCommandOptions: []*autocliv1.RpcCommandOptions{
				{
					RpcMethod: "UpdateParams",
					Skip:      true, // skipped because authority gated
				},
				{
					RpcMethod:      "CreateStakingReward",
					Use:            "create-staking-reward [prize-amount] [prize-denom] [staking-denom] [duration] [min-stake] [lock]",
					Short:          "Send a create-staking-reward tx",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "prize_amount"}, {ProtoField: "prize_denom"}, {ProtoField: "staking_denom"}, {ProtoField: "duration"}, {ProtoField: "min_stake"}, {ProtoField: "lock"}},
				},
				{
					RpcMethod:      "UpdateStakingReward",
					Use:            "update-staking-reward [reward-id] [duration]",
					Short:          "Send a update-staking-reward tx",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "reward_id"}, {ProtoField: "duration"}},
				},
				{
					RpcMethod:      "JoinStaking",
					Use:            "join-staking [reward-id] [amount]",
					Short:          "Send a join-staking tx",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "reward_id"}, {ProtoField: "amount"}},
				},
				{
					RpcMethod:      "ExitStaking",
					Use:            "exit-staking [reward-id]",
					Short:          "Send a exit-staking tx",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "reward_id"}},
				},
				{
					RpcMethod:      "ClaimStakingRewards",
					Use:            "claim-staking-rewards [reward-id]",
					Short:          "Send a claim-staking-rewards tx",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "reward_id"}},
				},
				{
					RpcMethod:      "DistributeStakingRewards",
					Use:            "distribute-staking-rewards [reward-id] [amount]",
					Short:          "Send a distribute-staking-rewards tx",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "reward_id"}, {ProtoField: "amount"}},
				},
				{
					RpcMethod:      "CreateTradingReward",
					Use:            "create-trading-reward [prize-amount] [prize-denom] [duration] [market-id] [slots]",
					Short:          "Send a create-trading-reward tx",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "prize_amount"}, {ProtoField: "prize_denom"}, {ProtoField: "duration"}, {ProtoField: "market_id"}, {ProtoField: "slots"}},
				},
				// this line is used by ignite scaffolding # autocli/tx
			},
		},
	}
}
