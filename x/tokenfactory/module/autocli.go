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
				{
					RpcMethod:      "DenomBranding",
					Use:            "denom-branding [denom]",
					Short:          "Query the branding package of a denom",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "denom"}},
				},
				{
					RpcMethod: "AllDenomBranding",
					Use:       "all-denom-branding",
					Short:     "Query the branding packages of all denoms",
				},
				{
					RpcMethod:      "DenomConfig",
					Use:            "denom-config [denom]",
					Short:          "Query the configuration of a denom (community pool, transfer fee, gas surcharge)",
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
					RpcMethod: "SetDenomAdmin",
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
					Use:            "set-denom-metadata [metadata]",
					Short:          "Send a set-denom-metadata tx",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "metadata"}},
				},
				{
					RpcMethod:      "SetDenomBranding",
					Use:            "set-denom-branding [denom]",
					Short:          "Set or clear the branding package of a denom (omit --branding to clear)",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "denom"}},
				},
				{
					RpcMethod:      "DisableMinting",
					Use:            "disable-minting [denom]",
					Short:          "Turn minting off for good on a factory denom (admin only, irreversible)",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "denom"}},
				},
				{
					RpcMethod: "SetDenomConfig",
					Use:       "set-denom-config [denom] [community-pool] [transfer-fee] [gas-surcharge]",
					Short:     "Replace the configuration of a factory denom (admin only)",
					Long: "Replace the whole configuration of a factory denom. Every argument is required. " +
						"Pass \"\" as community-pool for none. Rates are decimals: 0.001 is 0.1%. " +
						"Caps: transfer-fee 0.05, gas-surcharge 2.",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "denom"},
						{ProtoField: "community_pool"},
						{ProtoField: "transfer_fee"},
						{ProtoField: "gas_surcharge"},
					},
				},
				// this line is used by ignite scaffolding # autocli/tx
			},
		},
	}
}
