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
- **Create a pool** (`MsgCreateLiquidityPool`): set base/quote, pool fee, fee destination, and supply initial base/quote liquidity. Returns `pool_id`. The initial LP tokens are permanently locked (sent to the burner `black_hole` account) and cannot be recovered.
- **Add/Remove liquidity** (`MsgAddLiquidity` / `MsgRemoveLiquidity`): deposit both sides to mint LP shares or redeem LP shares back to coins. Single-side deposits are automatically balanced via optimal swap calculation.
- **MultiSwap** (`MsgMultiSwap`): route a swap through up to 5 pools; supply `routes`, `input`, and `min_output`. Always pays the taker fee.

### LP Token Details
- **Denom format**: `lp/<base>/<quote>` (e.g., `lp/ubze/ibc/xyz`).
- **Precision**: LP tokens are scaled by 10^18 to preserve accuracy.
- **Initial lock**: the first LP tokens minted at pool creation are permanently sent to the burner module’s black hole account — they can never be redeemed. This bootstraps the pool permanently.

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

## User Dust
Partial order fills can leave fractional coin amounts (dust) that are too small to settle. The module tracks dust per user address, and it accumulates across trades.

## Queries
- `bzed query tradebin params` – current fees and gas tuning knobs.
- `bzed query tradebin market <id>` / `markets` – market listings.
- `bzed query tradebin asset-markets <asset>` – all markets where an asset is base or quote.
- `bzed query tradebin user-market-orders <address> --market <id>` – paginated user orders in a market.
- `bzed query tradebin market-aggregated-orders <market> <buy|sell>` – aggregated order book at price levels.
- `bzed query tradebin market-history <market>` – execution history for a market.
- `bzed query tradebin market-order <market> <buy|sell> <order-id>` – single order details.
- `bzed query tradebin all-user-dust <address>` – fractional dust from partial fills.
- `bzed query tradebin liquidity-pool <id>` / `liquidity-pools` – pool details and LP supply.

## Fees and Destinations
- **Create-market fee** (`create_market_fee`): always routed to community pool via `txfeecollector`.
- **Order fees** (maker/taker): routed based on `maker_fee_destination`/`taker_fee_destination` params — valid destinations are `community_pool` or `burner`.
- **Pool fees** (`fee` field per pool): split three ways via `fee_destination` — `treasury` % to community pool, `burner` % to burner module, `providers` % to LP holders.
- Fees are captured from the sender, swapped to `native_denom` when possible, and forwarded to the destination module.
- `MsgUpdateParams` is restricted to the module authority (governance).
- Queue processing at `EndBlock` is capped by `order_book_per_block_messages`; messages beyond the cap remain queued for later blocks, and the queue counter resets only after the queue is emptied.
- Module-level swaps/add-liquidity helpers refuse to run unless the native/pair pool holds at least `min_native_liquidity_for_module_swap` in native reserves.

## Version History

### v8.1.0
- Added queue-based order processing: orders processed asynchronously at EndBlock, capped at 500 messages/block (`OrderBookPerBlockMessages`)
- Added dynamic gas surcharges for spam protection based on queue depth
- Added `MsgFillOrders`: batch-fill up to 50 orders at specific price/amount levels in one transaction (always taker fee)
- Added `MsgMultiSwap`: multi-hop swaps through up to 5 liquidity pool routes in a single transaction
- AMM enhancements: optimal swap calculation for single-token liquidity provision, `ModuleSwapForNativeDenom`, `ModuleAddLiquidityWithNativeDenom` with automatic balancing and dust handling
- Deep liquidity checks: `HasDeepLiquidityWithNativeDenom` ensures sufficient pool depth before swaps
- Duplicate cancel prevention: pending cancel requests tracked to prevent duplicate cancellations
- Order key precision migration: keys migrated from 24-char/10-decimal to 32-char/18-decimal format (module migration v3→v4)
- Fee fields migrated from string to `sdk.Coin` type; 6 new gas/liquidity parameters added
