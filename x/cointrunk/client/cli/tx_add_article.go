package cli

import (
	"strconv"

	"github.com/bze-alphateam/bze/x/cointrunk/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/spf13/cobra"
)

const (
	FlagPicture = "picture"
)

var _ = strconv.Itoa(0)

func CmdAddArticle() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add-article [title] [url]",
		Short: "Broadcast message add-article",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			argTitle := args[0]
			argUrl := args[1]
			argPicture, err := cmd.Flags().GetString(FlagPicture)
			if err != nil {
				return err
			}

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := types.NewMsgAddArticle(
				clientCtx.GetFromAddress().String(),
				argTitle,
				argUrl,
				argPicture,
			)
			if err := msg.ValidateBasic(); err != nil {
				return err
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	cmd.Flags().String(FlagPicture, "", "Picture of the article")
	flags.AddTxFlagsToCmd(cmd)

	return cmd
}
