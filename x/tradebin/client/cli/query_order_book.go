package cli

import (
	"context"
	"strconv"

	"github.com/bze-alphateam/bze/x/tradebin/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/spf13/cobra"
)

var _ = strconv.Itoa(0)

func CmdAllUserDust() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "all-user-dust",
		Short: "Query AllUserDust",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) (err error) {

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			params := &types.QueryAllUserDustRequest{}

			res, err := queryClient.AllUserDust(cmd.Context(), params)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}

func CmdAssetMarkets() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "asset-markets [asset]",
		Short: "Query asset-markets",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			reqAsset := args[0]

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			params := &types.QueryAssetMarketsRequest{
				Asset: reqAsset,
			}

			res, err := queryClient.AssetMarkets(cmd.Context(), params)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}

func CmdListMarket() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list-market",
		Short: "list all markets",
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
		Use:   "show-market [baseAsset] [quoteAsset]",
		Short: "shows a market for the given assets",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			clientCtx := client.GetClientContextFromCmd(cmd)

			queryClient := types.NewQueryClient(clientCtx)

			argAsset1 := args[0]
			argAsset2 := args[1]

			params := &types.QueryGetMarketRequest{
				Base:  argAsset1,
				Quote: argAsset2,
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

func CmdMarketAggregatedOrders() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "market-aggregated-orders [market] [order-type]",
		Short: "Query market-aggregated-orders",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			reqMarket := args[0]
			reqOrderType := args[1]

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			params := &types.QueryMarketAggregatedOrdersRequest{

				Market:    reqMarket,
				OrderType: reqOrderType,
			}

			pageReq, err := client.ReadPageRequest(cmd.Flags())
			if err != nil {
				return err
			}
			params.Pagination = pageReq

			res, err := queryClient.MarketAggregatedOrders(cmd.Context(), params)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}

func CmdMarketHistory() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "market-history [market]",
		Short: "Query market-history",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			reqMarket := args[0]

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			params := &types.QueryMarketHistoryRequest{

				Market: reqMarket,
			}

			pageReq, err := client.ReadPageRequest(cmd.Flags())
			if err != nil {
				return err
			}
			params.Pagination = pageReq

			res, err := queryClient.MarketHistory(cmd.Context(), params)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}

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

func CmdUserMarketOrders() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "user-market-orders [address] [market-id]",
		Short: "Query user-market-orders",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			reqAddress := args[0]
			reqMarketId := args[1]

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			params := &types.QueryUserMarketOrdersRequest{

				Address: reqAddress,
				Market:  reqMarketId,
			}

			pageReq, err := client.ReadPageRequest(cmd.Flags())
			if err != nil {
				return err
			}
			params.Pagination = pageReq

			res, err := queryClient.UserMarketOrders(cmd.Context(), params)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
