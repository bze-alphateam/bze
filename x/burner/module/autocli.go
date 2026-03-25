package burner

import (
	autocliv1 "cosmossdk.io/api/cosmos/autocli/v1"

	modulev1 "github.com/bze-alphateam/bze/api/bze/burner"
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
					RpcMethod: "Raffles",
					Use:       "raffles",
					Short:     "Query running raffles",
				},
				{
					RpcMethod:      "RaffleWinners",
					Use:            "raffle-winners [denom]",
					Short:          "Query raffle winners by denomination",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "denom"}},
				},
				{
					RpcMethod: "AllBurnedCoins",
					Use:       "all-burned-coins",
					Short:     "Query all coins burnings",
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
					RpcMethod:      "FundBurner",
					Use:            "fund-burner [amount]",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "amount"}},
				},
				{
					RpcMethod: "StartRaffle",
					Use:       "start-raffle [pot] [duration] [chances] [ratio] [ticket-price] [denom]",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "pot"},
						{ProtoField: "duration"},
						{ProtoField: "chances"},
						{ProtoField: "ratio"},
						{ProtoField: "ticket_price"},
						{ProtoField: "denom"},
					},
				},
				{
					RpcMethod:      "JoinRaffle",
					Use:            "join-raffle [denom] [tickets]",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "denom"}, {ProtoField: "tickets"}},
				},
				{
					RpcMethod:      "MoveIbcLockedCoins",
					Use:            "move-ibc-locked-coins [denom]",
					Short:          "Move IBC locked coins to liquidity and send native refund to burner",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "denom"}},
				},
				// this line is used by ignite scaffolding # autocli/tx
			},
		},
	}
}
