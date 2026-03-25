# TxFeeCollector Parameters

- **`validator_min_gas_fee`** (`DecCoin`, default `0.01ubze`): Minimum gas price the ante handler enforces. When transactions pay fees in another denom, the module uses the trade module’s spot price to convert this threshold into that denom; if spot price is unavailable, it falls back to the native denom minimum.

### How It’s Used
- During `CheckTx`, the ante decorator compares supplied fees against `validator_min_gas_fee` (or higher local min gas prices) and rejects transactions that underpay.
- Fee conversion routines operate independently of this param, but rely on the native denom set here when evaluating prices.

### Updating
- Only the module authority (typically governance) can set this via `MsgUpdateParams`. Supply the full `DecCoin` value (denom + amount) in the message.

## Version History

### v8.1.0
- Added `validator_min_gas_fee` parameter (default `0.01ubze`) — baseline minimum gas price enforced by the ante handler
- Added `max_balance_iterations` parameter (default 100) — caps iterations over module balances per block during EndBlock fee conversion
- ConsensusVersion bumped from 1 to 2; migration sets default values for new parameters
