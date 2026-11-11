# TradeBin Module – User Guide

TradeBin provides orderbook trading and AMM liquidity pools.

## Markets and Orders
- **Create a market** (`MsgCreateMarket`): define `base` and `quote` denoms; charges `create_market_fee` which is swapped to `native_denom` if needed and sent to the community-pool collector.
- **Place an order** (`MsgCreateOrder`): submit `order_type` (`buy`/`sell`), `amount` (base), `price`, and `market_id`. Maker vs. taker fee is chosen by whether the order immediately crosses existing opposite liquidity. The order executes inside the transaction: it is matched against the resting orders at exactly its price (oldest first), and what is left rests on the book as a new order when the opposite side at that price was emptied and the remainder is at least the minimum amount; otherwise the remainder is refunded in the same transaction. One order examines at most `max_fills_per_order` resting orders, filled or skipped (default 500); if more orders are left at its price, the remainder is refunded and a new order continues with them. Placing at a price where you already have an order (same market, side and price) adds to it: the new order is matched first, then what is left is merged with your resting order (no second order is created). The merged order takes the new order's id and goes to the back of the queue at its price, behind everyone already resting there: topping up gives up your place, and your old order id no longer exists. A remainder that would be refunded for a new order (below the minimum amount, or opposite orders left at the price) is refunded the same way and your resting order is untouched. Each placement still pays its own maker or taker fee.
- **Cancel an order** (`MsgCancelOrder`): remove your open order by `order_id` and `order_type`. The remaining amount is refunded in the same transaction.
- **Fill orders** (`MsgFillOrders`): fill existing orders at specific price/amount levels in one message. Each entry is an immediate-or-cancel order executed in the transaction: what it cannot fill is refunded, it never rests. The message always pays the taker fee once (after its entries ran) and consumes `fill_orders_extra_gas` once on entry and again per entry.
- **Order ids**: every order gets an id when it is processed, including a taker filled in full that never rests. Fills, refunds and the resting remainder of an order carry that id in their events (`order_id`).
- **Gas**: the matching work (every maker filled, its settlement, history and events) is metered by the transaction's gas, like any other store access. A transaction that runs out of gas, or fails for any other reason, is reverted whole: nothing is filled, escrowed or charged but the gas. Wallets simulate to estimate it; a taker crossing many resting orders uses more gas than one filling a single order. Crowded price levels cost more gas for new makers: placing an order at a price level of your side where more than `price_level_owners_window` other owners already rest (default 500), and where you have no order yet, adds `price_level_extra_gas` (default 25,000) for each owner above that number. Topping up your own order costs nothing extra, and no order is ever refused. The extra gas is consumed before the fee and the escrow, so a placement that runs out of gas costs only the gas.

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
Governance can halt a denom on the DEX with a `MsgHaltDenoms` proposal, and lift the halt with `MsgUnhaltDenoms`. This exists for tokens that may become worthless — for example a bridged asset whose issuer is winding the bridge down — so that they are never exchanged for coins that still have value. Halted denoms live in their own store, one key per denom (not in the module params): checking a denom is a single store lookup whatever the number of halted denoms, there is no cap on how many can be halted, and a halt proposal carries nothing but the denoms.

While a denom is halted:
- **Refused** (with `ErrDenomHalted`, before any fee or funds move): creating a market or pool with it, placing an order (`MsgCreateOrder`) or filling orders (`MsgFillOrders`) on a market whose base or quote is halted, adding liquidity to a pool that holds it, and any `MsgMultiSwap` whose route touches it — even as an intermediate hop. Paying tx fees in the denom is refused by the ante handler.
- Orders execute in their own transaction, so from the block a halt takes effect the market is frozen except for cancels and liquidity removals.
- **Still working**: `MsgCancelOrder` (refunded in its transaction as always), `MsgRemoveLiquidity`, bank sends, IBC transfers, staking and rewards that merely hold the denom, `MsgFundBurner`. Resting orders and pool reserves are left exactly as they are — nothing is swept or force-cancelled.
- **Never swapped, by anyone**: user routes, module swaps (fee conversion, burner add-liquidity) and fee swaps all refuse the denom. Coins of a halted denom that other modules hold as fee dust are treated like any non-swappable IBC denom and end up locked in the burner black hole; tx-fee dust in that denom is distributed to stakers in kind.

A market or pool is halted iff its base **or** quote is halted. Both sides of a book are frozen: every fill has one party receiving the halted denom, so there is no "exit-only" direction on an order book. Un-halting lets resting orders resume matching and pools resume swapping — nothing else to do.

### Halt queries
- `bzed query tradebin halted-denoms` (REST `GET /bze/tradebin/halted_denoms`, paginated) — every halted denom, in store order.
- `bzed query tradebin denom-halted <denom>` (REST `GET /bze/tradebin/denom_halted?denom=<denom>`) — `halted: true|false` for one denom. Over REST the denom goes in the query string because ibc/factory denoms contain `/`.
- Indexers can follow the `DenomHaltedEvent` / `DenomUnhaltedEvent` typed events, emitted once per denom whose state actually changed.

### Governance runbook (halting a denom)
1. Write the proposal with a single `/bze.tradebin.MsgHaltDenoms` message: `authority` = the gov module account (mainnet `bze10d07y265gmmuvt4z0w9aw880jnsr700j8xlwyy`), `denoms` = the denoms to halt — valid denoms, no duplicates, up to 1000 per message, never the native denom. No other field and no other parameter is involved.
   ```json
   {
     "messages": [
       {
         "@type": "/bze.tradebin.MsgHaltDenoms",
         "authority": "bze10d07y265gmmuvt4z0w9aw880jnsr700j8xlwyy",
         "denoms": ["ibc/6490A7EAB61059BFC1CDDEB05917DD70BDF3A611654162A1A47DB930D40D8AF4"]
       }
     ],
     "metadata": "<ipfs or url>",
     "deposit": "<amount>ubze",
     "title": "Halt USDC.n on the DEX",
     "summary": "<why>"
   }
   ```
   Submit it with `bzed tx gov submit-proposal proposal.json --from <key>`.
2. Rehearse on the testnet first with a factory denom that has both a market and a pool: submit, vote, then verify each effect listed above with real transactions (orders and fills refused, cancel and remove-liquidity working, swaps through the pool refused, the denom refused as a fee denom) and check that `denom-halted` answers `true`.
3. The message is idempotent: a denom that is already halted is left as is, so a batch proposal never fails because one of its entries was halted by an earlier proposal. Listing the native denom fails the whole proposal and halts nothing.
4. Reverting is a `/bze.tradebin.MsgUnhaltDenoms` proposal with the same `denoms`. Un-halting a denom that is not halted is a no-op.

## Changing a Pool's Fees
Pools are unique per pair (the pool ID is the sorted `base_quote`), so a pool's swap fee and fee split can't be changed by opening a second pool. Governance changes them instead, with a `MsgUpdatePools` proposal. Liquidity never moves.

- **What changes**: only `fee` and `fee_dest`. The curve (`stable`), the reserves, the LP denom, the creator and the LP shares stay as they are. The new values follow the pool creation rules: the fee is from 0.001 (0.1%) to 0.05 (5%), the three `fee_dest` shares (`treasury`, `burner`, `providers`) are non-negative and add up to exactly 1.
- **When**: at once. The next swap charges the new fee and splits it with the new destinations. No migration, and LP share accounting doesn't change.
- **All or nothing**: 1 to 100 pools per message, no pool twice. If any listed pool doesn't exist, the proposal fails at execution and no pool changes.
- **Halted pools** can be updated: the halt only blocks trading.
- **Idempotent**: an entry equal to the pool's current values is accepted and writes nothing. Every pool that does change emits `PoolFeesUpdatedEvent` with `pool_id`, `old_fee`, `new_fee`, `old_fee_dest` and `new_fee_dest`.
- There is no fast path for the pool creator or a token admin: every change, decreases included, goes through governance.

Example proposal (`bzed tx gov submit-proposal proposal.json --from <key>`; the fee and the shares are decimals given as strings):
```json
{
  "messages": [
    {
      "@type": "/bze.tradebin.MsgUpdatePools",
      "authority": "bze10d07y265gmmuvt4z0w9aw880jnsr700j8xlwyy",
      "updates": [
        {
          "pool_id": "<base>_<quote>",
          "fee": "0.003",
          "fee_dest": { "treasury": "0.25", "burner": "0.25", "providers": "0.5" }
        }
      ]
    }
  ],
  "metadata": "<ipfs or url>",
  "deposit": "<amount>ubze",
  "title": "Lower the fee of the <pair> pool",
  "summary": "<why>"
}
```
Query the pool with `bzed query tradebin liquidity-pool <id>` before and after the vote.

## Order Book Engine Events
`MsgCreateOrder`, `MsgFillOrders` and `MsgCancelOrder` execute in the transaction, so their outcome events are in the transaction result (not in the block's `finalize_block_events`). Every order the engine processes emits at least one of the events below, carrying the order's id in `order_id`:

| Event | When |
| --- | --- |
| `OrderExecutedEvent` | once per resting (maker) order a create or fill order matched. `id` is the maker order, `order_id` the taker order |
| `OrderSavedEvent` | a create order (or its remainder) rests on the book. `id` and `order_id` are the same |
| `OrderAmendedEvent` | a create order (or its remainder) was merged with the order its owner already rests at the same market, side and price, instead of being saved. The merged order rests under `id`, the create order's own id (which its other events carry too), at the back of the price level; `previous_id` is the order it replaced, which no longer exists; `added_amount` is the base amount added and `amount` the merged order's amount |
| `OrderMessageRefundedEvent` | the unexecuted part of an order is returned. `reason`: `no_match` (fill, nothing at its price), `unfilled_remainder` (fill, partly matched), `below_min_amount` (create, the remainder was not placed: below the minimum amount, or opposite orders were left at the price), `max_fills_reached` (create or fill, the order examined `max_fills_per_order` resting orders, filled or skipped, and more rest at its price), `upgrade` (a message still queued at the v8.3.0 upgrade, refunded by the module migration; its `order_id` is the old queue message id). `amount` is the base amount returned, `refund` the coin sent (empty when it rounded to zero and only dust was stored) |
| `OrderCanceledEvent` | a cancel removed its order and refunded it. `id` and `order_id` are the cancelled order |

So per message: a create ends with executed, saved (or amended) and/or refunded events; each fill entry with executed and/or refunded events; a cancel with a canceled event. A cancel of an order that does not exist (filled or cancelled meanwhile) is refused with `order not found` and the transaction reverts.

## Pool Price Average
Every pool keeps an exponential moving average of its price, `quote reserve / base reserve` in the pool's own base/quote orientation (a pool with a zero reserve, or a price that rounds to 0 at 18 decimals, has no price). It lets later features compare a pool's current price with where it has been over the last blocks.

- The average is updated in one place only: the first time a pool is saved in a block. The price stored at that moment is the price the pool had at the start of the block, and it is folded in for all blocks since the previous update. Trades inside the current block never reach the average.
- The fold is `avg = price + (avg − price) × 0.5^(elapsed / pool_price_half_life_blocks)`. After `pool_price_half_life_blocks` blocks the average has covered half the distance to the price. A gap above 20 half lives means the average is simply the stored price.
- A pool's first save starts the average at the price of the pool being saved. The v8.3.0 upgrade seeds every existing pool with its price at the upgrade height.
- Records are not part of the genesis export. Importing a genesis seeds them again through the pool saves.
- Params: `pool_price_half_life_blocks` (default 20), `fee_max_price_drift` (default 0.10) and `module_max_price_drift` (default 0.03), see [params.md](params.md). The two drift limits are stored here and used by the checks that compare a pool's price with its average.
- Query: `bzed query tradebin pool-price-average <pool-id>`, REST `/bze/tradebin/pool_price_average?pool_id=<id>` (the id is a query parameter because pool ids contain `/`). It returns the average brought up to the current block and the height at which the record was last written.

## User Dust
Partial order fills can leave fractional coin amounts (dust) that are too small to settle. The module tracks dust per user address, and it accumulates across trades.

## Queries
- `bzed query tradebin params` – current fees and gas tuning knobs.
- `bzed query tradebin market <id>` / `markets` – market listings.
- `bzed query tradebin asset-markets <asset>` – all markets where an asset is base or quote.
- `bzed query tradebin user-market-orders <address> --market <id>` – paginated user orders in a market.
- `bzed query tradebin market-aggregated-orders <market> <buy|sell>` – aggregated order book at price levels, with the number of distinct owners resting at each (`owners`). At a price where one owner rests several orders placed before v8.3.0, `owners` can count fewer owners than rest there, even 0 on a level that still holds orders, once the earliest of those orders was cancelled or filled in full; the crowded-level gas there is lower accordingly.
- `bzed query tradebin market-history <market>` – execution history for a market.
- `bzed query tradebin market-order <market> <buy|sell> <order-id>` – single order details.
- `bzed query tradebin all-user-dust <address>` – fractional dust from partial fills.
- `bzed query tradebin liquidity-pool <id>` / `liquidity-pools` – pool details and LP supply.
- `bzed query tradebin halted-denoms` / `denom-halted <denom>` – the governance-halted denoms (paginated) and the flag for one denom.

## Fees and Destinations
- **Create-market fee** (`create_market_fee`): always routed to community pool via `txfeecollector`.
- **Order fees** (maker/taker): routed based on `maker_fee_destination`/`taker_fee_destination` params — valid destinations are `community_pool` or `burner`.
- **Pool fees** (`fee` field per pool): split three ways via `fee_destination` — `treasury` % to the community pool (the token's own community pool when the coin is a factory token with one set, see the txfeecollector docs; the chain's otherwise), `burner` % to burner module, `providers` % to LP holders.
- Fees are captured from the sender, swapped to `native_denom` when possible, and forwarded to the destination module.
- **Factory token transfer fee on protocol fees:** when the sender pays a protocol fee (the denom, market, DAO, staking, trading and denom reward creation fees, the denom reward schedule and prize fees, and the order fees) in a factory token whose tokenfactory `DenomConfig` sets a `transfer_fee`, they pay `ceil(fee in the token × transfer_fee)` on top. It goes to `txfeecollector_cp`, forwarded to the token's community pool at EndBlock, or straight to the `burner` module, which burns it, when the token has no community pool. The balance check covers the fee plus the transfer fee; a sender short of it pays the protocol fee in `native_denom` instead, as before. The rate is read from the tokenfactory keeper, which the app sets on this keeper after building it; without it, nothing is charged and a warning is logged.
- `MsgUpdateParams`, `MsgHaltDenoms`, `MsgUnhaltDenoms` and `MsgUpdatePools` are restricted to the module authority (governance).
- Module-level swaps/add-liquidity helpers refuse to run unless the native/pair pool holds more than `min_native_liquidity_for_module_swap` in native reserves.

## Version History

### v8.3.0
- Pool price moving average, three new params and a query (see above).
- Order book orders execute inside their transaction, metered by its gas: the EndBlock queue, its counter, the pending-cancel marker, the queue gas surcharge, the queue price scan and the `order_book_queue_message_scan_extra_gas` param are removed. Every order gets an id before matching, and `order_id` replaces `message_id` in the order events; `OrderCreateMessageEvent`, `OrderCancelMessageEvent`, `OrderCancelSkippedEvent` and `OrderMessageFailedEvent` are removed. The upgrade refunds every message still queued (reason `upgrade`).
- One order examines at most `max_fills_per_order` resting orders, filled or skipped (param field 11, renamed from `order_book_per_block_messages`, default 500); the remainder is refunded with reason `max_fills_reached`.
- One owner has at most one order per market, side and price: a new order at a price where its owner already rests one is merged with that order. The merged order takes the new order's id and goes to the back of the queue at its price, so topping up an order gives up its place; the old id is gone (`OrderAmendedEvent` with `previous_id`, a cancel by the old id is refused with `order not found`). Orders placed before v8.3.0 stay as they are; new orders at their price are merged with the earliest of them.
- Each price level counts its distinct owners (`owners` on the aggregated order book). A new owner joining a level of more than `price_level_owners_window` owners pays `price_level_extra_gas` per owner above it. The params `order_book_extra_gas_window` and `order_book_queue_extra_gas` are renamed `price_level_owners_window` (set to 500 by the upgrade) and `price_level_extra_gas` (value kept).
- The factory token transfer fee is charged on protocol fees paid in the token (fee payer service).
- `MsgUpdatePools` governance message: changes the swap fee and fee split of existing pools (all or nothing, halted pools included, idempotent), `PoolFeesUpdatedEvent` per changed pool.
- Order book engine outcome events: `OrderMessageRefundedEvent` (kept), `OrderCancelSkippedEvent` and `OrderMessageFailedEvent` (removed again with the queue, see above).

### v8.2.0
- Governance-halted denoms: `MsgHaltDenoms` / `MsgUnhaltDenoms` (authority only, idempotent), the `halted-denoms` and `denom-halted` queries, `ErrDenomHalted` on every new-position message touching a halted market or pool, the no-swap invariant at `swapTokens`, and halted denoms refused as tx fee denoms. Cancel and remove-liquidity untouched. Nothing is halted at the upgrade.

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
