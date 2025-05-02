package tokenfactory

import (
	autocliv1 "cosmossdk.io/api/cosmos/autocli/v1"

	modulev1 "github.com/bze-alphateam/bze/api/bze/tokenfactory"
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
					RpcMethod:      "DenomAuthority",
					Use:            "denom-authority [denom]",
					Short:          "Query denom-authority",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "denom"}},
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
					RpcMethod:      "CreateDenom",
					Use:            "create-denom [subdenom]",
					Short:          "Send a create-denom tx",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "subdenom"}},
				},
				{
					RpcMethod:      "Mint",
					Use:            "mint [coins]",
					Short:          "Send a mint tx",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "coins"}},
				},
				{
					RpcMethod:      "Burn",
					Use:            "burn [coins]",
					Short:          "Send a burn tx",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "coins"}},
				},
				{
					RpcMethod:      "ChangeAdmin",
					Use:            "change-admin [denom] [new-admin]",
					Short:          "Send a change-admin tx",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "denom"}, {ProtoField: "newAdmin"}},
				},
				{
					RpcMethod:      "SetDenomMetadata",
					Use:            "set-denom-metadata [metadada]",
					Short:          "Send a set-denom-metadata tx",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "metadada"}},
				},
				// this line is used by ignite scaffolding # autocli/tx
			},
		},
	}
}
