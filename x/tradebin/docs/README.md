# TradeBin Module – User Guide

TradeBin provides orderbook trading and AMM liquidity pools.

## Markets and Orders
- **Create a market** (`MsgCreateMarket`): define `base` and `quote` denoms; charges `create_market_fee` which is swapped to `native_denom` if needed and sent to the community-pool collector.
- **Place an order** (`MsgCreateOrder`): submit `order_type` (`buy`/`sell`), `amount` (base), `price`, and `market_id`. Maker vs. taker fee is chosen by whether the order immediately crosses existing opposite liquidity. If the queue depth passes `order_book_extra_gas_window`, extra gas is consumed per `order_book_queue_extra_gas`; validating prices also spends `order_book_queue_message_scan_extra_gas` per queued message.
- **Cancel an order** (`MsgCancelOrder`): remove your open order by `order_id` and `order_type`.
- **Fill orders** (`MsgFillOrders`): batch-fill existing orders at specific price/amount levels, typically run by keepers/relayers; always pays the taker fee and consumes `fill_orders_extra_gas` once on entry and again per order enqueued.

Example:
```bash
bzed tx tradebin create-market ubze ibc/xyz --from mykey
bzed tx tradebin create-order buy 1000000 0.5 <market-id> --from mykey
bzed tx tradebin cancel-order <market-id> <order-id> buy --from mykey
```

## Liquidity Pools and Swaps
- **Create a pool** (`MsgCreateLiquidityPool`): set base/quote, pool fee, fee destination, whether it’s stable, and supply initial base/quote liquidity. Returns `pool_id`.
- **Add/Remove liquidity** (`MsgAddLiquidity` / `MsgRemoveLiquidity`): deposit both sides to mint LP shares or redeem LP shares back to coins.
- **MultiSwap** (`MsgMultiSwap`): route a swap through multiple pools; supply `routes`, `input`, and `min_output`.

Example:
```bash
# Create a pool with 1% fee, sending fees to burner
bzed tx tradebin create-liquidity-pool \
  ubze ibc/xyz 0.01 burner false 1000000 2000000 --from mykey

# Add liquidity
bzed tx tradebin add-liquidity <pool-id> 500000 1000000 1 --from mykey

# Swap across a route
bzed tx tradebin multi-swap '["<pool1>","<pool2>"]' \
  --input 1000000ubze --min-output 900000ibc/xyz --from mykey
```

## Queries
- `bzed query tradebin params` – current fees and gas tuning knobs.
- `bzed query tradebin market <id>` / `markets` – market listings.
- `bzed query tradebin liquidity-pool <id>` / `liquidity-pools` – pool details and LP supply.
- Order and queue queries are also exposed via gRPC/REST for explorers/relayers.

## Fees and Destinations
- Maker/taker fees and the create-market fee come from params. Create-market fees always go to the community-pool collector; maker/taker fees are routed to either the community-pool collector or burner fee collector based on `maker_fee_destination`/`taker_fee_destination`.
- Fees are captured from the sender, swapped to `native_denom` when possible, and then forwarded to the destination module. Pools also carry their own fee and destination.
- Queue processing at `EndBlock` is capped by `order_book_per_block_messages`; messages beyond the cap remain queued for later blocks, and the queue counter resets only after the queue is emptied.
- Module-level swaps/add-liquidity helpers refuse to run unless the native/pair pool holds at least `min_native_liquidity_for_module_swap` in native reserves.
