package cli

import (
	"encoding/json"
	"fmt"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/spf13/cast"
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
	cmd.AddCommand(CmdFillOrders())
	cmd.AddCommand(CmdCreateLiquidityPool())
	cmd.AddCommand(CmdAddLiquidity())
	cmd.AddCommand(CmdRemoveLiquidity())
	cmd.AddCommand(CmdMultiSwap())
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
				argMarketId,
				argOrderType,
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

func CmdCreateLiquidityPool() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create-liquidity-pool [base] [quote] [fee] [fee-dest] [stable] [initial-base] [initial-quote]",
		Short: "Broadcast message create-liquidity-pool",
		Long:  "The fee-dest should be an array of objects of form: {treasury: 0.25, burner: 0.25, providers: 0.25, liquidity: 0.25} and the sum of the values should be 1.",
		Args:  cobra.ExactArgs(7),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			argBase := args[0]
			argQuote := args[1]
			argFee := args[2]
			argFeeDest := args[3]
			argStable, err := cast.ToBoolE(args[4])
			if err != nil {
				return err
			}
			argInitialBase, err := cast.ToUint64E(args[5])
			if err != nil {
				return err
			}
			argInitialQuote, err := cast.ToUint64E(args[6])
			if err != nil {
				return err
			}

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := types.NewMsgCreateLiquidityPool(
				clientCtx.GetFromAddress().String(),
				argBase,
				argQuote,
				argFee,
				argFeeDest,
				argStable,
				argInitialBase,
				argInitialQuote,
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

func CmdAddLiquidity() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add-liquidity [pool-id] [base-amount] [quote-amount] [min-lp-tokens]",
		Short: "Broadcast message add-liquidity",
		Args:  cobra.ExactArgs(4),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			argPoolId := args[0]
			argBaseAmount, err := cast.ToUint64E(args[1])
			if err != nil {
				return err
			}
			argQuoteAmount, err := cast.ToUint64E(args[2])
			if err != nil {
				return err
			}
			argMinLpTokens, err := cast.ToUint64E(args[3])
			if err != nil {
				return err
			}

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := types.NewMsgAddLiquidity(
				clientCtx.GetFromAddress().String(),
				argPoolId,
				argBaseAmount,
				argQuoteAmount,
				argMinLpTokens,
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

func CmdRemoveLiquidity() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove-liquidity [pool-id] [lp-tokens] [min-base] [min-quote]",
		Short: "Broadcast message remove-liquidity",
		Args:  cobra.ExactArgs(4),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			argPoolId := args[0]

			lpTokens, err := strconv.ParseUint(args[1], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid LP tokens amount: %v", err)
			}

			// Parse min base as uint64
			minBase, err := strconv.ParseUint(args[2], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid min base amount: %v", err)
			}

			// Parse min quote as uint64
			minQuote, err := strconv.ParseUint(args[3], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid min quote amount: %v", err)
			}

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := types.NewMsgRemoveLiquidity(
				clientCtx.GetFromAddress().String(),
				argPoolId,
				lpTokens,
				minBase,
				minQuote,
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

func CmdMultiSwap() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "multi-swap [routes] [input] [min-output]",
		Short: "Broadcast message multi-swap",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			argRoutes := args[0]
			var routes []string
			err = json.Unmarshal([]byte(argRoutes), &routes)
			if err != nil {
				return fmt.Errorf("failed to parse routes: %w", err)
			}

			argInput := args[1]
			argMinOutput := args[2]

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := types.NewMsgMultiSwap(
				clientCtx.GetFromAddress().String(),
				routes,
				argInput,
				argMinOutput,
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
