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

## Halted Denoms
Governance can halt a denom on the DEX through the `halted_denoms` parameter (see [params.md](params.md)). This exists for tokens that may become worthless — for example a bridged asset whose issuer is winding the bridge down — so that they are never exchanged for coins that still have value.

While a denom is halted:
- **Refused** (with `ErrDenomHalted`, before any fee or funds move): creating a market or pool with it, placing an order (`MsgCreateOrder`) or filling orders (`MsgFillOrders`) on a market whose base or quote is halted, adding liquidity to a pool that holds it, and any `MsgMultiSwap` whose route touches it — even as an intermediate hop. Paying tx fees in the denom is refused by the ante handler.
- **Refunded**: order book messages that were already queued when the halt took effect are refunded in full by `EndBlock` instead of being matched (typed event `QueueMessageRefundedEvent`, reason `market_halted`). This happens in the very block the proposal passes and for any backlog beyond `order_book_per_block_messages`. Refunded messages are not replayed after an un-halt; resubmit them.
- **Still working**: `MsgCancelOrder` (refund through the queue as always), `MsgRemoveLiquidity`, bank sends, IBC transfers, staking and rewards that merely hold the denom, `MsgFundBurner`. Resting orders and pool reserves are left exactly as they are — nothing is swept or force-cancelled.
- **Never swapped, by anyone**: user routes, module swaps (fee conversion, burner add-liquidity) and fee swaps all refuse the denom. Coins of a halted denom that other modules hold as fee dust are treated like any non-swappable IBC denom and end up locked in the burner black hole; tx-fee dust in that denom is distributed to stakers in kind.

A market or pool is halted iff its base **or** quote is halted. Both sides of a book are frozen: every fill has one party receiving the halted denom, so there is no "exit-only" direction on an order book. Un-halting is the same proposal without the denom — resting orders resume matching and pools resume swapping.

### Governance runbook (halting a denom)
`MsgUpdateParams` replaces the **whole** `Params` object, so the proposal must carry every current field value plus the new list.

1. Query the current params and copy every field: `bzed query tradebin params` (or `GET https://rest.getbze.com/bze/tradebin/params` on mainnet).
2. Write the proposal with a single `/bze.tradebin.MsgUpdateParams` message: `authority` = the gov module account (mainnet `bze10d07y265gmmuvt4z0w9aw880jnsr700j8xlwyy`), `params` = the copied values plus `halted_denoms: ["<denom>"]`. Submit it with `bzed tx gov submit-proposal proposal.json`.
3. Rehearse on the testnet first with a factory denom that has both a market and a pool: submit, vote, then verify each effect listed above with real transactions (orders and fills refused, queued messages refunded, cancel and remove-liquidity working, swaps through the pool refused, the denom refused as a fee denom).
4. Reverting is the same proposal without the denom.

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

### v8.2.0
- Governance-halted denoms: `halted_denoms` param (ships empty), `ErrDenomHalted` on every new-position message touching a halted market or pool, EndBlock refunds of queued messages on halted markets (`QueueMessageRefundedEvent`), the no-swap invariant at `swapTokens`, and halted denoms refused as tx fee denoms. Cancel and remove-liquidity untouched.

### v8.1.0
- Fee payer service (`CaptureAndSwapUserFee`) for fee capture and conversion to native denom via liquidity pools
- Queue-based order processing with bounded EndBlock execution, capped at 500 messages/block (`OrderBookPerBlockMessages`)
- Dynamic gas surcharges for spam protection based on queue depth via 6 new gas/liquidity parameters
- Duplicate cancel prevention: pending cancel requests tracked via `HasPendingCancel` to prevent duplicates
- Queue message keys restructured to composite `{market}/{id}` format for market-scoped queries
- Order key precision migration: keys migrated from 24-char/10-decimal to 32-char/18-decimal format (module migration v3→v4)
- Fee fields (`create_market_fee`, `market_maker_fee`, `market_taker_fee`) migrated from string to `sdk.Coin` type (v2 parameters)
- `min_native_liquidity_for_module_swap` param replaces hardcoded liquidity threshold
- `FillOrders` gas consumption now uses `FillOrdersExtraGas` param instead of hardcoded constant
- ConsensusVersion bumped from 3 to 4
