package cli

import (
	"strconv"

	"github.com/bze-alphateam/bze/x/rewards/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/spf13/cobra"
)

var _ = strconv.Itoa(0)

func CmdGetMarketIdTradingRewardIdHandler() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get-market-id-trading-reward-id-handler [market-id]",
		Short: "Query GetMarketIdTradingRewardIdHandler",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			reqMarketId := args[0]

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			params := &types.QueryGetMarketIdTradingRewardIdHandlerRequest{

				MarketId: reqMarketId,
			}

			res, err := queryClient.GetMarketIdTradingRewardIdHandler(cmd.Context(), params)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
