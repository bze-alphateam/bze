package cli

import (
    "context"
	
    "github.com/spf13/cobra"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
    "github.com/bze-alphateam/bze/x/rewards/types"
)

func CmdListStakingReward() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list-staking-reward",
		Short: "list all StakingReward",
		RunE: func(cmd *cobra.Command, args []string) error {
            clientCtx := client.GetClientContextFromCmd(cmd)

            pageReq, err := client.ReadPageRequest(cmd.Flags())
            if err != nil {
                return err
            }

            queryClient := types.NewQueryClient(clientCtx)

            params := &types.QueryAllStakingRewardRequest{
                Pagination: pageReq,
            }

            res, err := queryClient.StakingRewardAll(context.Background(), params)
            if err != nil {
                return err
            }

            return clientCtx.PrintProto(res)
		},
	}

	flags.AddPaginationFlagsToCmd(cmd, cmd.Use)
	flags.AddQueryFlagsToCmd(cmd)

    return cmd
}

func CmdShowStakingReward() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show-staking-reward [reward-id]",
		Short: "shows a StakingReward",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
            clientCtx := client.GetClientContextFromCmd(cmd)

            queryClient := types.NewQueryClient(clientCtx)

             argRewardId := args[0]
            
            params := &types.QueryGetStakingRewardRequest{
                RewardId: argRewardId,
                
            }

            res, err := queryClient.StakingReward(context.Background(), params)
            if err != nil {
                return err
            }

            return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

    return cmd
}
