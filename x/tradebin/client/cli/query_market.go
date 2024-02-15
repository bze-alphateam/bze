package cli

import (
	"context"

	"github.com/bze-alphateam/bze/x/tradebin/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/spf13/cobra"
)

func CmdListMarket() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list-market",
		Short: "list all market",
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx := client.GetClientContextFromCmd(cmd)

			pageReq, err := client.ReadPageRequest(cmd.Flags())
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			params := &types.QueryAllMarketRequest{
				Pagination: pageReq,
			}

			res, err := queryClient.MarketAll(context.Background(), params)
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

func CmdShowMarket() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show-market [asset-1] [asset-2]",
		Short: "shows a market",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			clientCtx := client.GetClientContextFromCmd(cmd)

			queryClient := types.NewQueryClient(clientCtx)

			argAsset1 := args[0]
			argAsset2 := args[1]

			params := &types.QueryGetMarketRequest{
				Asset1: argAsset1,
				Asset2: argAsset2,
			}

			res, err := queryClient.Market(context.Background(), params)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
