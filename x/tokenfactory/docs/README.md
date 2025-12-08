# TokenFactory Module – User Guide

The tokenfactory lets any account create and manage custom denoms under their own admin key.

## What You Can Do
- **Create a denom** (`MsgCreateDenom`): pays the configured creation fee (captured and swapped to native, then sent to the fee collector), creates `factory/<creator>/<subdenom>`, and sets you as admin.
- **Mint** (`MsgMint`): admin-only; mints the specified amount to the admin’s account.
- **Burn** (`MsgBurn`): admin-only; burns tokens from the admin’s balance.
- **Change admin** (`MsgChangeAdmin`): optionally transfer admin rights (or clear admin to lock the supply).
- **Set metadata** (`MsgSetDenomMetadata`): admin-only; updates bank metadata for wallets and explorers.

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
```

## Queries
- `bzed query tokenfactory params` – view the creation fee.
- Bank module queries (`denom-metadata`, balances) apply to factory denoms as usual.

## Permissions
- Only the current admin may mint, burn, change admin, or set metadata for a factory denom. Creation is open to anyone who can pay the fee.
