# TxFeeCollector Parameters

- **`validator_min_gas_fee`** (`DecCoin`, default `0.01ubze`): Minimum gas price the ante handler enforces. When transactions pay fees in another denom, the module uses the trade module’s spot price to convert this threshold into that denom; if spot price is unavailable, it falls back to the native denom minimum.
- **`blocked_ibc_inbound`** (`[]BlockedIbcTransfer`, default empty): pairs of `channel_id` (a channel on this chain) and `base_denom` (the denomination as written by the sending chain) whose **incoming** ICS-20 packets are answered with an error acknowledgement instead of minting a voucher. The sending chain refunds the escrowed tokens when it relays that acknowledgement, so nothing is trapped anywhere.

### How It’s Used
- During `CheckTx`, the ante decorator compares supplied fees against `validator_min_gas_fee` (or higher local min gas prices) and rejects transactions that underpay.
- Fee conversion routines operate independently of this param, but rely on the native denom set here when evaluating prices.
- `blocked_ibc_inbound` is read by the module’s IBC middleware (`x/txfeecollector/ibcmiddleware`), which sits between the ICS-29 fee middleware and the transfer application. Only received packets are inspected: outgoing transfers, their acknowledgements and their timeout refunds are never affected, channels are never closed, and vouchers of tokens that originally left this chain always pass (they are unescrowed, not minted).

### Updating
- Only the module authority (typically governance) can set this via `MsgUpdateParams`. Supply the full `DecCoin` value (denom + amount) in the message.

## Version History

### v8.2.0
- Added `blocked_ibc_inbound` parameter (default empty) — refuses selected inbound IBC transfers with an error acknowledgement
- ConsensusVersion bumped from 2 to 3; migration sets the new parameter to its empty default

### v8.1.0
- Added `validator_min_gas_fee` parameter (default `0.01ubze`) — baseline minimum gas price enforced by the ante handler
- Added `max_balance_iterations` parameter (default 100) — caps iterations over module balances per block during EndBlock fee conversion
- ConsensusVersion bumped from 1 to 2; migration sets default values for new parameters
