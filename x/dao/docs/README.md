# DAO Module – User Guide

The DAO module lets anyone create an on-chain organisation with its own treasury, run proposals and execute them as the DAO.

> **Work in progress.** This version contains the module skeleton only: it is wired into the chain and holds its parameters, but has no DAO features yet. Each feature adds its section to this page as it lands.

## Messages
- `MsgUpdateParams` **(authority)**: replaces the module parameters. Reserved for the module authority (governance).

Example (CLI):
```bash
# Submit a governance proposal that sets the module parameters
bzed tx dao update-params '{}' \
  --deposit 1000000ubze --title "Update x/dao params" --summary "..." \
  --from mykey
```

## Queries
- `bzed query dao params` – show the module parameters.

## Parameters
The module has no parameters yet. Each one is added together with the feature that uses it.
