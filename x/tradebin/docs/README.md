# TradeBin Module – User Guide

TradeBin provides orderbook trading and AMM liquidity pools.

## Markets and Orders
- **Create a market** (`MsgCreateMarket`): define `base` and `quote` denoms; pays the create-market fee from params.
- **Place an order** (`MsgCreateOrder`): submit `order_type` (`buy`/`sell`), `amount` (base), `price`, and `market_id`. Maker/taker fees are applied per params.
- **Cancel an order** (`MsgCancelOrder`): remove your open order by `order_id` and `order_type`.
- **Fill orders** (`MsgFillOrders`): batch-fill existing orders at specific price/amount levels, typically run by keepers/relayers; earns maker/taker fees depending on side.

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
- Maker/taker fees and the create-market fee come from params.
- Fee destinations can be the community pool or burner module; pools also carry their own fee and destination.
