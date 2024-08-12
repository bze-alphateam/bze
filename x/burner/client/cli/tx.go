package cli

import (
	"fmt"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/spf13/cobra"
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	// "github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/bze-alphateam/bze/x/burner/types"
)

var _ = strconv.Itoa(0)

// GetTxCmd returns the transaction commands for this module
func GetTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      fmt.Sprintf("%s transactions subcommands", types.ModuleName),
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(CmdFundBurner())
	cmd.AddCommand(CmdStartRaffle())
	cmd.AddCommand(CmdJoinRaffle())
	// this line is used by starport scaffolding # 1

	return cmd
}

func CmdFundBurner() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "fund-burner [amount]",
		Short: "Broadcast message fund-burner",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			argAmount := args[0]

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := types.NewMsgFundBurner(
				clientCtx.GetFromAddress().String(),
				argAmount,
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

func CmdStartRaffle() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start-raffle [pot] [duration] [chances] [ratio] [ticket-price] [denom]",
		Short: "Broadcast message start-raffle",
		Args:  cobra.ExactArgs(6),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			argPot := args[0]
			argDuration := args[1]
			argChances := args[2]
			argRatio := args[3]
			argTicketPrice := args[4]
			argDenom := args[5]

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := types.NewMsgStartRaffle(
				clientCtx.GetFromAddress().String(),
				argPot,
				argDuration,
				argChances,
				argRatio,
				argTicketPrice,
				argDenom,
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
