# DAO Module – User Guide

The DAO module lets anyone create an on-chain organisation with its own treasury, run proposals and execute them as the DAO.

> **Work in progress.** DAOs can be created, but proposals, voting and administration are not there yet. Each feature adds its section to this page as it lands.

## Concepts
- **DAO account**: every DAO gets its own account, derived from its id (`address.Module("dao", uint64-big-endian(id))`). Nobody holds a key for it; it is the DAO's treasury.
- **Electorate**: a DAO votes either through a fixed member list (`static-list`, one vote per member) or through stakers of reward programs (`stake`, coming later). The two kinds are never mixed.
- **Admin**: optional. Without one the DAO owns itself (its admin is its own address). The admin can be a person or another DAO.

## Messages
- `MsgCreateDao`: creates a DAO. Anyone can send it; the creator pays the `dao_creation_fee` (burned or sent to the community pool, per `dao_creation_fee_destination`). Returns the new DAO's id and address. Rules:
  - `name` 1–64 bytes, `description` up to 2,000 bytes.
  - `config`: `min_quorum`, `threshold`, `expedite_quorum` and `expedite_threshold` in (0, 1]; `expedite_quorum ≥ min_quorum`, `expedite_threshold ≥ threshold`; `voting_period` and `expedited_period` within [`min_voting_period`, `max_voting_period`] and `expedited_period ≤ voting_period`. The expedite fields are checked even when `allow_expedite` is off. `proposal_cost` and `proposal_deposit` are optional coins (amount ≥ 0).
  - `static-list`: 1 to `max_static_members` unique member addresses, no stake electorate. `stake` DAOs are rejected for now.
- `MsgUpdateParams` **(authority)**: replaces the module parameters. Reserved for the module authority (governance).

Example (CLI):
```bash
# Create a static-list DAO. In raw JSON, ratios are 18-decimal integers
# (500000000000000000 = 0.5) and periods are seconds.
bzed tx dao create-dao static-list "BZE builders" \
  '{"min_quorum":"400000000000000000","threshold":"500000000000000000","voting_period":"259200s","allow_expedite":true,"expedited_period":"86400s","expedite_quorum":"600000000000000000","expedite_threshold":"660000000000000000"}' \
  --members bze1...,bze1... --description "Funds BZE tooling" --from mykey

# Submit a governance proposal that sets the module parameters (all of them)
bzed tx dao update-params \
  '{"max_static_members":"128","min_voting_period":"3600s","max_voting_period":"7776000s","dao_creation_fee":{"denom":"ubze","amount":"50000000000"},"dao_creation_fee_destination":"burner"}' \
  --deposit 1000000ubze --title "Update x/dao params" --summary "..." \
  --from mykey
```

## Events
- `bze.dao.EventDaoCreated{id, address, admin, type}`

## Queries
- `bzed query dao params` – show the module parameters.

## Parameters
| Parameter | Default | Meaning |
| --- | --- | --- |
| `max_static_members` | 128 | Most members a static-list DAO can have. |
| `min_voting_period` | 1h | Shortest voting or expedited period a DAO may configure. |
| `max_voting_period` | 90 days | Longest voting or expedited period a DAO may configure. |
| `dao_creation_fee` | 50,000 BZE (`50000000000ubze`) | Paid by the creator. A zero amount makes creation free. |
| `dao_creation_fee_destination` | `burner` | `burner` or `community_pool`. |
