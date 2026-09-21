# TradeBin – Technical Notes

## Order Book
- Markets are keyed by `base/quote`. Orders carry `order_type`, `amount`, `price`, and `market_id`.
- `MsgCreateOrder` applies maker or taker fees based on whether opposite liquidity already exists at the submitted price. If the global queue counter exceeds `order_book_extra_gas_window`, it consumes `(queueCounter-window)*order_book_queue_extra_gas` gas as a spam surcharge; price validation scans each queued message and charges `order_book_queue_message_scan_extra_gas` per scan.
- `MsgFillOrders` batches fills with price/amount pairs; it always follows the taker fee path and charges `fill_orders_extra_gas` once on entry and again per order added to the queue.
- Queue processing at `EndBlock` stops after `order_book_per_block_messages` messages; leftovers remain queued for later blocks. The queue counter resets only when the queue becomes empty.
- Maker/taker fees use params: maker fees are routed via `maker_fee_destination`, taker fees via `taker_fee_destination` (community-pool collector or burner fee collector). Create-market fees always go to the community-pool collector after optional swap to native.
- Halted markets (`halted_denoms`, keeper helpers in `service_halt.go`): `MsgCreateOrder` and `MsgFillOrders` return `ErrDenomHalted` right after loading the market, before gas surcharges, fee capture or escrow. The `EndBlock` engine is deliberately untouched: a queued message was already accepted (validated, fee captured, funds escrowed) a block or a few earlier, so it executes as always — the queue is the asynchronous execution stage, not a second gate. Gov's `EndBlock` runs before tradebin's, so from the block a halt proposal passes no new message enters the queue and the backlog simply drains.

## Liquidity Pools
- Pools are created with an initial deposit and optional `stable` flag; each pool has its own fee and fee destination.
- LP tokens follow the AMM math in `liquidity_pool.go`; slippage controls use `min_lp_tokens`, `min_base`, `min_quote`.
- `MultiSwap` walks a list of pool IDs; input/output slippage is enforced via `min_output`. `getRoutesPools` refuses a route containing a pool with a halted denom (`ErrDenomHalted`, surfaced as-is rather than wrapped in `ErrInvalidRoutes`), `validateMarketAssets` refuses new markets/pools on a halted denom, and `AddLiquidity` refuses a halted pool — all before funds move.
- `Keeper.swapTokens` is the single choke point of every swap (user routes, `ModuleSwapForNativeDenom`, `ModuleAddLiquidityWithNativeDenom`, the fee payer). It returns `ErrDenomHalted` for a pool holding a halted denom, which is what makes the no-swap invariant hold for every caller at once. `ModuleAddLiquidityWithNativeDenom` already treats a failed swap as "refund the coin to the caller"; the fee-payer entry points fall back to capturing the native fee as-is when the preferred fee denom's pool is halted.

## Inter-module Hooks
- `native_denom` is the target for swapping captured fees and module balances; `MsgUpdateParams` enforces it exists in bank supply.
- `min_native_liquidity_for_module_swap` must be met in the native/pair pool before helpers like `ModuleSwapForNativeDenom`, `ModuleAddLiquidityWithNativeDenom`, or the liquidity-depth checks (`HasDeepLiquidityWithNativeDenom`, `CanSwapForNativeDenom`) will act. This prevents draining shallow pools during module-driven swaps.
- `HasLiquidityWithNativeDenom`, `HasDeepLiquidityWithNativeDenom` and `CanSwapForNativeDenom` answer `false` for a halted denom regardless of the pool depth, so the txfeecollector ante handler refuses it as a fee denom up front and the fee collector / burner classify it as non-swappable without attempting a swap (its dust takes the existing path to the black hole). Nothing changes in those modules.
- Trading fees and the create-market fee are captured via the trade module fee paths, optionally swapped to native, then forwarded to fee collector (community pool) or burner depending on configured destinations. Fees sent to `burner` are later destroyed or swapped through burner logic depending on denom type.

## Storage / Queries
- Prefixed stores keep markets, orders, queues, pools, and user dust; gRPC/REST exposes markets, pools, order books, and params for indexers.

## Version History

### v8.2.0
- `halted_denoms` param (`Params.IsDenomHalted`, keeper `IsDenomHalted` / `IsMarketHalted` / `isPoolHalted`), `ErrDenomHalted` (4017), halt guards in the msg servers, in `swapTokens` and in the liquidity answers; the EndBlock engine is untouched. No migration, `ConsensusVersion` stays 4.

### v8.1.0
- Fee payer service (`CaptureAndSwapUserFee`) for fee capture and conversion to native denom via liquidity pools
- Queue rate limiting via `OrderBookPerBlockMessages` parameter (default 500); counter resets only when queue empties
- Pending cancel tracking: `CancelOrder` checks `HasPendingCancel` before queuing to prevent duplicates
- Min liquidity threshold now uses `MinNativeLiquidityForModuleSwap` param instead of hardcoded `50_000_000_000`
- `ModuleSwapForNativeDenom` reordered to capture coins first then swap; returns `sdk.Coin{}` on error
- `FillOrders` gas consumption uses `FillOrdersExtraGas` param (default 5,000) instead of hardcoded constant
- `IterateAllQueueMessages` handler now returns `bool` to control iteration; `RemoveQueueMessage` requires `marketId` parameter
- Queue message keys restructured to composite `{market}/{id}` format for market-scoped queries
- History order index format changed to `{messageId}{orderId}` for correct sorting
- `CalculateMinAmount` now returns error and is properly checked
- Order key precision migration (24-char/10-decimal → 32-char/18-decimal) moved to module migration (v3→v4) with write-before-delete pattern
- v2 parameters: `sdk.Coin`-based fee fields + 6 new gas/liquidity parameters
- ConsensusVersion bumped from 3 to 4
