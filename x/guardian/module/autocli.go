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
					RpcMethod: "AutomationPaused",
					Use:       "automation-paused",
					Short:     "Shows whether the protocol automation is paused",
					Long:      "Shows whether the txfeecollector EndBlock conversions and the burner's periodic burn are paused. The paused field is printed as false when the automation runs.",
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
					RpcMethod: "DisableMessages",
					Use:       "disable-messages [msg-type-urls...]",
					Short:     "Disable message types through the x/circuit circuit breaker",
					Long: "Adds the message types to the x/circuit disable list: the chain refuses every message of these types, sent in a tx or executed by another module. " +
						"Signed by the guardian or the governance authority. Governance, circuit, upgrade and guardian messages, and the x/dao submit/vote/settle messages, cannot be disabled. " +
						"Only a governance proposal with x/circuit's MsgResetCircuitBreaker re-enables a message; 'bzed q circuit disabled-list' shows the list.",
					Example:        "bzed tx guardian disable-messages /bze.tradebin.MsgMultiSwap --from guardian",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "msg_type_urls", Varargs: true}},
				},
				{
					RpcMethod: "HaltTradingDenoms",
					Use:       "halt-trading-denoms [denom]...",
					Short:     "Halt denoms on the DEX",
					Long: "Halts denoms on the DEX with the same effect as the tradebin governance message: no new order, fill, market, pool, " +
						"liquidity add or swap involving them and no tx fee in them; cancelling orders, removing liquidity and transfers keep working. " +
						"The native denom cannot be halted and a denom already halted is left as it is. Signed by the guardian or the governance authority; " +
						"only a governance proposal with tradebin's MsgUnhaltDenoms lifts the halt. List the halted denoms with: bzed query tradebin halted-denoms",
					Example:        "bzed tx guardian halt-trading-denoms factory/bze1.../utoken ibc/ABC... --from guardian",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "denoms", Varargs: true}},
				},
				{
					RpcMethod: "HaltChain",
					Use:       "halt-chain [reason]",
					Short:     "Stop the chain at the next block (last resort)",
					Long: "Schedules an upgrade plan named emergency-halt-<height> at the next block height; every node stops there with UPGRADE NEEDED. " +
						"It replaces any plan already scheduled and cannot be undone by a message: only a binary with an upgrade handler for the plan, " +
						"or all validators restarting with --unsafe-skip-upgrades <height>, restarts the chain. Signed by the guardian or the governance authority. " +
						"The reason (1 to 256 bytes) is emitted in the event; the plan info is the fixed text \"emergency halt\".",
					Example:        `bzed tx guardian halt-chain "incident under investigation" --from guardian`,
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "reason"}},
				},
				{
					RpcMethod: "PauseAutomation",
					Use:       "pause-automation",
					Short:     "Pause the txfeecollector EndBlock conversions and the burner's periodic burn",
					Long: "Stops the protocol's own automatic fund movements: the txfeecollector EndBlock conversions (gas fees and community pool fees) " +
						"and the burner's periodic burn. The coins stay where they are. Signed by the guardian or the governance authority; " +
						"only a governance proposal can resume the automation.",
					Example: "bzed tx guardian pause-automation --from guardian",
				},
				{
					RpcMethod:   "ResumeAutomation",
					Use:         "resume-automation",
					Short:       "Submit a governance proposal to resume the protocol automation",
					Example:     `bzed tx guardian resume-automation --deposit 1000000ubze --title "..." --summary "..." --from mykey`,
					GovProposal: true, // authority gated: wrapped in a x/gov proposal, authority filled with the gov address
				},
			},
		},
	}
}
