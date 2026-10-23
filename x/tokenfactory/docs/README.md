# TokenFactory Module – User Guide

The tokenfactory lets any account create and manage custom denoms under their own admin key.

## What You Can Do
- **Create a denom** (`MsgCreateDenom`): pays the configured creation fee (captured and swapped to native, then sent to the fee collector), creates `factory/<creator>/<subdenom>`, and sets you as admin.
- **Mint** (`MsgMint`): admin-only, and only while the denom is mintable; mints the specified amount to the admin’s account.
- **Burn** (`MsgBurn`): admin-only; burns tokens from the admin’s balance.
- **Change admin** (`MsgChangeAdmin`): optionally transfer admin rights (or clear admin to lock the supply).
- **Disable minting** (`MsgDisableMinting`): admin-only and irreversible; fixes the supply while the admin keeps every other right. See [Fixed supply](#fixed-supply).
- **Set metadata** (`MsgSetDenomMetadata`): admin-only; updates bank metadata for wallets and explorers.
- **Configure the token** (`MsgSetDenomConfig`): admin-only; sets the token's community pool address, transfer fee and gas surcharge. See [Token configuration](#token-configuration).
- **Assign an admin by governance** (`MsgSetDenomAdmin`): governance-only; gives a factory denom a new admin, including a renounced one. See [Governance: assigning an admin](#governance-assigning-an-admin).

Example (CLI):
```bash
# Create a denom
bzed tx tokenfactory create-denom mytoken --from mykey

# Mint 1,000 units
bzed tx tokenfactory mint "1000factory/<myaddr>/mytoken" --from mykey

# Burn 100 units
bzed tx tokenfactory burn "100factory/<myaddr>/mytoken" --from mykey

# Transfer admin
bzed tx tokenfactory change-admin factory/<myaddr>/mytoken <new-admin> --from mykey

# Turn minting off for good
bzed tx tokenfactory disable-minting factory/<myaddr>/mytoken --from mykey

# Configure the token: community pool, 0.1% transfer fee, 50% gas surcharge
bzed tx tokenfactory set-denom-config factory/<myaddr>/mytoken <cp-address> 0.001 0.5 --from mykey

# Back to the defaults: no community pool, zero rates
bzed tx tokenfactory set-denom-config factory/<myaddr>/mytoken "" 0 0 --from mykey
```

## Queries
- `bzed query tokenfactory params` – view the creation fee.
- `bzed query tokenfactory denom-authority <denom>` – view the current admin of a factory denom and its `mintable` flag. An empty admin means the denom is locked until a governance proposal assigns a new admin; `mintable: false` means nobody can ever mint it again.
- `bzed query tokenfactory denom-config <denom>` – view the token's community pool, transfer fee and gas surcharge. A token that never set them returns the defaults.
- Bank module queries (`denom-metadata`, balances) apply to factory denoms as usual.

REST: `/bze/tokenfactory/params`, `/bze/tokenfactory/denom_authority?denom=<denom>`, `/bze/tokenfactory/denom_config?denom=<denom>`.

## Fixed supply
Every denom carries a `mintable` flag on its authority record. New denoms are mintable. Being the admin is not enough to mint: the denom must also be mintable, otherwise `MsgMint` is refused with `ErrMintingDisabled`.

- `MsgDisableMinting` turns the flag off. Only the current admin may send it, and it emits `DenomMintingDisabledEvent{denom, admin}`. Sending it for a denom that is already non-mintable is refused with `ErrMintingDisabled`.
- It is **irreversible**: no message turns minting back on, governance included. `MsgChangeAdmin` and `MsgSetDenomAdmin` change the admin and never touch the flag.
- The admin keeps every other right: burn their own coins, metadata, branding, change admin or renounce. This is the way to promise holders a fixed supply while still rebranding the token, for example for a DAO or a community.

## Token configuration
Each factory denom has a `DenomConfig`, stored next to its authority record:

| Field | Meaning | Allowed values | Default |
|---|---|---|---|
| `community_pool` | address that receives the token's fee coins | empty (none) or a valid, non-blocked address | empty |
| `transfer_fee` | ratio charged on top of user sends | `0` to `0.05` (5%) | `0` |
| `gas_surcharge` | ratio added when the token pays gas | `0` to `2` (200%) | `0` |

- `MsgSetDenomConfig` replaces the whole configuration. **Every field must be sent**: a missing rate is refused, so send `0` explicitly. An empty `community_pool` means none, and it is the only way to clear it.
- Rates are decimal ratios, like staking and DEX rates: `0.001` is 0.1%. The caps (`MaxTransferFee`, `MaxGasSurcharge`) are chain constants; changing them needs an upgrade.
- Only the current admin may send it, a user or a DAO. The admin can raise or lower both rates and change or clear the community pool at any time. After a renounce nobody can change it: the token keeps its configuration as it was.
- A community pool the bank module blocks (module accounts such as the fee collector or the x/dao module) is refused with `ErrInvalidDenomConfig`. A DAO treasury is a plain account and is accepted.
- Rates can be set without a community pool. The transfer fee and gas surcharge are then still charged, and the coins are burned.
- Rate changes, increases included, apply from the next block. The caps are the holders' protection.
- Sending the defaults deletes the stored entry. A token without an entry, and any non-factory denom, reads as the defaults; the query never returns "not found".
- Every change emits `DenomConfigChangeEvent{denom, community_pool, transfer_fee, gas_surcharge}` with the new values.
- Genesis exports the configuration on each `GenesisDenom` as `denom_config`, only for tokens that have one.

Tokenfactory only stores and validates these settings and never charges or routes anything itself. Other modules act on them: txfeecollector forwards the token's fee coins to the community pool and charges the gas surcharge, and tradebin charges the transfer fee on protocol fees paid in the token — see [Transfer fee and gas surcharge](../../txfeecollector/docs/README.md#transfer-fee-and-gas-surcharge).

## Locking Supply
To lock a denom completely (no future minting, burning, or metadata changes), call `MsgChangeAdmin` with an empty `newAdmin`. This is irreversible by the admin: only a governance proposal can assign a new admin afterwards (see below).

Renouncing also turns minting off in the same write (and emits `DenomMintingDisabledEvent` when the denom was still mintable). An admin that governance later assigns to a renounced denom therefore cannot mint it: holders who were told the supply was fixed stay protected.

## Permissions
- Only the current admin may mint, burn, change admin, disable minting, set metadata, or set the token configuration for a factory denom. Minting also needs the denom to be mintable. Creation is open to anyone who can pay the fee.
- `MsgUpdateParams` and `MsgSetDenomAdmin` are restricted to the module authority (governance).

## Governance: assigning an admin
`MsgSetDenomAdmin` lets a governance proposal set the admin of a factory denom to any account. Its main use is handing a token, typically a renounced one, to a DAO, but the new admin can be any valid address: a DAO account, a user or a module account.

- It works whatever the current admin is: an empty (renounced) admin or a live one, which is overwritten without its consent. The vote is the safeguard.
- `new_admin` must be a valid, non-empty address. Governance gives a denom to someone; it never locks one. Locking stays the admin's own decision (`MsgChangeAdmin` with an empty `newAdmin`).
- Only denoms created by the tokenfactory are accepted (they have an authority record). `ubze`, IBC and LP denoms are refused with `ErrDenomDoesNotExist`.
- One denom per message. A proposal that reassigns several denoms carries several messages.
- The new admin then has every right an original creator has: burn, metadata, branding, disable minting, change admin or renounce, and mint while the denom is still mintable. A renounced denom is never mintable, so its new admin cannot mint it.
- A `DenomAdminChangeEvent` is emitted, with `admin` set to the previous admin (empty for a renounced denom) and `new_admin` to the new one.

Runbook:
1. Write `proposal.json` with a single `/bze.tokenfactory.MsgSetDenomAdmin` message. `authority` is the gov module account (mainnet `bze10d07y265gmmuvt4z0w9aw880jnsr700j8xlwyy`).
   ```json
   {
     "messages": [
       {
         "@type": "/bze.tokenfactory.MsgSetDenomAdmin",
         "authority": "bze10d07y265gmmuvt4z0w9aw880jnsr700j8xlwyy",
         "denom": "factory/<creator>/<subdenom>",
         "new_admin": "<new admin address>"
       }
     ],
     "metadata": "<ipfs or url>",
     "deposit": "<amount>ubze",
     "title": "Assign an admin to <subdenom>",
     "summary": "<why>"
   }
   ```
2. Submit it with `bzed tx gov submit-proposal proposal.json --from <key>`.
3. Once it passes, `bzed query tokenfactory denom-authority <denom>` shows the new admin.

## Version History

### v8.3.0
- `MsgSetDenomAdmin`: governance assigns a new admin to any factory denom, renounced denoms included
- `mintable` flag on the denom authority and `MsgDisableMinting` (admin-only, irreversible); `MsgMint` is refused on non-mintable denoms; renouncing the admin turns minting off
- Consensus version 3→4: the migration sets `mintable = true` on every denom that has an admin and `mintable = false` on every renounced denom
- `DenomConfig` (community pool, transfer fee, gas surcharge) with `MsgSetDenomConfig` (admin-only), the `denom_config` query and `DenomConfigChangeEvent`; no migration, a missing config means the defaults

### v8.1.0
- Subdenom validation now disallows slash (`/`) characters to prevent ambiguous denom paths
- Creation fee routing changed from `FundCommunityPool` (distribution keeper) to `CaptureAndSwapUserFee` (trade keeper) with forwarding to fee collector
