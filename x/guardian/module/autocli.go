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
				{
					RpcMethod: "BlockedDenoms",
					Use:       "blocked-denoms",
					Short:     "Lists the denoms whose IBC transfers are blocked",
					Long:      "Lists the denoms on the rate limit module's blacklist. Every IBC transfer of these denoms is refused, in both directions, on every channel.",
				},
				{
					RpcMethod: "ClosedChannels",
					Use:       "closed-channels",
					Short:     "Lists the transfer channels whose IBC transfers are stopped",
					Long:      "Lists the transfer channels closed by the guardian or governance. Every ICS-20 transfer on them is refused, in both directions; the channels themselves stay open in IBC.",
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
				{
					RpcMethod: "BlockDenom",
					Use:       "block-denom [denom]",
					Short:     "Stop every IBC transfer of a denom, on every channel",
					Long: "Puts the denom on the rate limit blacklist: every ICS-20 transfer of it is refused in both directions on every channel, " +
						"whether or not it has a rate limit. The denom must exist on chain, spelled exactly as stored (ibc/ followed by the upper-case hash for an IBC token). " +
						"Signed by the guardian or the governance authority; only a governance proposal can lift it.",
					Example:        "bzed tx guardian block-denom ibc/ABC... --from guardian",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "denom"}},
				},
				{
					RpcMethod:      "UnblockDenom",
					Use:            "unblock-denom [denom]",
					Short:          "Submit a governance proposal to lift the IBC block of a denom",
					Example:        `bzed tx guardian unblock-denom ibc/ABC... --deposit 1000000ubze --title "..." --summary "..." --from mykey`,
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "denom"}},
					GovProposal:    true, // authority gated: wrapped in a x/gov proposal, authority filled with the gov address
				},
				{
					RpcMethod: "CloseChannel",
					Use:       "close-channel [channel-id]",
					Short:     "Stop every IBC transfer on a transfer channel, in both directions",
					Long: "Closes the transfer channel for ICS-20 transfers: sends on it fail and received transfers are answered with an error acknowledgement, " +
						"so the sending chain refunds them. Acknowledgements and timeouts of transfers already in flight still go through. " +
						"The channel stays open in IBC. Signed by the guardian or the governance authority; only a governance proposal can reopen it.",
					Example:        "bzed tx guardian close-channel channel-0 --from guardian",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "channel_id"}},
				},
				{
					RpcMethod:      "ReopenChannel",
					Use:            "reopen-channel [channel-id]",
					Short:          "Submit a governance proposal to reopen a closed transfer channel",
					Example:        `bzed tx guardian reopen-channel channel-0 --deposit 1000000ubze --title "..." --summary "..." --from mykey`,
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "channel_id"}},
					GovProposal:    true, // authority gated: wrapped in a x/gov proposal, authority filled with the gov address
				},
			},
		},
	}
}
