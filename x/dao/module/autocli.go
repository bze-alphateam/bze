package dao

import (
	autocliv1 "cosmossdk.io/api/cosmos/autocli/v1"

	modulev1 "github.com/bze-alphateam/bze/api/bze/dao"
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
					RpcMethod:      "Dao",
					Use:            "dao [id]",
					Short:          "Shows a DAO by its id",
					Example:        "bzed q dao dao 1",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "id"}},
				},
				{
					RpcMethod:      "DaoByAddress",
					Use:            "dao-by-address [address]",
					Short:          "Shows the DAO that owns an account",
					Example:        "bzed q dao dao-by-address bze1...",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "address"}},
				},
				{
					RpcMethod: "DaoController",
					Use:       "dao-controller [dao-id]",
					Short:     "Shows who ultimately controls a DAO by following its admin chain",
					Long: "Follows the DAO's admin, and the admin's admin while it is a DAO, and prints where the chain ends: " +
						"an account (ACCOUNT), a self-owned DAO (SELF_OWNED), a DAO already visited (CYCLE) or the 16th DAO (DEPTH_CAP), " +
						"with the DAOs visited on the way. Pending nominations are ignored.",
					Example:        "bzed q dao dao-controller 1",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "dao_id"}},
				},
				{
					RpcMethod: "Proposal",
					Use:       "proposal [proposal-id]",
					Short:     "Shows a proposal with its live status",
					Long: "Shows the proposal record and, at the current block: voting_ended (it is VOTING past its voting end, so it can be settled), " +
						"retry_deadline (the end of the retry window of an EXECUTION_FAILED proposal, unix seconds, else 0) and " +
						"expedite_settleable (an expedited proposal whose tally already meets the expedite bars before its voting end).",
					Example:        "bzed q dao proposal 1",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "id"}},
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
					Example:        `bzed tx dao update-params '{}' --deposit 1000000ubze --title "..." --summary "..." --from mykey`,
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "params"}},
					GovProposal:    true, // authority gated: wrapped in a x/gov proposal, authority filled with the gov address
				},
				{
					RpcMethod: "CreateDao",
					Use:       "create-dao [type] [name] [config]",
					Short:     "Create a DAO; the creator pays the creation fee",
					Long: "Create a DAO with its own derived account. type is static-list or stake, " +
						"config is the DaoConfig as JSON: ratios as 18-decimal integers (500000000000000000 = 0.5), periods as seconds (\"259200s\"). A static-list DAO needs --members; a stake DAO takes --electorate '{\"sr_ids\":[\"3\"],\"dr_denom\":\"ubze\"}'. Without --admin the DAO is self-owned and needs a live electorate.",
					Example: `bzed tx dao create-dao static-list "My DAO" '{"min_quorum":"500000000000000000","threshold":"500000000000000000","voting_period":"259200s",` +
						`"expedited_period":"86400s","expedite_quorum":"600000000000000000","expedite_threshold":"660000000000000000"}' ` +
						`--members bze1...,bze1... --description "..." --from mykey`,
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "type"}, {ProtoField: "name"}, {ProtoField: "config"}},
					FlagOptions: map[string]*autocliv1.FlagOptions{
						"admin":       {Usage: "admin address (a person or another DAO); empty = self-owned"},
						"description": {Usage: "DAO description, at most 2000 bytes"},
						"members":     {Usage: "comma-separated member addresses of a static-list DAO"},
					},
				},
				{
					RpcMethod: "AdminExecute",
					Use:       "admin-execute [dao-id] [msgs...]",
					Short:     "Run messages as a DAO; only its external admin can",
					Long: "Run messages as the DAO, in order; if one fails, none is applied. Every message must be signed by the DAO account only, " +
						"so it can spend the treasury but never another address. Each message is a JSON Any (with \"@type\") or a .json file holding one. " +
						"A self-owned DAO has no admin and executes through its proposals.",
					Example:        `bzed tx dao admin-execute 1 '{"@type":"/cosmos.bank.v1beta1.MsgSend","from_address":"bze1<dao>","to_address":"bze1...","amount":[{"denom":"ubze","amount":"1000000"}]}' --from admin`,
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "dao_id"}, {ProtoField: "msgs", Varargs: true}},
				},
				{
					RpcMethod: "NominateAdmin",
					Use:       "nominate-admin [dao-id] [nominee]",
					Short:     "Nominate the next admin of a DAO, or hand it to its voters",
					Long: "First step of the admin handover: the DAO's current admin nominates a successor, who becomes admin once they accept. " +
						"Only one nomination can be pending. Pass \"\" as the nominee to make the DAO self-owned at once; its electorate must be able to vote.",
					Example:        `bzed tx dao nominate-admin 1 bze1... --from admin` + "\n" + `bzed tx dao nominate-admin 1 "" --from admin`,
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "dao_id"}, {ProtoField: "nominee"}},
				},
				{
					RpcMethod:      "AcceptAdminNomination",
					Use:            "accept-admin-nomination [dao-id]",
					Short:          "Accept a DAO's admin nomination and become its admin",
					Long:           "Second step of the admin handover: the pending nominee signs and becomes the DAO's admin. A DAO nominee accepts through its own proposal or admin.",
					Example:        `bzed tx dao accept-admin-nomination 1 --from nominee`,
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "dao_id"}},
				},
				{
					RpcMethod:      "WithdrawAdminNomination",
					Use:            "withdraw-admin-nomination [dao-id]",
					Short:          "Withdraw a DAO's pending admin nomination",
					Long:           "The current admin cancels a pending handover before the nominee accepts it; the admin stays in charge and can nominate again.",
					Example:        `bzed tx dao withdraw-admin-nomination 1 --from admin`,
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "dao_id"}},
				},
				{
					RpcMethod: "SubmitProposal",
					Use:       "submit-proposal [dao-id] [title] [msgs...]",
					Short:     "Submit a DAO proposal; it runs its messages as the DAO if it passes",
					Long: "Submit a proposal of zero or more messages. Each message is a JSON Any (with \"@type\") or a .json file holding one, " +
						"and must be signed by the DAO account only. Members (or stakers) of the DAO can submit; anyone can when the DAO has open submission. " +
						"The DAO's proposal cost is charged and its deposit escrowed; --expedited asks for the shorter voting period when the DAO allows it.",
					Example:        `bzed tx dao submit-proposal 1 "Fund tooling" msg.json --summary "Pays 1 BZE to the tooling team" --from member`,
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "dao_id"}, {ProtoField: "title"}, {ProtoField: "msgs", Varargs: true}},
					FlagOptions: map[string]*autocliv1.FlagOptions{
						"summary":   {Usage: "proposal summary"},
						"expedited": {Usage: "use the DAO's expedited voting period"},
					},
				},
				{
					RpcMethod: "Vote",
					Use:       "vote [proposal-id] [option]",
					Short:     "Vote yes, no or abstain on a DAO proposal in voting",
					Long: "Vote on a proposal with your current voting power: one vote per member of a static-list DAO, " +
						"or your stake in the proposal's sources that joined before it was submitted. Voting again with another option replaces your ballot. " +
						"The option is yes, no or abstain, in lowercase.",
					Example:        "bzed tx dao vote 1 yes --from member",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "proposal_id"}, {ProtoField: "option"}},
				},
				{
					RpcMethod: "CancelProposal",
					Use:       "cancel-proposal [proposal-id]",
					Short:     "Cancel your own DAO proposal while it is in voting",
					Long: "The proposer withdraws a proposal in voting, even after its voting end until it is settled. " +
						"The deposit is refunded only when the DAO's deposit refund policy is ALWAYS; otherwise it is burned or sent to the DAO's treasury.",
					Example:        "bzed tx dao cancel-proposal 1 --from proposer",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "proposal_id"}},
				},
				{
					RpcMethod: "SettleProposal",
					Use:       "settle-proposal [proposal-id]",
					Short:     "Settle a DAO proposal after its voting period",
					Long: "Anyone can settle. The first settle decides the proposal from its tally, releases its deposit and, when it passed, " +
						"runs its messages as the DAO. A failed execution can be retried with another settle within the execution_window param " +
						"after the voting end; a settle after the window expires it.",
					Example:        "bzed tx dao settle-proposal 1 --from anyone",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "proposal_id"}},
				},
				{
					RpcMethod: "CleanupProposal",
					Use:       "cleanup-proposal [proposal-id]",
					Short:     "Delete a finished DAO proposal's ballots in batches, then the proposal",
					Long: "Anyone can clean up a REJECTED, EXECUTED, EXPIRED or CANCELLED proposal. Each call deletes up to --limit ballots " +
						"(0 or anything above the cleanup_batch_size param means cleanup_batch_size); the call that deletes the last one " +
						"also deletes the proposal. Repeat until the response says completed. Events keep the history.",
					Example:        "bzed tx dao cleanup-proposal 1 --limit 50 --from anyone",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "proposal_id"}},
					FlagOptions: map[string]*autocliv1.FlagOptions{
						"limit": {Usage: "ballots to delete in this call (0 = the cleanup_batch_size param)"},
					},
				},
				{
					RpcMethod: "UpdateDaoConfig",
					Use:       "update-dao-config [dao-id] [config]",
					Short:     "Replace a DAO's config; signed by the DAO account only",
					Long: "Replace the DAO's whole config, as JSON in the create-dao format (ratios as 18-decimal integers, periods as seconds). " +
						"Only the DAO account can sign it: build the message with --generate-only and run it through admin-execute or a proposal. " +
						"Open proposals keep the config they pinned.",
					Example: `bzed tx dao update-dao-config 1 '{"min_quorum":"500000000000000000","threshold":"500000000000000000","voting_period":"259200s",` +
						`"expedited_period":"86400s","expedite_quorum":"600000000000000000","expedite_threshold":"660000000000000000"}' --from bze1<dao address> --generate-only`,
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "dao_id"}, {ProtoField: "config"}},
				},
				{
					RpcMethod: "UpdateMetadata",
					Use:       "update-metadata [dao-id] [name] [description]",
					Short:     "Replace a DAO's name and description; signed by the DAO account only",
					Long: "Replace the DAO's name (1 to 64 bytes) and description (up to 2,000 bytes); both are always replaced. " +
						"Only the DAO account can sign it: build the message with --generate-only and run it through admin-execute or a proposal.",
					Example:        `bzed tx dao update-metadata 1 "Bee Guild" "The guild's treasury" --from bze1<dao address> --generate-only`,
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "dao_id"}, {ProtoField: "name"}, {ProtoField: "description"}},
				},
				{
					RpcMethod: "UpdateElectorate",
					Use:       "update-electorate [dao-id] [electorate]",
					Short:     "Replace a stake DAO's reward sources; signed by the DAO account only",
					Long: "Replace the stake DAO's whole electorate, as JSON in the create-dao --electorate format. " +
						"Only the DAO account can sign it: build the message with --generate-only and run it through admin-execute or a proposal. " +
						"Open proposals keep the electorate they pinned; a self-owned DAO must keep a live electorate.",
					Example:        `bzed tx dao update-electorate 1 '{"sr_ids":["3","7"],"dr_denom":"ubze"}' --from bze1<dao address> --generate-only`,
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "dao_id"}, {ProtoField: "electorate"}},
				},
				{
					RpcMethod: "UpdateMembers",
					Use:       "update-members [dao-id] --add [address] --remove [address]",
					Short:     "Add and remove members of a static-list DAO; signed by the DAO account only",
					Long: "Add new members and remove current ones; at least one member must remain. " +
						"Only the DAO account can sign it: build the message with --generate-only and run it through admin-execute or a proposal. " +
						"A removed member's votes on open proposals stop counting; added members can vote on them. " +
						"For several addresses, repeat the flag (--add bze1... --add bze1...); a comma-separated list is not accepted.",
					Example:        `bzed tx dao update-members 1 --add bze1<new member> --add bze1<another> --remove bze1<old member> --from bze1<dao address> --generate-only`,
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "dao_id"}},
				},
			},
		},
	}
}
