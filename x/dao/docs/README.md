# DAO Module – User Guide

The DAO module lets anyone create an on-chain organisation with its own treasury, run proposals and execute them as the DAO.

> **Work in progress.** DAOs can be created, their admin can run messages as the DAO and hand it over to a successor, and proposals can be submitted and voted on, but settlement and execution are not there yet. Each feature adds its section to this page as it lands.

## Concepts
- **DAO account**: every DAO gets its own account, derived from its id (`address.Module("dao", uint64-big-endian(id))`). Nobody holds a key for it; it is the DAO's treasury.
- **Electorate**: a DAO votes either through a fixed member list (`static-list`, one vote per member) or through the stakers of `x/rewards` programs (`stake`): up to `max_sr_per_dao` staking rewards and/or one Denom Reward, all staking the same denom. The two kinds are never mixed.
- **Sources and pins**: each staking reward or Denom Reward in a stake electorate is a *source*. The module counts how many DAOs use each source; a staking reward used by any DAO cannot be deleted from `x/rewards` (its removal is vetoed until no DAO uses it).
- **Join records**: when stake joins a source that a DAO or an open proposal uses, the module records the block height. Stake counts for a proposal only when it joined strictly before the block the proposal was submitted in (or has no record: it joined while nothing used the source). Topping up keeps the record; exiting deletes it.
- **Ballots**: a voter has one ballot per proposal. Its weight is 1 in a `static-list` DAO; in a `stake` DAO it is the voter's eligible stake, kept as one share per source. When the voter exits a source, that source's share leaves every ballot of theirs on a proposal in voting, and the tally drops with it.
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
- `MsgSubmitProposal`: opens a proposal of zero or more messages that the DAO runs if it passes. Returns the proposal id. Rules:
  - The signer (`proposer`) must be a voter: a member of a `static-list` DAO, or for a `stake` DAO an address with eligible stake in one of its sources now (see join records). With `open_submission` anyone can propose. The DAO account itself can always propose (through a proposal or its admin) and pays from its treasury.
  - The DAO must have voting power: members, or stake in its sources. At most `max_open_proposals` proposals of a DAO can be in voting at once.
  - `title` 1–`max_title_len` bytes, `summary` up to `max_summary_len` bytes; 0 to `max_msgs_per_proposal` messages, at most `max_proposal_bytes` bytes in total. Every signer of every message must be the DAO account.
  - `expedited` needs `allow_expedite` in the config; voting then lasts `expedited_period` instead of `voting_period`.
  - `proposal_cost` (if set) is paid to the burner (`cost_destination` `BURN`) or the DAO's treasury (`TREASURY`); `proposal_deposit` (if set) is escrowed in the `dao` module account.
  - The proposal pins the DAO's config, its electorate and its total voting power (member count, or the stake of its sources): later DAO changes never move an open vote. Every source it pins counts as used, so a staking reward a proposal votes with cannot be deleted.
- `MsgVote`: casts or changes the signer's (`voter`) ballot: `YES`, `NO` or `ABSTAIN`. A vote with no option (`VOTE_OPTION_UNSPECIFIED`, the value an omitted field decodes to) is refused. Rules:
  - The proposal must be in voting, and the block time strictly before its `voting_end`.
  - Weight: in a `static-list` DAO, 1 if the voter is on the DAO's member list now; in a `stake` DAO, the voter's stake in each source the proposal pinned, counting only stake with no join record or one strictly below the proposal's submit height. Zero weight is refused.
  - Voting again with another option replaces the ballot at the voter's current weight, so stake added since the last vote counts; the same option twice is refused.
  - Exiting a staking reward or Denom Reward takes its share out of the voter's ballots on every proposal in voting (`EventBallotReduced`). Rejoining does not bring it back on those proposals.
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

# As a member of DAO 1, propose to pay 1 BZE from its treasury (msg.json holds the MsgSend,
# from the DAO address); several messages are several arguments
bzed tx dao submit-proposal 1 "Fund tooling" msg.json --summary "Pays the tooling team" --from member

# As a member of DAO 1, vote YES on proposal 1; voting again with NO replaces the ballot
bzed tx dao vote 1 YES --from member

# Submit a governance proposal that sets the module parameters (all of them)
bzed tx dao update-params \
  '{"max_sr_per_dao":"5","max_static_members":"128","max_open_proposals":"10","max_msgs_per_proposal":"16","max_proposal_bytes":"20000","max_title_len":"140","max_summary_len":"5000","min_voting_period":"3600s","max_voting_period":"7776000s","dao_creation_fee":{"denom":"ubze","amount":"50000000000"},"dao_creation_fee_destination":"burner","min_electorate_lock_days":5}' \
  --deposit 1000000ubze --title "Update x/dao params" --summary "..." \
  --from mykey
```

## Events
- `bze.dao.EventDaoCreated{id, address, admin, type}`
- `bze.dao.EventAdminExecuted{id, admin, msg_count}`, after the events of the executed messages
- `bze.dao.EventAdminNominated{id, admin, nominee}`
- `bze.dao.EventAdminChanged{id, old_admin, new_admin}`, when a nominee accepts or the DAO is handed to its voters
- `bze.dao.EventAdminNominationWithdrawn{id, nominee}`
- `bze.dao.EventProposalSubmitted{id, dao_id, proposer, expedited, voting_end, denominator}`
- `bze.dao.EventVote{proposal_id, voter, option, weight}`, on every vote and revote
- `bze.dao.EventBallotReduced{proposal_id, voter, source, amount}`, when an exit takes a share out of a ballot

## Queries
- `bzed query dao params` – show the module parameters.

## Parameters
| Parameter | Default | Meaning |
| --- | --- | --- |
| `max_sr_per_dao` | 5 | Most staking rewards a stake DAO's electorate can have. |
| `max_static_members` | 128 | Most members a static-list DAO can have. |
| `max_open_proposals` | 10 | Most proposals a DAO can have in voting at once. |
| `max_msgs_per_proposal` | 16 | Most messages a proposal or a `MsgAdminExecute` can carry. |
| `max_proposal_bytes` | 20,000 | Most bytes those messages can take in total. |
| `max_title_len` | 140 | Longest proposal title, in bytes. |
| `max_summary_len` | 5,000 | Longest proposal summary, in bytes (0 allows titles only). |
| `min_voting_period` | 1h | Shortest voting or expedited period a DAO may configure. |
| `max_voting_period` | 90 days | Longest voting or expedited period a DAO may configure. |
| `dao_creation_fee` | 50,000 BZE (`50000000000ubze`) | Paid by the creator. A zero amount makes creation free. |
| `dao_creation_fee_destination` | `burner` | `burner` or `community_pool`. |
| `min_electorate_lock_days` | 5 | Shortest unstaking lock (days) a staking reward or Denom Reward needs to join a stake electorate. Must be at least 1. |
