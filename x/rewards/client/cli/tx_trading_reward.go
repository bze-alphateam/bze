package cli

import (
	"github.com/bze-alphateam/bze/x/rewards/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/spf13/cobra"
)

func CmdCreateTradingReward() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create-trading-reward [prize-amount] [prize-denom] [duration] [market-id] [slots]",
		Short: "Create a new TradingReward",
		Args:  cobra.ExactArgs(5),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			// Get value arguments
			argPrizeAmount := args[0]
			argPrizeDenom := args[1]
			argDuration := args[2]
			argMarketId := args[3]
			argSlots := args[4]

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := types.NewMsgCreateTradingReward(
				clientCtx.GetFromAddress().String(),
				argPrizeAmount,
				argPrizeDenom,
				argDuration,
				argMarketId,
				argSlots,
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
