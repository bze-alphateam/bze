package cli

import (
	"strconv"

	"github.com/spf13/cobra"

	"github.com/bze-alphateam/bze/x/scavenge/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
)

var _ = strconv.Itoa(0)

func CmdRevealSolution() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "reveal-solution [scavengeIndex] [solution]",
		Short: "Broadcast message reveal-solution",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			argScavengeIndex := args[0]
			argSolution := args[1]

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := types.NewMsgRevealSolution(
				clientCtx.GetFromAddress().String(),
				argSolution,
				argScavengeIndex,
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
