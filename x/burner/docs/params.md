# Burner Module Parameters

This module exposes a single on-chain parameter that controls how often the accumulated burner funds are destroyed.

## `periodic_burning_weeks`
- **Type:** `int64`
- **Default:** `1`
- **Purpose:** Determines how many weekly epochs to wait between automatic burns of the burner module account.
- **Hook:** The parameter is checked on the `week` epoch identifier. When `epochNumber % periodic_burning_weeks == 0`, the module attempts to burn everything it holds.
- **Recommendation:** Use a value `>= 1`. Setting `0` is technically allowed by validation but would break the modulo check at runtime.

### How It Is Used
- When the condition above is met, `burnModuleCoins` gathers all balances held by the `burner` module account and passes them through `BurnAnyCoins`.
- Burnable denoms (native and token factory) are destroyed; LP tokens are locked to a black hole account; IBC denoms are swapped to native via the trade module when possible before burning; any successfully burned coins are recorded under the current block height.

### Updating Parameters
- Message: `MsgUpdateParams` (authority-only, usually executed through governance).
- Fields: `authority` (must match the module authority), `params.periodic_burning_weeks`.
- Example (submitted as a governance proposal depending on chain setup):
```bash
bzed tx gov submit-proposal update-params burner \
  --authority <gov-address> \
  --periodic-burning-weeks 2 \
  --from mykey
```
Adjust the exact CLI flags to your chain’s governance tooling; the message itself only accepts the two fields above.
