# TradeBin Parameters

- **`create_market_fee`** (`string coin`, default `25000000000ubze`): Charged to creators of new markets.
- **`market_maker_fee`** (`string coin`, default `1000ubze`): Fee applied to maker-side trades.
- **`market_taker_fee`** (`string coin`, default `100000ubze`): Fee applied to taker-side trades.
- **`maker_fee_destination`** (`community_pool` | `burner`, default `burner`): Where maker fees are sent.
- **`taker_fee_destination`** (`community_pool` | `burner`, default `burner`): Where taker fees are sent.
- **`native_denom`** (`string`, default `ubze`): Treated as the chain’s base denom for internal conversions.
- **Gas tuning:** `order_book_extra_gas_window`, `order_book_queue_extra_gas`, `fill_orders_extra_gas` add gas to heavy operations to protect block load.
- **`min_native_liquidity_for_module_swap`** (`Int`, default `100000000000`): Required native liquidity before module-initiated swaps are allowed.

### How They’re Used
- Fees are charged during order placement/fill and market creation, then routed to the configured destinations.
- Gas knobs are applied internally when queuing and executing order fills.
- The liquidity threshold prevents module-driven swaps when native liquidity is thin.

### Updating
- Params are authority-only via `MsgUpdateParams`. All fields must be supplied when updating.
