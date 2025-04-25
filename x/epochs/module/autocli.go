package epochs

import (
	autocliv1 "cosmossdk.io/api/cosmos/autocli/v1"
	"github.com/bze-alphateam/bze/x/epochs/types"

	modulev1 "github.com/bze-alphateam/bze/api/bze/epoch"
)

// AutoCLIOptions implements the autocli.HasAutoCLIConfig interface.
func (am AppModule) AutoCLIOptions() *autocliv1.ModuleOptions {
	return &autocliv1.ModuleOptions{
		Query: &autocliv1.ServiceCommandDescriptor{
			Service: types.Query_serviceDesc.ServiceName,
			RpcCommandOptions: []*autocliv1.RpcCommandOptions{
				{
					RpcMethod: "EpochInfos",
					Use:       "all",
					Short:     "Query epochs info",
				},
				{
					RpcMethod:      "CurrentEpoch",
					Use:            "identifier [identifier]",
					Short:          "Query epoch info by identifier",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "identifier"}},
				},
				// this line is used by ignite scaffolding # autocli/query
			},
		},
		Tx: &autocliv1.ServiceCommandDescriptor{
			Service:              modulev1.Msg_ServiceDesc.ServiceName,
			EnhanceCustomCommand: true, // only required if you want to use the custom command
			RpcCommandOptions: []*autocliv1.RpcCommandOptions{
				//{
				//	RpcMethod: "UpdateParams",
				//	Skip:      true, // skipped because authority gated
				//},
				// this line is used by ignite scaffolding # autocli/tx
			},
		},
	}
}
