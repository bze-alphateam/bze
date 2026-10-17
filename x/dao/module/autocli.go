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
					Example: `bzed tx dao admin-execute 1 '{"@type":"/cosmos.bank.v1beta1.MsgSend","from_address":"bze1<dao>","to_address":"bze1...","amount":[{"denom":"ubze","amount":"1000000"}]}' --from admin`,
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
			},
		},
	}
}
