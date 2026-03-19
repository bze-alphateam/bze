# TxFeeCollector Module – Operator Guide

TxFeeCollector standardizes fees by converting module balances to the native denom and enforcing a minimum gas price.

## What It Does
- **Fee conversion:** Swaps non-native fees held by:
  - the main fee collector,
  - the burner module,
  - the community pool fee collector,
  into the native denom via the trade module when possible. Skipped coins (non-swappable) can be forwarded to burner.
- **Minimum gas price:** Ante handler enforces a per-validator minimum gas price, with cross-denom support using spot prices from the trade module.
- **Governance:** Parameters can be updated via `MsgUpdateParams` (authority only).

## Module Accounts
The module registers three dedicated accounts:
- **`txfeecollector`** – holds non-native fee tokens awaiting conversion to native denom.
- **`txfeecollector_burner`** – holds fees designated for the burner module.
- **`txfeecollector_cp`** – holds fees designated for the community pool via the distribution module.

At EndBlock, collected fees are converted to native denom and distributed to their respective destinations.

## Ante Handler
Two decorators run in the transaction processing pipeline:
- **`ValidateTxFeeDenomsDecorator`**: enforces single-denomination fees per transaction and validates that non-native fee denoms have sufficient liquidity via the trade module.
- **`DeductFeeDecorator`**: splits fees by denom — native fees go directly to `auth.FeeCollectorName` (validators), non-native fees go to the `txfeecollector` module for later conversion. Supports cross-denom minimum gas price calculation using trade module spot prices.

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

### v8.1.0
- Added three-way fee distribution: fees split between validators/stakers, burner (`txfeecollector_burner`), and community pool (`txfeecollector_cp`) via dedicated module accounts
- Fee conversion moved from epoch hooks to EndBlock processing (`service_fees_handler.go`)
- Dynamic fee denomination validation: ante handler enforces single-denomination fees per transaction, uses `HasDeepLiquidityWithNativeDenom` for stricter liquidity checks, and tracks fee denom in context via `FeeDenomKey`
- `DeductFeeDecorator` now uses the module keeper directly with `getContextMinGasPrices()` for dynamic min gas price calculation based on spot prices
- Added `ValidatorMinGasFee` (default `0.01ubze`) and `MaxBalanceIterations` (default 100) parameters
- ConsensusVersion bumped from 1 to 2
