# TradeBin Parameters

- **`create_market_fee`** (`sdk.Coin`, default `25000000000ubze`): Charged during `MsgCreateMarket`; captured from the creator, swapped to `native_denom` if needed, and forwarded to the community-pool fee collector. Non-positive disables the charge.
- **`market_maker_fee`** (`sdk.Coin`, default `1000ubze`): Taken on `MsgCreateOrder` when the order **does not** immediately match opposite liquidity (maker path). Captured, optionally swapped to `native_denom`, then sent to `maker_fee_destination`.
- **`market_taker_fee`** (`sdk.Coin`, default `100000ubze`): Taken on `MsgCreateOrder` when the order crosses existing liquidity and on every `MsgFillOrders` call (taker path). Captured, optionally swapped to `native_denom`, then sent to `taker_fee_destination`.
- **`maker_fee_destination`** (`community_pool` | `burner`, default `burner`): Module that receives maker fees (community-pool collector or burner fee collector).
- **`taker_fee_destination`** (`community_pool` | `burner`, default `burner`): Module that receives taker fees.
- **`native_denom`** (`string`, default `ubze`): Base denom used when swapping collected fees or module balances; must exist in bank supply.
- **Gas controls:**
  - `fill_orders_extra_gas` (`uint64`, default `5000`): Gas charged once when `MsgFillOrders` starts and again for each entry, scaled by its index.
  - `max_fills_per_order` (`uint64`, default `500`, JSON `maxFillsPerOrder`, field 11): the most resting orders one order examines, filled or skipped (a resting order it can not fill without leaving that order below the minimum amount is skipped): one `MsgCreateOrder`, or one entry of `MsgFillOrders`. When the order has examined that many while more orders rest at its price, it stops and its remainder is refunded with reason `max_fills_reached`; the next order continues with the next resting order. It bounds the matching work, and so the gas, of one order. Must be at least 1. Before v8.3.0 field 11 was `order_book_per_block_messages`; the upgrade keeps its stored value (the default was 500 in both) and sets the default only over a 0.
  - `order_book_extra_gas_window` (`uint64`, default `100`) and `order_book_queue_extra_gas` (`uint64`, default `25000`): kept in the params (fields 7 and 8) but unused since orders execute inside their transaction; v8.3.0 gives them a new meaning in later changes.
  - Field 12 (`order_book_queue_message_scan_extra_gas`) was removed in v8.3.0 and is reserved.
- **`min_native_liquidity_for_module_swap`** (`Int`, default `100000000000`): Minimum native reserves required in a native/pair pool before module-driven swaps or add-liquidity helpers execute.

### How They’re Used
- `MsgCreateMarket` charges `create_market_fee` and forwards it to the community-pool fee collector after optional swap to `native_denom`.
- `MsgCreateOrder` decides maker vs. taker by whether the price crosses existing opposite liquidity and charges the corresponding fee. The matching it does in the transaction is metered by the transaction's gas; no param adds gas to it.
- `MsgFillOrders` always uses the taker fee path and consumes `fill_orders_extra_gas` at entry and per entry.
- `MsgCreateOrder` and each `MsgFillOrders` entry examine at most `max_fills_per_order` resting orders (filled or skipped).
- Module-level swaps/add-liquidity helpers refuse to run unless the relevant native/pair pool holds more than `min_native_liquidity_for_module_swap` in native reserves.

### Updating
- Params are authority-only via `MsgUpdateParams`. All fields must be supplied when updating.
- Halting a denom on the DEX is **not** a parameter. Halted denoms have their own store (one key per denom), edited by the `MsgHaltDenoms` / `MsgUnhaltDenoms` governance messages, so a halt proposal carries nothing but the denoms and the parameter set is untouched. See the halted denoms section in [README.md](README.md#halted-denoms).

## Version History

### v8.3.0
- Removed `order_book_queue_message_scan_extra_gas` (field 12, reserved): orders execute inside their transaction, there is no queue to scan. The params JSON no longer has the field.

### v8.1.0
- Fee fields (`create_market_fee`, `market_maker_fee`, `market_taker_fee`) changed from string to `sdk.Coin` for type safety
- Added `order_book_extra_gas_window` (default 100), `order_book_queue_extra_gas` (default 25,000), `fill_orders_extra_gas` (default 5,000), `order_book_queue_message_scan_extra_gas` (default 5,000), `order_book_per_block_messages` (default 500), `min_native_liquidity_for_module_swap` (default 100,000,000,000)
- Migration v3→v4 converts string fees to `sdk.Coin` and sets defaults for new parameters
