# Burner Module Parameters

This module exposes three on-chain parameters: how often the accumulated burner funds are destroyed, and the two limits of `MsgMoveIbcLockedCoins`.

## `periodic_burning_weeks`
- **Type:** `int64`
- **Default:** `1`
- **Purpose:** Determines how many weekly epochs to wait between automatic burns of the burner module account.
- **Hook:** The parameter is checked on the `week` epoch identifier. When `epochNumber % periodic_burning_weeks == 0`, the module attempts to burn everything it holds.
- **Recommendation:** Use a value `>= 1`. Setting `0` is technically allowed by validation but would break the modulo check at runtime.

### How It Is Used
- When the condition above is met, `burnModuleCoins` gathers all balances held by the `burner` module account and passes them through `BurnAnyCoins`.
- Burnable denoms (native and token factory) are destroyed; LP tokens and IBC denoms are locked in the black hole account; any successfully burned coins are recorded under the current block height.

## `ibc_move_max_reserve_ratio`
- **Type:** `cosmos.Dec` (`LegacyDec`)
- **Default:** `0.03`
- **Valid:** `(0, 1]`
- **Purpose:** One `MsgMoveIbcLockedCoins` adds at most this share of the pool's current reserve of the IBC denom: `min(locked balance, floor(ratio × reserve))`. The rest stays locked for later calls. A move whose cap rounds down to 0 is refused.

## `ibc_move_cooldown_blocks`
- **Type:** `uint64`
- **Default:** `1000`
- **Valid:** `>= 1`
- **Purpose:** After a successful move of a denom, the next move of that denom is refused until this many blocks have passed. Each denom has its own cooldown.

Chains upgrading to consensus version 5 get both defaults written by the module migration.

### Updating Parameters
- Message: `MsgUpdateParams` (authority-only, usually executed through governance).
- Fields: `authority` (must match the module authority), `params` (all three fields; send the current values of the ones you don't change).
- Example (submitted as a governance proposal depending on chain setup):
```bash
bzed tx gov submit-proposal update-params burner \
  --authority <gov-address> \
  --periodic-burning-weeks 2 \
  --from mykey
```
Adjust the exact CLI flags to your chain’s governance tooling; `MsgUpdateParams` replaces the whole params object.
