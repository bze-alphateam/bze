package guardian

import (
	autocliv1 "cosmossdk.io/api/cosmos/autocli/v1"

	modulev1 "github.com/bze-alphateam/bze/api/bze/guardian/v1"
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
			},
		},
		Tx: &autocliv1.ServiceCommandDescriptor{
			Service:              modulev1.Msg_ServiceDesc.ServiceName,
			EnhanceCustomCommand: true, // only required if you want to use the custom command
			RpcCommandOptions: []*autocliv1.RpcCommandOptions{
				{
					RpcMethod:      "UpdateParams",
					Use:            "update-params [params]",
					Short:          "Submit a governance proposal to update the module parameters",
					Long:           "Submit a governance proposal to update the module parameters. All parameters must be supplied, as JSON.",
					Example:        `bzed tx guardian update-params '{"guardian":"bze1..."}' --deposit 1000000ubze --title "..." --summary "..." --from mykey`,
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "params"}},
					GovProposal:    true, // authority gated: wrapped in a x/gov proposal, authority filled with the gov address
				},
				{
					RpcMethod: "TightenRateLimit",
					Use:       "tighten-rate-limit [denom] [channel-id] [max-percent-send] [max-percent-recv]",
					Short:     "Lower the send and receive quotas of an IBC rate limit",
					Long: "Lowers the quotas of an existing rate limit; signed by the guardian or the governance authority. " +
						"A quota can never be raised, the window is unchanged and the flow already counted in the window is kept. 0 stops the direction.",
					Example:        "bzed tx guardian tighten-rate-limit ubze channel-0 1 0 --from guardian",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "denom"}, {ProtoField: "channel_id"}, {ProtoField: "max_percent_send"}, {ProtoField: "max_percent_recv"}},
				},
			},
		},
	}
}
