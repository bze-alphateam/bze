package cmd

import (
	"cosmossdk.io/client/v2/autocli"
	"cosmossdk.io/client/v2/autocli/flag"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/gogoproto/proto"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/reflect/protoregistry"

	daocli "github.com/bze-alphateam/bze/x/dao/client/cli"
)

// enhanceRootCommand adds the autocli commands to the root command, as
// AppOptions.EnhanceRootCommand does, with the app's own flag types: x/dao's
// DaoConfig takes its ratios as decimals ("0.5").
func enhanceRootCommand(opts autocli.AppOptions, rootCmd *cobra.Command) error {
	files, err := proto.MergedRegistry()
	if err != nil {
		return err
	}

	builder := &autocli.Builder{
		Builder: flag.Builder{
			TypeResolver:          protoregistry.GlobalTypes,
			FileResolver:          files,
			AddressCodec:          opts.AddressCodec,
			ValidatorAddressCodec: opts.ValidatorAddressCodec,
			ConsensusAddressCodec: opts.ConsensusAddressCodec,
		},
		GetClientConn: func(cmd *cobra.Command) (grpc.ClientConnInterface, error) {
			return client.GetClientQueryContext(cmd)
		},
		AddQueryConnFlags: func(c *cobra.Command) {
			flags.AddQueryFlagsToCmd(c)
			flags.AddKeyringFlags(c.Flags())
		},
		AddTxConnFlags: flags.AddTxFlagsToCmd,
	}
	builder.DefineMessageFlagType("bze.dao.DaoConfig", daocli.DecJSONMessageType{Name: "bze.dao.DaoConfig"})

	return opts.EnhanceRootCommandWithBuilder(rootCmd, builder)
}
