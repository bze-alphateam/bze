# DAO Module – User Guide

The DAO module lets anyone create an on-chain organisation with its own treasury, run proposals and execute them as the DAO.

> **Work in progress.** DAOs can be created, their admin can run messages as the DAO and hand it over to a successor, but proposals and voting are not there yet. Each feature adds its section to this page as it lands.

## Concepts
- **DAO account**: every DAO gets its own account, derived from its id (`address.Module("dao", uint64-big-endian(id))`). Nobody holds a key for it; it is the DAO's treasury.
- **Electorate**: a DAO votes either through a fixed member list (`static-list`, one vote per member) or through the stakers of `x/rewards` programs (`stake`): up to `max_sr_per_dao` staking rewards and/or one Denom Reward, all staking the same denom. The two kinds are never mixed.
- **Sources and pins**: each staking reward or Denom Reward in a stake electorate is a *source*. The module counts how many DAOs use each source; a staking reward used by any DAO cannot be deleted from `x/rewards` (its removal is vetoed until no DAO uses it).
- **Admin**: optional. Without one the DAO owns itself (its admin is its own address). The admin can be a person or another DAO. An external admin is a superuser: it can run any message as the DAO, including spending the treasury.

## Messages
- `MsgCreateDao`: creates a DAO. Anyone can send it; the creator pays the `dao_creation_fee` (burned or sent to the community pool, per `dao_creation_fee_destination`). Returns the new DAO's id and address. Rules:
  - `name` 1–64 bytes, `description` up to 2,000 bytes.
  - `config`: `min_quorum`, `threshold`, `expedite_quorum` and `expedite_threshold` in (0, 1]; `expedite_quorum ≥ min_quorum`, `expedite_threshold ≥ threshold`; `voting_period` and `expedited_period` within [`min_voting_period`, `max_voting_period`] and `expedited_period ≤ voting_period`. The expedite fields are checked even when `allow_expedite` is off. `proposal_cost` and `proposal_deposit` are optional coins (amount ≥ 0).
  - `static-list`: 1 to `max_static_members` unique member addresses, no stake electorate.
  - `stake`: no member list; `electorate.sr_ids` 0 to `max_sr_per_dao` unique existing staking reward ids, `electorate.dr_denom` optionally an existing Denom Reward (by its staking denom). Every source stakes the same denom, and every source's unstaking lock is at least `min_electorate_lock_days` days, so stake can't leave and vote again from another wallet during a vote. The lock rule applies when a source joins a DAO: raising the param later leaves existing DAOs untouched.
  - A self-owned DAO (no admin, or the DAO's own address as admin) needs a live electorate at creation: members for `static-list`; for `stake`, at least one source with something staked. With another admin, a `stake` DAO can start with no sources or no stake and be set up later.
- `MsgAdminExecute`: the DAO's external admin runs messages as the DAO. Rules:
  - The signer must be the DAO's admin when the message runs. A self-owned DAO has no external admin and is refused; it executes through its proposals.
  - 1 to `max_msgs_per_proposal` messages, at most `max_proposal_bytes` bytes in total (packed size of the messages).
  - Every signer of every message must be the DAO account, so the admin can spend the treasury but never another address. A message with a second signer, or a nested `MsgAdminExecute`, is refused.
  - The messages run in order; if one fails, the transaction fails and none of them is applied.
- `MsgNominateAdmin`: first step of the two-step admin handover. Rules:
  - The signer (`authority`) must be the DAO's current admin: a person signs directly, a self-owned DAO through a passed proposal, a parent DAO through its own proposal or admin.
  - Only one nomination can be pending at a time.
  - A nominee must be a valid address other than the current admin and the DAO itself. It is stored as the pending admin; the current admin stays in charge until the nominee accepts.
  - An empty nominee hands the DAO to its own voters at once, with no accept step: the DAO becomes self-owned. Its electorate must be able to vote (members for `static-list`; for `stake`, at least one source with something staked). A DAO that is already self-owned is refused.
- `MsgAcceptAdminNomination`: second step of the handover. Rules:
  - A nomination must be pending, and the signer (`nominee`) must be the pending admin. A DAO nominee signs through its own proposal or admin, which proves it can act; it then administers the other DAO as its parent.
  - The nominee becomes the admin and the nomination is cleared. The former admin loses all its powers at once, `MsgAdminExecute` included.
  - The electorate is not checked: the new admin is external and can fix it.
- `MsgWithdrawAdminNomination`: cancels a pending handover before the nominee accepts it. Rules:
  - The signer (`authority`) must be the DAO's current admin, signing as for `MsgNominateAdmin`. The nominee cannot withdraw; it simply does not accept.
  - A nomination must be pending. It is cleared, the admin stays in charge and can nominate again.
- `MsgUpdateParams` **(authority)**: replaces the module parameters. Reserved for the module authority (governance).

Example (CLI):
```bash
# Create a static-list DAO. In raw JSON, ratios are 18-decimal integers
# (500000000000000000 = 0.5) and periods are seconds.
bzed tx dao create-dao static-list "BZE builders" \
  '{"min_quorum":"400000000000000000","threshold":"500000000000000000","voting_period":"259200s","allow_expedite":true,"expedited_period":"86400s","expedite_quorum":"600000000000000000","expedite_threshold":"660000000000000000"}' \
  --members bze1...,bze1... --description "Funds BZE tooling" --from mykey

# Create a stake DAO voted by the stakers of staking rewards 3 and 7 and of the ubze Denom Reward
bzed tx dao create-dao stake "BZE stakers" \
  '{"min_quorum":"400000000000000000","threshold":"500000000000000000","voting_period":"259200s","expedited_period":"86400s","expedite_quorum":"600000000000000000","expedite_threshold":"660000000000000000"}' \
  --electorate '{"sr_ids":["3","7"],"dr_denom":"ubze"}' --from mykey

# As the external admin of DAO 1, pay 1 BZE from its treasury
bzed tx dao admin-execute 1 \
  '{"@type":"/cosmos.bank.v1beta1.MsgSend","from_address":"bze1<dao address>","to_address":"bze1...","amount":[{"denom":"ubze","amount":"1000000"}]}' \
  --from admin

# As the admin of DAO 1, nominate a successor, or hand the DAO to its voters
bzed tx dao nominate-admin 1 bze1... --from admin
bzed tx dao nominate-admin 1 "" --from admin

# As the nominee, accept and become the admin of DAO 1
bzed tx dao accept-admin-nomination 1 --from nominee

# As the admin of DAO 1, withdraw the pending nomination before it is accepted
bzed tx dao withdraw-admin-nomination 1 --from admin

# Submit a governance proposal that sets the module parameters (all of them)
bzed tx dao update-params \
  '{"max_sr_per_dao":"5","max_static_members":"128","max_msgs_per_proposal":"16","max_proposal_bytes":"20000","min_voting_period":"3600s","max_voting_period":"7776000s","dao_creation_fee":{"denom":"ubze","amount":"50000000000"},"dao_creation_fee_destination":"burner","min_electorate_lock_days":5}' \
  --deposit 1000000ubze --title "Update x/dao params" --summary "..." \
  --from mykey
```

## Events
- `bze.dao.EventDaoCreated{id, address, admin, type}`
- `bze.dao.EventAdminExecuted{id, admin, msg_count}`, after the events of the executed messages
- `bze.dao.EventAdminNominated{id, admin, nominee}`
- `bze.dao.EventAdminChanged{id, old_admin, new_admin}`, when a nominee accepts or the DAO is handed to its voters
- `bze.dao.EventAdminNominationWithdrawn{id, nominee}`

## Queries
- `bzed query dao params` – show the module parameters.

## Parameters
| Parameter | Default | Meaning |
| --- | --- | --- |
| `max_sr_per_dao` | 5 | Most staking rewards a stake DAO's electorate can have. |
| `max_static_members` | 128 | Most members a static-list DAO can have. |
| `max_msgs_per_proposal` | 16 | Most messages a `MsgAdminExecute` (and later a proposal) can carry. |
| `max_proposal_bytes` | 20,000 | Most bytes those messages can take in total. |
| `min_voting_period` | 1h | Shortest voting or expedited period a DAO may configure. |
| `max_voting_period` | 90 days | Longest voting or expedited period a DAO may configure. |
| `dao_creation_fee` | 50,000 BZE (`50000000000ubze`) | Paid by the creator. A zero amount makes creation free. |
| `dao_creation_fee_destination` | `burner` | `burner` or `community_pool`. |
| `min_electorate_lock_days` | 5 | Shortest unstaking lock (days) a staking reward or Denom Reward needs to join a stake electorate. Must be at least 1. |
