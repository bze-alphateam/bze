package daodao

import (
	autocliv1 "cosmossdk.io/api/cosmos/autocli/v1"

	modulev1 "github.com/bze-alphateam/bze/api/bze/daodao"
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
					Use:            "dao [dao-id]",
					Short:          "Show a DAO by id",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "dao_id"}},
				},
				{
					RpcMethod:      "DaoByAddress",
					Use:            "dao-by-address [address]",
					Short:          "Show a DAO by its on-chain account address",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "address"}},
				},
				{
					RpcMethod: "Daos",
					Use:       "daos",
					Short:     "List all DAOs (paginated)",
				},
				{
					RpcMethod:      "DaosByCreator",
					Use:            "daos-by-creator [creator]",
					Short:          "List DAOs created by an address (paginated)",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "creator"}},
				},
				{
					RpcMethod:      "SubDaos",
					Use:            "sub-daos [parent-dao-id]",
					Short:          "List the sub-DAOs of a given parent DAO",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "parent_dao_id"}},
				},
				{
					RpcMethod:      "VotingPower",
					Use:            "voting-power [dao-id] [address]",
					Short:          "Show an address's current voting power within a DAO",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "dao_id"}, {ProtoField: "address"}},
				},
				{
					RpcMethod:      "TotalVotingPower",
					Use:            "total-voting-power [dao-id]",
					Short:          "Show a DAO's current total voting power",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "dao_id"}},
				},
				{
					RpcMethod:      "Members",
					Use:            "members [dao-id]",
					Short:          "List the (address, weight) members of a STATIC DAO",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "dao_id"}},
				},
				{
					RpcMethod:      "GovernanceConfig",
					Use:            "governance-config [dao-id]",
					Short:          "Show a DAO's proposal-track governance config",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "dao_id"}},
				},
				{
					RpcMethod:      "Proposal",
					Use:            "proposal [dao-id] [proposal-id]",
					Short:          "Show a single proposal",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "dao_id"}, {ProtoField: "proposal_id"}},
				},
				{
					RpcMethod: "Proposals",
					Use:       "proposals [dao-id]",
					Short:     "List a DAO's proposals (paginated; optional --status-filter)",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "dao_id"}},
					FlagOptions: map[string]*autocliv1.FlagOptions{
						"status_filter": {Name: "status-filter", Usage: "ProposalStatus enum value (default UNSPECIFIED = no filter)"},
					},
				},
				{
					RpcMethod:      "Tally",
					Use:            "tally [dao-id] [proposal-id]",
					Short:          "Show a proposal's running tally and status",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "dao_id"}, {ProtoField: "proposal_id"}},
				},
				{
					RpcMethod:      "Vote",
					Use:            "vote [dao-id] [proposal-id] [voter]",
					Short:          "Show a single voter's vote on a proposal",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "dao_id"}, {ProtoField: "proposal_id"}, {ProtoField: "voter"}},
				},
				{
					RpcMethod:      "Votes",
					Use:            "votes [dao-id] [proposal-id]",
					Short:          "List all votes on a proposal (paginated)",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "dao_id"}, {ProtoField: "proposal_id"}},
				},
				{
					RpcMethod:      "DepositConfig",
					Use:            "deposit-config [dao-id]",
					Short:          "Show a DAO's deposit-period config",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "dao_id"}},
				},
				{
					RpcMethod:      "Deposits",
					Use:            "deposits [dao-id] [proposal-id]",
					Short:          "List per-depositor deposit records for a proposal (paginated)",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "dao_id"}, {ProtoField: "proposal_id"}},
				},
				{
					RpcMethod:      "Poll",
					Use:            "poll [dao-id] [poll-id]",
					Short:          "Show a single poll",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "dao_id"}, {ProtoField: "poll_id"}},
				},
				{
					RpcMethod:      "Polls",
					Use:            "polls [dao-id]",
					Short:          "List a DAO's polls (paginated; optional --status-filter)",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "dao_id"}},
					FlagOptions: map[string]*autocliv1.FlagOptions{
						"status_filter": {Name: "status-filter", Usage: "PollStatus enum value (default UNSPECIFIED = no filter)"},
					},
				},
				{
					RpcMethod:      "PollVote",
					Use:            "poll-vote [dao-id] [poll-id] [voter]",
					Short:          "Show a single voter's selection on a poll",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "dao_id"}, {ProtoField: "poll_id"}, {ProtoField: "voter"}},
				},
				{
					RpcMethod:      "PollVotes",
					Use:            "poll-votes [dao-id] [poll-id]",
					Short:          "List all votes on a poll (paginated)",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "dao_id"}, {ProtoField: "poll_id"}},
				},
				{
					RpcMethod:      "PollTally",
					Use:            "poll-tally [dao-id] [poll-id]",
					Short:          "Show a poll's running tally + status + winning_choice_index",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "dao_id"}, {ProtoField: "poll_id"}},
				},
				{
					RpcMethod:      "PollDeposits",
					Use:            "poll-deposits [dao-id] [poll-id]",
					Short:          "List per-depositor deposit records for a poll (paginated)",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "dao_id"}, {ProtoField: "poll_id"}},
				},
			},
		},
		Tx: &autocliv1.ServiceCommandDescriptor{
			Service:              modulev1.Msg_ServiceDesc.ServiceName,
			EnhanceCustomCommand: true,
			RpcCommandOptions: []*autocliv1.RpcCommandOptions{
				{
					RpcMethod: "UpdateParams",
					Skip:      true, // chain-gov authority only
				},
				{
					// MsgCreateDao has nested DaoMetadata which autocli handles via JSON
					// for the `metadata` flag. Admin and parent_dao_id are optional flags.
					RpcMethod: "CreateDao",
					Use:       "create-dao",
					Short:     "Create a new DAO",
					FlagOptions: map[string]*autocliv1.FlagOptions{
						"admin":         {Name: "admin", Usage: "DAO admin address (defaults to creator)"},
						"parent_dao_id": {Name: "parent-dao-id", Usage: "Parent DAO id (0 = no parent)"},
					},
				},
				{
					RpcMethod: "UpdateDaoMetadata",
					Use:       "update-dao-metadata [dao-id]",
					Short:     "Update a DAO's metadata (admin-only)",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "dao_id"},
					},
				},
				{
					RpcMethod: "UpdateDaoAdmin",
					Use:       "update-dao-admin [dao-id] [new-admin]",
					Short:     "Nominate a new admin for a DAO (admin-only; nominee must accept)",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "dao_id"},
						{ProtoField: "new_admin"},
					},
				},
				{
					RpcMethod: "AcceptDaoAdmin",
					Use:       "accept-dao-admin [dao-id]",
					Short:     "Accept admin handoff for a DAO (signed by the pending admin)",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "dao_id"},
					},
				},
				{
					RpcMethod: "UpdateMembers",
					Use:       "update-members [dao-id]",
					Short:     "Add and/or remove members of a STATIC DAO (admin-gated)",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "dao_id"},
					},
				},
				{
					RpcMethod: "CreateProposal",
					Use:       "create-proposal [dao-id]",
					Short:     "Create a new proposal on a DAO (proposer must be a current member in Epic 3)",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "dao_id"},
					},
				},
				{
					RpcMethod: "Vote",
					Use:       "vote [dao-id] [proposal-id] [option]",
					Short:     "Cast (or replace, with revote enabled) a vote on a proposal",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "dao_id"},
						{ProtoField: "proposal_id"},
						{ProtoField: "option"},
					},
				},
				{
					RpcMethod: "UpdateGovernanceConfig",
					Use:       "update-governance-config [dao-id]",
					Short:     "Replace a DAO's proposal-track governance config (admin-gated)",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "dao_id"},
					},
				},
				{
					RpcMethod: "Deposit",
					Use:       "deposit [dao-id] [proposal-id] [amount]",
					Short:     "Top up a DEPOSIT_PERIOD proposal's escrow",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "dao_id"},
						{ProtoField: "proposal_id"},
						{ProtoField: "amount"},
					},
				},
				{
					RpcMethod: "UpdateDepositConfig",
					Use:       "update-deposit-config [dao-id]",
					Short:     "Replace a DAO's deposit-period config (admin-gated)",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "dao_id"},
					},
				},
				{
					RpcMethod: "ExecuteProposal",
					Use:       "execute-proposal [dao-id] [proposal-id]",
					Short:     "Dispatch a PASSED proposal's msgs[] atomically (anyone can submit)",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "dao_id"},
						{ProtoField: "proposal_id"},
					},
				},
				{
					RpcMethod: "RenounceAdmin",
					Use:       "renounce-admin [dao-id]",
					Short:     "Flip the DAO to self-governance (admin-gated; irreversible)",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "dao_id"},
					},
				},
				{
					RpcMethod: "UpdateVotingBackend",
					Use:       "update-voting-backend [dao-id]",
					Short:     "Reconfigure the DAO's voting backend (same-type only in v1; admin-gated)",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "dao_id"},
					},
				},
				{
					RpcMethod: "CreatePoll",
					Use:       "create-poll [dao-id]",
					Short:     "Open a new informational poll on a DAO",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "dao_id"},
					},
				},
				{
					RpcMethod: "VoteOnPoll",
					Use:       "vote-on-poll [dao-id] [poll-id]",
					Short:     "Cast an approval-style selection set on a poll",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "dao_id"},
						{ProtoField: "poll_id"},
					},
				},
				{
					RpcMethod: "DepositOnPoll",
					Use:       "deposit-on-poll [dao-id] [poll-id] [amount]",
					Short:     "Top up a DEPOSIT_PERIOD poll's escrow",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "dao_id"},
						{ProtoField: "poll_id"},
						{ProtoField: "amount"},
					},
				},
			},
		},
	}
}
