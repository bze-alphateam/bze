package cli

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/bze-alphateam/bzedgev5/x/scavenge/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
)

func CmdListScavenge() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list-scavenge",
		Short: "list all scavenge",
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx := client.GetClientContextFromCmd(cmd)

			pageReq, err := client.ReadPageRequest(cmd.Flags())
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			params := &types.QueryAllScavengeRequest{
				Pagination: pageReq,
			}

			res, err := queryClient.ScavengeAll(context.Background(), params)
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

func CmdShowScavenge() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show-scavenge [index]",
		Short: "shows a scavenge",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			clientCtx := client.GetClientContextFromCmd(cmd)

			queryClient := types.NewQueryClient(clientCtx)

			argIndex := args[0]

			params := &types.QueryGetScavengeRequest{
				Index: argIndex,
			}

			res, err := queryClient.Scavenge(context.Background(), params)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
