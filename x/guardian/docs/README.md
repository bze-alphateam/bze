# Guardian Module – User Guide

The guardian module is the emergency path for the IBC rate limits of the `ratelimit` module (`cosmos/ibc-apps` rate-limiting). It can only make IBC transfers stricter; loosening them is left to governance.

## Concepts
- **Guardian**: one address, kept in the module params (empty means only governance). The guardian or the governance authority signs the emergency messages. Only governance changes the guardian (`MsgUpdateParams`).
- **Blacklist**: the `ratelimit` module's denom blacklist. Every ICS-20 transfer of a blacklisted denom is refused in both directions on every channel, with or without a rate limit (outgoing transfers fail as transactions, incoming ones are refunded on the sending chain).

## Messages
- **Tighten a rate limit** (`MsgTightenRateLimit`, guardian or governance): lowers the send and receive percents of an existing rate limit. A raise of either percent is refused, no limit is created, the window length is unchanged and the flow already counted in the current window is kept. `0`/`0` stops the pair.
- **Block a denom** (`MsgBlockDenom`, guardian or governance): puts the denom on the blacklist. The blacklist matches the exact denom string, so the denom must exist on chain (have a supply), written exactly as the chain stores it: for an IBC token that is `ibc/` followed by the upper-case hash. A lower-case hash, a base name such as `uusdc`, a trace path such as `transfer/channel-3/uusdc` or an unknown denom is refused with `denom does not exist on chain`. Blocking an already blocked denom is refused.
- **Unblock a denom** (`MsgUnblockDenom`, governance only): takes the denom off the blacklist. Any entry can be removed, whether or not the denom still has a supply. The guardian can never reopen a denom.
- **Halt the chain** (`MsgHaltChain`, guardian or governance): the last resort. Schedules an x/upgrade plan named `emergency-halt-<height>` at the next block height (the earliest x/upgrade accepts). The `reason` (1 to 256 bytes) is emitted in `ChainHaltScheduledEvent` only; the plan info is always the fixed text `emergency halt` (tools such as Cosmovisor read the info as JSON with binary download links, so no signer-chosen text goes there). Every node stops at that height. There is no message to restart it, see [Chain halt runbook](#chain-halt-runbook).

```bash
# As the guardian, stop every IBC transfer of an IBC denom (exact on-chain spelling)
bzed tx guardian block-denom ibc/6490A7EAB61059BFC1CDDEB05917DD70BDF3A611654162A1A47DB930D40D8AF4 --from guardian

# As the guardian, lower the ubze quotas on channel-0 to 3% send, 0% receive
bzed tx guardian tighten-rate-limit ubze channel-0 3 0 --from guardian

# As the guardian, stop the whole chain at the next block
bzed tx guardian halt-chain "incident under investigation" --from guardian
```

## Chain halt runbook
`MsgHaltChain` is an ordinary upgrade plan, so the chain stops the way it stops for a release:

1. **Halt.** The guardian (or a governance proposal) sends `MsgHaltChain`. The tx response and the `ChainHaltScheduledEvent` give the plan name and height `H` (the tx height + 1). `bzed query upgrade plan` shows the plan until the chain stops.
2. **The chain stops at `H`.** Every node logs `UPGRADE "emergency-halt-H" NEEDED at height: H: emergency halt`, writes `upgrade-info.json` to its data directory and stops committing blocks (Cosmovisor then looks for a binary under `upgrades/emergency-halt-H`). State is the one after block `H - 1`; no transaction of block `H` is executed.
3. **Replaced plan.** x/upgrade keeps one plan, so a release plan scheduled for a later height is replaced by the halt plan. Governance schedules it again after the restart.
4. **Restart**, either way:
   - **New binary** (the normal way, e.g. with the fix): the binary registers an upgrade handler named `emergency-halt-H` (it may just return the version map, or run a migration that repairs state). Validators install it (under Cosmovisor: `upgrades/emergency-halt-H/bin/bzed`) and start; block `H` runs the handler and the chain continues. The plan is cleared.
   - **Skip the height** (no code change needed): every validator, with at least 2/3 of the voting power, restarts the *same* binary with `bzed start --unsafe-skip-upgrades H`. x/upgrade then clears the plan at `H` and the chain continues. Remove the flag on the next restart.
5. **Afterwards**, governance reschedules any release plan the halt replaced.

The guardian cannot cancel a halt (a `MsgCancelUpgrade` is governance only and would have to pass before `H`, which is one block away). Only use it when a running chain does more harm than a stopped one.

## Queries
- `bzed query guardian params`: the guardian address.
- `bzed query guardian blocked-denoms`: the blacklisted denoms (the `ratelimit` module has no CLI for its blacklist; its gRPC `AllBlacklistedDenoms` works too).

## Events
- `TightenRateLimitEvent` (signer, denom, channel, old and new percents), `DenomBlockedEvent` (`signer`, `denom`) and `DenomUnblockedEvent` (`authority`, `denom`).
- `ChainHaltScheduledEvent` (`signer`, `name`, `height`, `reason`).

## Guardian as a DAO
The guardian address can be the account of a static-list `x/dao` DAO (e.g. five members with `expedite_threshold` 0.5 and `expedite_quorum` 0.6, so three yes votes pass). Static-list DAOs are exempt from `min_expedite_delay`: an expedited proposal carrying a guardian message is settled (and run) as soon as the third member has voted, with `MsgSettleProposal` in the next tx or in the same tx as the last vote.
