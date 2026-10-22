# TxFeeCollector Module – Operator Guide

TxFeeCollector standardizes fees by converting module balances to the native denom and enforcing a minimum gas price.

## What It Does
- **Fee conversion:** Swaps non-native fees held by:
  - the main fee collector,
  - the burner module,
  - the community pool fee collector,
  into the native denom via the trade module when possible. Skipped coins (non-swappable) can be forwarded to burner.
- **Token community pools:** Before converting `txfeecollector_cp`, the module forwards every coin there whose denom has a community pool address (set by the factory token's admin in its `DenomConfig`, `MsgSetDenomConfig` in x/tokenfactory) to that address, once per block. Each forwarded coin emits a `CommunityPoolFeePaidEvent` (denom, recipient, amount). See [Community pool fees](#community-pool-fees).
- **Token transfer fees and gas surcharges:** Charges the transfer fee and gas surcharge a factory token's admin sets in its `DenomConfig`, into `txfeecollector_cp` (or `txfeecollector_burner` when the token has no community pool). See [Transfer fee and gas surcharge](#transfer-fee-and-gas-surcharge).
- **Minimum gas price:** Ante handler enforces a per-validator minimum gas price, with cross-denom support using spot prices from the trade module.
- **Inbound IBC filter:** Refuses incoming ICS-20 transfers listed in the `blocked_ibc_inbound` param with an error acknowledgement, so the sending chain refunds the sender and no voucher is minted here. Exits are untouched — sends, acknowledgements, timeout refunds and the channel lifecycle all behave as if the filter were not there. Every refusal emits a `BlockedIbcInboundEvent` typed event (channel, denom, amount, sender, receiver) so relayers and indexers can see what was bounced.
- **Governance:** Parameters can be updated via `MsgUpdateParams` (authority only).

## Module Accounts
The module registers three dedicated accounts:
- **`txfeecollector`** – holds non-native fee tokens awaiting conversion to native denom.
- **`txfeecollector_burner`** – holds fees designated for the burner module.
- **`txfeecollector_cp`** – holds fees designated for the community pool via the distribution module.

At EndBlock, collected fees are converted to native denom and distributed to their respective destinations.

## Community pool fees
`txfeecollector_cp` receives the `treasury` share of AMM pool fees (in the coin being sold), the tradebin create fees (already swapped to ubze), the DAO creation fee when it is set to go there, and order book fees too small to swap. Each EndBlock it is emptied in two steps, each committed on its own:
1. **Routing:** every coin whose denom has a community pool in its tokenfactory `DenomConfig` is sent to that address. The routing is by denom only, whoever sent the coin. The step looks at no more than `max_balance_iterations` coins per block; coins past the bound wait for a later block. If any send fails, the error is logged and the whole step is discarded, sends already made in it included, so that block every coin stays for step 2. It never fails the EndBlock, and a failed conversion in step 2 does not roll it back.
2. **Conversion:** what is left (ubze, IBC coins, factory tokens without a community pool, and every coin of a block whose routing failed) is swapped to ubze and funds the chain's community pool through x/distribution, exactly as before. Coins that cannot be swapped go to the burner. After a successful routing, a coin whose denom has a community pool is never converted: it is past the routing bound and waits for a later block's routing.

So a factory token's community pool earns the treasury share of the pool fees paid in that token, that is on sells of the token; a buy pays its share in the other coin. Only factory tokens can have a community pool; ubze and IBC coins always fund the chain's community pool.

## Transfer fee and gas surcharge
A factory token's admin can set two rates in its `DenomConfig` (`MsgSetDenomConfig` in x/tokenfactory, see its [Token configuration](../../tokenfactory/docs/README.md#token-configuration)): a **transfer fee** (0 to 5%) and a **gas surcharge** (0 to 200%). This module charges both. ubze, IBC coins, LP shares and factory tokens without rates are never charged and cost no extra store read. A rate change, increase included, applies from the next send or tx: the caps are the holders' protection.

**Where the fees go.** When the token has a community pool, both fees go to `txfeecollector_cp`, and the [routing](#community-pool-fees) forwards them to the token's community pool at EndBlock. When it has none, they go to `txfeecollector_burner` and are burned (swapped to ubze and burned, or burned as is when unswappable). Each fee coin emits an event naming its `destination` module account.

### Transfer fee
Charged **on top** of the amount whenever a **user account** sends the token: the sender pays amount + fee and the recipient gets the full amount. It is an x/bank send restriction, so it covers every bank send (`MsgSend`, `MsgMultiSend`, and every module that takes the token from a user), but not mints, burns or delegations.

- **Amount:** `ceil(amount × transfer_fee)` per coin. It rounds **up**, so splitting a send into dust doesn't avoid it: 1 unit at 0.1% pays 1 unit. Wallets computing a "max" send must use the same formula, or the send fails by one unit.
- **Only the initial user send pays.** Module accounts never pay (pools, order escrow, rewards, the DAO module, module to module), and neither do the ICS-20 escrow accounts, which are plain accounts in ibc-go v10.
- **Balance check:** the sender's spendable balance must cover amount + fee, otherwise the send fails with `insufficient funds` naming the transfer fee, and nothing is charged.
- **Multisend:** each output pays its own fee. The balance is checked one output at a time, before x/bank debits the input, so when every output fits with its fee but the total doesn't, the send fails with x/bank's generic `insufficient funds` instead. The tx reverts, so nothing is charged.
- Every charged coin emits `TransferFeePaidEvent{payer, denom, amount, destination}`.

Examples at a 1% transfer fee:

| Action | Who pays the fee |
|---|---|
| A sends 1 TOKEN to B | A pays 1.01, B gets 1 |
| Swap 1 TOKEN for ubze | the user pays 1.01, the swap runs on exactly 1 |
| Burn 1 TOKEN | the user pays 1.01: 1 is burned, 0.01 is the fee |
| Swap ubze for 10 TOKEN | nobody: the pool sends, the user gets 10 |
| IBC transfer out, order placement, adding liquidity, rewards staking, a DAO treasury spend (treasuries are plain accounts) | the sending user or treasury |
| A protocol fee paid in TOKEN (denom, market, reward or DAO creation) | the payer; with too little TOKEN for fee + transfer fee, the protocol fee is taken in ubze instead |
| IBC transfers in, refunds and timeouts, order fills, cancel refunds, removing liquidity, unstaking, claims | nobody: a module or escrow sends |

### Gas surcharge
Charged when the token **pays the tx fee**, instead of the transfer fee (gas never pays the transfer fee, and protocol fees never pay the surcharge).

- **Minimum gas price** (CheckTx only): the token's price, converted from the ubze minimum gas price through its ubze pool, is multiplied by `1 + gas_surcharge` before the comparison with the validator's own price. A spot-price failure keeps the ubze fallback as before.
- **Split:** the whole fee goes to the `txfeecollector` module as before, then `floor(fee × s / (1 + s))` moves on to the token's destination; the rest is converted for the validators. The split runs on the fee actually paid: DeliverTx doesn't check the minimum gas price, so a validator that includes a cheaper tx gives the community pool the same share of a smaller fee.
- Fee grants are charged the full declared fee, surcharge included; the granter pays it. The surcharge stays charged when the tx fails, like the rest of the fee.
- Every charged coin emits `GasSurchargePaidEvent{payer, denom, amount, destination}`.

## Ante Handler
Two decorators run in the transaction processing pipeline:
- **`ValidateTxFeeDenomsDecorator`**: enforces single-denomination fees per transaction and validates that non-native fee denoms have sufficient liquidity via the trade module.
- **`DeductFeeDecorator`**: splits fees by denom — native fees go directly to `auth.FeeCollectorName` (validators), non-native fees go to the `txfeecollector` module for later conversion, less a factory token's [gas surcharge](#gas-surcharge). Supports cross-denom minimum gas price calculation using trade module spot prices. The deduction never pays a transfer fee.

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
- Coins in `txfeecollector_cp` whose factory denom has a community pool address (x/tokenfactory `DenomConfig`) are forwarded to it in EndBlock, before the conversion for the chain's community pool
- Added the `CommunityPoolFeePaidEvent` typed event (`proto/bze/txfeecollector/events.proto`), one per forwarded coin
- No store change and no migration; ConsensusVersion stays 3
- Factory tokens' transfer fee (an x/bank send restriction) and gas surcharge (ante handler) are charged into `txfeecollector_cp`, or `txfeecollector_burner` for tokens without a community pool
- Added the `TransferFeePaidEvent` and `GasSurchargePaidEvent` typed events

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
