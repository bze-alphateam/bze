package tradebin

import (
	autocliv1 "cosmossdk.io/api/cosmos/autocli/v1"

	modulev1 "github.com/bze-alphateam/bze/api/bze/tradebin"
)

// AutoCLIOptions implements the autocli.HasAutoCLIConfig interface.
func (am AppModule) AutoCLIOptions() *autocliv1.ModuleOptions {
	return &autocliv1.ModuleOptions{
		Query: &autocliv1.ServiceCommandDescriptor{
			Service: modulev1.Query_ServiceDesc.ServiceName,
			RpcCommandOptions: []*autocliv1.RpcCommandOptions{
				{
					RpcMethod: "Params",
					Use:       "params",
					Short:     "Shows the parameters of the module",
				},
				// this line is used by ignite scaffolding # autocli/query
			},
		},
		Tx: &autocliv1.ServiceCommandDescriptor{
			Service:              modulev1.Msg_ServiceDesc.ServiceName,
			EnhanceCustomCommand: true, // only required if you want to use the custom command
			RpcCommandOptions: []*autocliv1.RpcCommandOptions{
				{
					RpcMethod: "UpdateParams",
					Skip:      true, // skipped because authority gated
				},
				{
					RpcMethod:      "CreateMarket",
					Use:            "create-market [base] [quote]",
					Short:          "Send a create-market tx",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "base"}, {ProtoField: "quote"}},
				},
				{
					RpcMethod:      "CreateOrder",
					Use:            "create-order [order-type] [amount] [price] [market-id]",
					Short:          "Send a create-order tx",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "order_type"}, {ProtoField: "amount"}, {ProtoField: "price"}, {ProtoField: "market_id"}},
				},
				{
					RpcMethod:      "CancelOrder",
					Use:            "cancel-order [market-id] [order-id] [order-type]",
					Short:          "Send a cancel-order tx",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "market_id"}, {ProtoField: "order_id"}, {ProtoField: "order_type"}},
				},
				{
					RpcMethod:      "FillOrders",
					Use:            "fill-orders [market-id] [order-type] [orders]",
					Short:          "Send a fill-orders tx",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "market_id"}, {ProtoField: "order_type"}, {ProtoField: "orders"}},
				},
				{
					RpcMethod:      "CreateLiquidityPool",
					Use:            "create-liquidity-pool [base] [quote] [fee] [fee-dest] [stable] [initial-base] [initial-quote]",
					Short:          "Send a create-liquidity-pool tx",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "base"}, {ProtoField: "quote"}, {ProtoField: "fee"}, {ProtoField: "fee_dest"}, {ProtoField: "stable"}, {ProtoField: "initial_base"}, {ProtoField: "initial_quote"}},
				},
				{
					RpcMethod:      "AddLiquidity",
					Use:            "add-liquidity [pool-id] [base-amount] [quote-amount] [min-lp-tokens]",
					Short:          "Send a add-liquidity tx",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "pool_id"}, {ProtoField: "base_amount"}, {ProtoField: "quote_amount"}, {ProtoField: "min_lp_tokens"}},
				},
				{
					RpcMethod:      "RemoveLiquidity",
					Use:            "remove-liquidity [pool-id] [lp-tokens] [min-base] [min-quote]",
					Short:          "Send a remove-liquidity tx",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "pool_id"}, {ProtoField: "lp_tokens"}, {ProtoField: "min_base"}, {ProtoField: "min_quote"}},
				},
				{
					RpcMethod:      "MultiSwap",
					Use:            "multi-swap [routes] [input] [min-output]",
					Short:          "Send a multi-swap tx",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "routes"}, {ProtoField: "input"}, {ProtoField: "min_output"}},
				},
				// this line is used by ignite scaffolding # autocli/tx
			},
		},
	}
}
