package cli

import (
	"github.com/bze-alphateam/bze/x/rewards/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/spf13/cobra"
)

func CmdCreateStakingReward() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create-staking-reward [prize-amount] [prize-denom] [staking-denom] [duration] [min-stake] [lock]",
		Short: "Create a new staking reward",
		Args:  cobra.ExactArgs(6),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			// Get indexes
			amount := args[0]
			denom := args[1]
			stakingDenom := args[2]
			duration := args[3]
			minStake := args[4]
			lock := args[4]

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := types.NewMsgCreateStakingReward(
				clientCtx.GetFromAddress().String(),
				amount,
				denom,
				stakingDenom,
				duration,
				minStake,
				lock,
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

func CmdUpdateStakingReward() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update-staking-reward [reward-id] [duration]",
		Short: "Adds new funds to a staking reward for the specified duration",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			// Get indexes
			indexRewardId := args[0]
			argDuration := args[1]

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := types.NewMsgUpdateStakingReward(
				clientCtx.GetFromAddress().String(),
				indexRewardId,
				argDuration,
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
