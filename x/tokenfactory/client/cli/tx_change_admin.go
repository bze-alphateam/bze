package cli

import (
	"strconv"

	"github.com/bze-alphateam/bze/x/tokenfactory/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/spf13/cobra"
)

var _ = strconv.Itoa(0)

func CmdChangeAdmin() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "change-admin [denom] [new-admin]",
		Short: "Broadcast message ChangeAdmin",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			argDenom := args[0]
			argNewAdmin := args[1]

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := types.NewMsgChangeAdmin(
				clientCtx.GetFromAddress().String(),
				argDenom,
				argNewAdmin,
			)
			if err := msg.ValidateBasic(); err != nil {
				return err
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)

	return cmd
}
