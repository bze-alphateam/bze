package cli

import (
	"strconv"

	"github.com/bze-alphateam/bze/x/tradebin/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/spf13/cobra"
)

var _ = strconv.Itoa(0)

func CmdMarketOrder() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "market-order [market] [order-type] [order-id]",
		Short: "Query market-order",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			reqMarket := args[0]
			reqOrderType := args[1]
			reqOrderId := args[2]

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			params := &types.QueryMarketOrderRequest{

				Market:    reqMarket,
				OrderType: reqOrderType,
				OrderId:   reqOrderId,
			}

			res, err := queryClient.MarketOrder(cmd.Context(), params)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
