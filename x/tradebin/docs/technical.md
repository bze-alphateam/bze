# TradeBin – Technical Notes

## Order Book
- Markets are keyed by `base/quote`. Orders carry `order_type`, `amount`, `price`, and `market_id`.
- `MsgFillOrders` batches fills with price/amount pairs. Extra gas parameters guard against large batches (`order_book_extra_gas_window`, `order_book_queue_extra_gas`, `fill_orders_extra_gas`).
- Maker/taker fees are assessed per side; destinations are either the burner module or community pool. Fees use the denom of the traded asset per message logic.

## Liquidity Pools
- Pools are created with an initial deposit and optional `stable` flag; each pool has its own fee and fee destination.
- LP tokens follow the AMM math in `liquidity_pool.go`; slippage controls use `min_lp_tokens`, `min_base`, `min_quote`.
- `MultiSwap` walks a list of pool IDs; input/output slippage is enforced via `min_output`.

## Inter-module Hooks
- `native_denom` and `min_native_liquidity_for_module_swap` gate module-initiated swaps used by other modules (e.g., burner/fee conversion).
- Trading fees and the create-market fee are captured via the trade module fee paths, optionally swapped to native, then forwarded to fee collector (community pool) or burner depending on configured destinations.
- Fees sent to `burner` are later destroyed or swapped through burner logic depending on denom type.

## Storage / Queries
- Prefixed stores keep markets, orders, queues, pools, and user dust; gRPC/REST exposes markets, pools, order books, and params for indexers.
