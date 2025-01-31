package cli

import (
	"encoding/json"
	"fmt"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"strconv"
	"time"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	// "github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/bze-alphateam/bze/x/tradebin/types"
)

var (
	DefaultRelativePacketTimeoutTimestamp = uint64((time.Duration(10) * time.Minute).Nanoseconds())
	_                                     = strconv.Itoa(0)
)

const (
	flagPacketTimeoutTimestamp = "packet-timeout-timestamp"
	listSeparator              = ","
)

// GetTxCmd returns the transaction commands for this module
func GetTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      fmt.Sprintf("%s transactions subcommands", types.ModuleName),
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(CmdCreateMarket())
	cmd.AddCommand(CmdCreateOrder())
	cmd.AddCommand(CmdCancelOrder())
	// this line is used by starport scaffolding # 1

	return cmd
}

func CmdCancelOrder() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cancel-order [market-id] [order-id] [order-type]",
		Short: "Broadcast message CancelOrder",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			argMarketId := args[0]
			argOrderId := args[1]
			argOrderType := args[2]

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := types.NewMsgCancelOrder(
				clientCtx.GetFromAddress().String(),
				argMarketId,
				argOrderId,
				argOrderType,
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

func CmdCreateMarket() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create-market [base] [quote]",
		Short: "Broadcast message CreateMarket",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			argAsset1 := args[0]
			argAsset2 := args[1]

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := types.NewMsgCreateMarket(
				clientCtx.GetFromAddress().String(),
				argAsset1,
				argAsset2,
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

func CmdCreateOrder() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create-order [order-type] [amount] [price] [market-id]",
		Short: "Broadcast message CreateOrder",
		Args:  cobra.ExactArgs(4),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			argOrderType := args[0]
			argAmount := args[1]
			argPrice := args[2]
			argMarketId := args[3]

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := types.NewMsgCreateOrder(
				clientCtx.GetFromAddress().String(),
				argOrderType,
				argAmount,
				argPrice,
				argMarketId,
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

func CmdFillOrders() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "fill-orders [order-type] [market-id] [orders]",
		Short: "Broadcast message FillOrders",
		Long:  "This command expects [orders] to be an array of objects of form: {amount: 231, price:0.32}",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			argOrderType := args[0]
			argMarketId := args[1]
			argOrders := args[2]
			var decodedOrders []*types.FillOrderItem
			err = json.Unmarshal([]byte(argOrders), &decodedOrders)
			if err != nil {
				return err
			}

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := types.NewMsgFillOrders(
				clientCtx.GetFromAddress().String(),
				argOrderType,
				argMarketId,
				decodedOrders,
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
