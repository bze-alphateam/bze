# TxFeeCollector Module – Operator Guide

TxFeeCollector standardizes fees by converting module balances to the native denom and enforcing a minimum gas price.

## What It Does
- **Fee conversion:** Swaps non-native fees held by:
  - the main fee collector,
  - the community pool fee collector,
  into the native denom via the trade module when possible. See [Collector rules](#collector-rules).
- **Token community pools:** In its EndBlock pass over `txfeecollector_cp`, the module forwards every coin there whose denom has a community pool address (set by the factory token's admin in its `DenomConfig`, `MsgSetDenomConfig` in x/tokenfactory) to that address. Each forwarded coin emits a `CommunityPoolFeePaidEvent` (denom, recipient, amount). See [Community pool fees](#community-pool-fees).
- **Token gas surcharges:** Charges the gas surcharge a factory token's admin sets in its `DenomConfig`, into `txfeecollector_cp` (or straight to the `burner` module when the token has no community pool). See [Transfer fee and gas surcharge](#transfer-fee-and-gas-surcharge).
- **Minimum gas price:** Ante handler enforces a per-validator minimum gas price, with cross-denom support using spot prices from the trade module.
- **Inbound IBC filter:** Refuses incoming ICS-20 transfers listed in the `blocked_ibc_inbound` param with an error acknowledgement, so the sending chain refunds the sender and no voucher is minted here. Exits are untouched — sends, acknowledgements, timeout refunds and the channel lifecycle all behave as if the filter were not there. Every refusal emits a `BlockedIbcInboundEvent` typed event (channel, denom, amount, sender, receiver) so relayers and indexers can see what was bounced.
- **Governance:** Parameters can be updated via `MsgUpdateParams` (authority only).

## Module Accounts
The module registers two dedicated accounts:
- **`txfeecollector`** – holds non-native fee tokens awaiting conversion to native denom.
- **`txfeecollector_cp`** – holds fees designated for the community pool via the distribution module.

At EndBlock, collected fees are converted to native denom and distributed to their respective destinations.

## Collector rules
`txfeecollector` (gas fees paid in non-native coins, for the validators):
- ubze → the validators (the SDK fee collector).
- swappable (`CanSwapForNativeDenom`: a deep enough ubze pool, a swap fee that doesn't round to 0 when the pool charges one, and a swap output of at least 1 ubze) → swapped to ubze → the validators.
- a deep pool but not swappable now (too small, or the pool fee rounds to 0) → kept in the collector until enough builds up to swap it. This is common for gas fees in tokens worth more than 1 ubze per unit, so they are never burned.
- no pool or a shallow one → forwarded as-is to the SDK fee collector, so the validators receive the token.

`txfeecollector_cp` (the community pool collector); every coin reaches a community pool:
- a denom with its own community pool → a bank send to that address (see [Community pool fees](#community-pool-fees)).
- ubze → the chain's community pool.
- swappable → swapped to ubze → the chain's community pool.
- anything else (too small, the pool fee rounds to 0, no pool or a shallow one) → kept in the collector until it becomes swappable. Nothing goes to the burner from this collector, except LP shares: they can never be swapped, so they go to the `burner` module, which locks them.

### Bounded passes and the cursor
Each collector has a cursor in the module store: the last denom its pass looked at. Each EndBlock pass reads at most `max_balance_iterations` balances of the collector straight from the bank store, starting strictly after the cursor, in denom order, and moves the cursor to the last one. Every coin read counts towards the bound, whether it is swapped, sent, kept or skipped. A page shorter than the bound reached the end of the balances: the cursor is cleared and the next pass starts from the first denom. A cursor whose denom has left the collector still works, the read starts after it; one past the last denom gives an empty page, and the same pass reads again from the first denom. So coins kept to wait can never starve the coins after them: every denom is looked at within `balances / max_balance_iterations + 1` blocks. A failed pass moves no cursor. The cursor is derived state: it has no genesis field and an absent cursor means "start from the first denom".

## Community pool fees
`txfeecollector_cp` receives the `treasury` share of AMM pool fees (in the coin being sold), the tradebin create fees (already swapped to ubze), the DAO creation fee when it is set to go there, and order book fees too small to swap. Each EndBlock one pass over a page of its balances (see [Bounded passes and the cursor](#bounded-passes-and-the-cursor)) decides per coin, following the [collector rules](#collector-rules): every coin whose denom has a community pool in its tokenfactory `DenomConfig` is sent to that address (by denom only, whoever sent the coin), the rest funds the chain's community pool through x/distribution, swapped to ubze, or waits.

The pass is committed as one unit. If any send to a denom's community pool fails, the error is logged and the whole pass is discarded, sends already made in it and the cursor move included. Such a send can only fail on a blocked recipient (refused by `MsgSetDenomConfig`) or a missing balance (impossible, the collector sends what it holds), so a failure is a bug: the coins stay in `txfeecollector_cp`, they are never rerouted, swapped or burned, and they wait for the fix in an upgrade. A failed swap discards the pass the same way. It never fails the EndBlock.

So a factory token's community pool earns the treasury share of the pool fees paid in that token, that is on sells of the token; a buy pays its share in the other coin. Only factory tokens can have a community pool; ubze and IBC coins always fund the chain's community pool.

## Transfer fee and gas surcharge
A factory token's admin can set two rates in its `DenomConfig` (`MsgSetDenomConfig` in x/tokenfactory, see its [Token configuration](../../tokenfactory/docs/README.md#token-configuration)): a **transfer fee** (0 to 5%) and a **gas surcharge** (0 to 200%). ubze, IBC coins, LP shares and factory tokens without rates are never charged and cost no extra store read. A rate change, increase included, applies from the next tx: the caps are the holders' protection.

**Where the fees go.** When the token has a community pool, both fees go to `txfeecollector_cp`, and the [routing](#community-pool-fees) forwards them to the token's community pool at EndBlock (one `CommunityPoolFeePaidEvent` per forwarded coin). When it has none, they go straight to the `burner` module, which burns the factory token as is.

### Transfer fee
For now the transfer fee is charged only on **protocol fees paid in the token**, by x/tradebin's fee payer (see the [tradebin docs](../../tradebin/docs/README.md#fees-and-destinations)): the denom, market, DAO, staking, trading and denom reward creation fees, the denom reward schedule and prize fees, and the order fees. The payer pays `ceil(fee in the token × transfer_fee)` on top; a payer short of it pays the protocol fee in ubze instead. Plain sends between accounts, IBC transfers and every other module flow are not charged.

### Gas surcharge
Charged when the token **pays the tx fee**. Gas never pays the transfer fee, and protocol fees never pay the surcharge.

- **Minimum gas price** (CheckTx only): the token's price, converted from the ubze minimum gas price through its ubze pool, is multiplied by `1 + gas_surcharge` before the comparison with the validator's own price. A spot-price failure keeps the ubze fallback as before.
- **Split:** the whole fee goes to the `txfeecollector` module as before, then `floor(fee × s / (1 + s))` moves on to the token's destination, and at least 1 unit when the fee is above 1 (a fee of 1 stays whole with the validators, who always keep at least 1); the rest is converted for the validators. The split runs on the fee actually paid: DeliverTx doesn't check the minimum gas price, so a validator that includes a cheaper tx gives the community pool the same share of a smaller fee.
- Fee grants are charged the full declared fee, surcharge included; the granter pays it. The surcharge stays charged when the tx fails, like the rest of the fee.
- Every charged coin emits `GasSurchargePaidEvent{payer, denom, amount, destination}`.

## Ante Handler
Two decorators run in the transaction processing pipeline:
- **`ValidateTxFeeDenomsDecorator`**: enforces single-denomination fees per transaction and validates that non-native fee denoms have sufficient liquidity via the trade module.
- **`DeductFeeDecorator`**: splits fees by denom — native fees go directly to `auth.FeeCollectorName` (validators), non-native fees go to the `txfeecollector` module for later conversion, less a factory token's [gas surcharge](#gas-surcharge). Supports cross-denom minimum gas price calculation using trade module spot prices.

## User-Facing Surface
Most users don’t interact directly; the module runs in the background. Operators/governance may:
- Set/inspect params via `MsgUpdateParams` and `bzed query txfeecollector params`.
- Run fee-conversion hooks (called internally by the app) if building custom tooling.

Example governance-style update:
```bash
bzed tx gov submit-proposal update-params txfeecollector \
  --validator-min-gas-fee 0.02ubze \
  --from mykey
```
Adjust flags to your governance CLI; the message fields are just `validator_min_gas_fee` and `authority`.

## Version History

### v8.3.0
- Coins in `txfeecollector_cp` whose factory denom has a community pool address (x/tokenfactory `DenomConfig`) are forwarded to it in EndBlock, in the same pass that converts the rest for the chain's community pool
- Added the `CommunityPoolFeePaidEvent` typed event (`proto/bze/txfeecollector/events.proto`), one per forwarded coin
- Collector passes are bounded and cursor-based (see [Bounded passes and the cursor](#bounded-passes-and-the-cursor)): each reads at most `max_balance_iterations` balances after its cursor straight from the bank store instead of loading the whole account, so coins kept to wait can't starve the ones after them
- `txfeecollector_cp` is handled in one pass: a denom with a community pool is sent there, the rest funds the chain's community pool, swapped to ubze, or waits until it can be swapped. Only LP shares go to the burner. A failed send leaves every coin in place instead of falling back to swapping or burning coins owed to a token's community pool
- Two new store keys hold the collector cursors (`collector_cursor/<module>`); no migration, no genesis field; ConsensusVersion stays 3
- Factory tokens' gas surcharge (ante handler) is charged into `txfeecollector_cp`, or straight to the `burner` module for tokens without a community pool
- Burn-bound fees (the token transfer fee and gas surcharge, the order book fees, the DAO creation fee, proposal costs and kept deposits) go straight to the `burner` module instead of `txfeecollector_burner`: factory tokens are burned as is instead of being sold for ubze first
- Added the `GasSurchargePaidEvent` typed event
- Removed the `txfeecollector_burner` module account and its EndBlock conversion. The upgrade handler sends whatever the address still holds to the `burner` module. The old account stays in auth state and its address is no longer blocked: coins sent there by mistake stay there

### v8.2.0
- Added the inbound IBC transfer filter (`ibcmiddleware`), wrapping the transfer application inside the ICS-29 fee middleware, driven by the new `blocked_ibc_inbound` parameter
- Added the `BlockedIbcInboundEvent` typed event (`proto/bze/txfeecollector/events.proto`), emitted for every refused inbound packet
- ConsensusVersion bumped from 2 to 3

### v8.1.0
- Added three-way fee distribution: fees split between validators/stakers, burner (`txfeecollector_burner`), and community pool (`txfeecollector_cp`) via dedicated module accounts
- Fee conversion moved from epoch hooks to EndBlock processing (`service_fees_handler.go`)
- Dynamic fee denomination validation: ante handler enforces single-denomination fees per transaction, uses `HasDeepLiquidityWithNativeDenom` for stricter liquidity checks, and tracks fee denom in context via `FeeDenomKey`
- `DeductFeeDecorator` now uses the module keeper directly with `getContextMinGasPrices()` for dynamic min gas price calculation based on spot prices
- Added `ValidatorMinGasFee` (default `0.01ubze`) and `MaxBalanceIterations` (default 100) parameters
- ConsensusVersion bumped from 1 to 2
