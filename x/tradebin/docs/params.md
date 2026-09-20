# TradeBin Parameters

- **`create_market_fee`** (`sdk.Coin`, default `25000000000ubze`): Charged during `MsgCreateMarket`; captured from the creator, swapped to `native_denom` if needed, and forwarded to the community-pool fee collector. Non-positive disables the charge.
- **`market_maker_fee`** (`sdk.Coin`, default `1000ubze`): Taken on `MsgCreateOrder` when the order **does not** immediately match opposite liquidity (maker path). Captured, optionally swapped to `native_denom`, then sent to `maker_fee_destination`.
- **`market_taker_fee`** (`sdk.Coin`, default `100000ubze`): Taken on `MsgCreateOrder` when the order crosses existing liquidity and on every `MsgFillOrders` call (taker path). Captured, optionally swapped to `native_denom`, then sent to `taker_fee_destination`.
- **`maker_fee_destination`** (`community_pool` | `burner`, default `burner`): Module that receives maker fees (community-pool collector or burner fee collector).
- **`taker_fee_destination`** (`community_pool` | `burner`, default `burner`): Module that receives taker fees.
- **`native_denom`** (`string`, default `ubze`): Base denom used when swapping collected fees or module balances; must exist in bank supply.
- **Gas controls:**
  - `order_book_extra_gas_window` (`uint64`, default `100`): Queue size allowed before surcharging order submissions.
  - `order_book_queue_extra_gas` (`uint64`, default `25000`): Extra gas charged in `MsgCreateOrder` for each queued message beyond the window.
  - `fill_orders_extra_gas` (`uint64`, default `5000`): Gas charged once when `MsgFillOrders` starts and again for each order it enqueues to the queue.
  - `order_book_queue_message_scan_extra_gas` (`uint64`, default `5000`): Gas charged per queued message scanned while validating a new order’s price.
  - `order_book_per_block_messages` (`uint64`, default `500`): Max queue messages processed in a block; processing stops at this limit and resumes next block. The queue counter resets only when the queue becomes empty.
- **`min_native_liquidity_for_module_swap`** (`Int`, default `100000000000`): Minimum native reserves required in a native/pair pool before module-driven swaps or add-liquidity helpers execute.
- **`halted_denoms`** (`[]string`, default empty): Denoms governance has halted on the DEX. Exact string match, at most 32 entries, no duplicates, and `native_denom` can never be listed (it would halt every market and pool). A market or pool whose base **or** quote is halted refuses `MsgCreateMarket`, `MsgCreateLiquidityPool`, `MsgCreateOrder`, `MsgFillOrders`, `MsgAddLiquidity` and any `MsgMultiSwap` route that touches it (also as an intermediate hop) with `ErrDenomHalted` (code 4017) before any fee is captured or funds are escrowed. Queued order book messages on a halted market are refunded in full by `EndBlock` (`QueueMessageRefundedEvent`, reason `market_halted`). `MsgCancelOrder` and `MsgRemoveLiquidity` keep working. A halted denom is never swapped by anyone — user routes, module swaps and fee swaps all refuse it — and it stops being accepted as a tx fee denom; fee dust already collected in that denom follows the non-swappable path (fee collector → burner → black hole).

### How They’re Used
- `MsgCreateMarket` charges `create_market_fee` and forwards it to the community-pool fee collector after optional swap to `native_denom`.
- `MsgCreateOrder` decides maker vs. taker by whether the price crosses existing opposite liquidity; it charges the corresponding fee, applies queue-surcharge gas when the global queue counter is above `order_book_extra_gas_window`, and spends scan gas per queued message while checking prices.
- `MsgFillOrders` always uses the taker fee path and consumes `fill_orders_extra_gas` both at entry and per order it enqueues.
- Queue processing at `EndBlock` respects `order_book_per_block_messages`; remaining messages stay queued for later blocks.
- Module-level swaps/add-liquidity helpers refuse to run unless the relevant native/pair pool holds at least `min_native_liquidity_for_module_swap` in native reserves.
- `halted_denoms` is read by every order book and AMM message, by the `EndBlock` engine (halted-market messages are refunded instead of matched), by `swapTokens` (the single choke point of every swap) and by the liquidity answers the ante handler, the fee collector and the burner rely on (`HasLiquidityWithNativeDenom`, `HasDeepLiquidityWithNativeDenom`, `CanSwapForNativeDenom` all answer `false` for a halted denom).

### Updating
- Params are authority-only via `MsgUpdateParams`. All fields must be supplied when updating.
- Halting or un-halting a denom is therefore a `MsgUpdateParams` governance proposal carrying every current field plus the new `halted_denoms` list; see the governance runbook in [README.md](README.md#halted-denoms). Un-halting is the same proposal without the denom: resting orders resume matching and pools resume swapping, nothing else to do.

## Version History

### v8.2.0
- Added `halted_denoms` (default empty). No migration and no consensus version bump: a proto3 repeated field decodes as empty from the params already stored on chain.

### v8.1.0
- Fee fields (`create_market_fee`, `market_maker_fee`, `market_taker_fee`) changed from string to `sdk.Coin` for type safety
- Added `order_book_extra_gas_window` (default 100), `order_book_queue_extra_gas` (default 25,000), `fill_orders_extra_gas` (default 5,000), `order_book_queue_message_scan_extra_gas` (default 5,000), `order_book_per_block_messages` (default 500), `min_native_liquidity_for_module_swap` (default 100,000,000,000)
- Migration v3→v4 converts string fees to `sdk.Coin` and sets defaults for new parameters
