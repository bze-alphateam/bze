# Guardian Module – User Guide

The guardian module holds the chain's emergency switches: the IBC rate limits of the `ratelimit` module (`cosmos/ibc-apps` rate-limiting), the `x/circuit` message circuit breaker, the DEX halt, the chain halt and the protocol's own automatic fund movements. Every switch follows one rule: the guardian or governance can switch something **off**, only governance switches it back **on**, and the guardian can never lock governance or itself out.

| Switch | Off (guardian or governance) | On (governance only) |
|---|---|---|
| Lower an IBC quota | `MsgTightenRateLimit` | `ratelimit` `MsgUpdateRateLimit` |
| Block a denom on IBC | `MsgBlockDenom` | `MsgUnblockDenom` |
| Disable message types | `MsgDisableMessages` | `x/circuit` `MsgResetCircuitBreaker` |
| Pause the protocol automation (txfeecollector EndBlock, burner periodic burn) | `MsgPauseAutomation` | `MsgResumeAutomation` |

## Concepts
- **Guardian**: one address, kept in the module params (empty means only governance). The guardian or the governance authority signs the emergency messages. Only governance changes the guardian (`MsgUpdateParams`).
- **Circuit breaker**: the `x/circuit` disable list. The app's message router refuses every message whose type is on the list with `circuit breaker disables execution of this message: <type url>`, whether it comes in a transaction or is executed by another module (a governance proposal, `authz` `MsgExec`, `x/dao` execution). `x/circuit`'s authority is the governance module account and no account holds `x/circuit` permissions; x/guardian writes the list directly, so the guardian can add to it but never take anything off.
- **Blacklist**: the `ratelimit` module's denom blacklist. Every ICS-20 transfer of a blacklisted denom is refused in both directions on every channel, with or without a rate limit (outgoing transfers fail as transactions, incoming ones are refunded on the sending chain).

## Messages
- **Tighten a rate limit** (`MsgTightenRateLimit`, guardian or governance): lowers the send and receive percents of an existing rate limit. A raise of either percent is refused, no limit is created, the window length is unchanged and the flow already counted in the current window is kept. `0`/`0` stops the pair.
- **Block a denom** (`MsgBlockDenom`, guardian or governance): puts the denom on the blacklist. The blacklist matches the exact denom string, so the denom must exist on chain (have a supply), written exactly as the chain stores it: for an IBC token that is `ibc/` followed by the upper-case hash. A lower-case hash, a base name such as `uusdc`, a trace path such as `transfer/channel-3/uusdc` or an unknown denom is refused with `denom does not exist on chain`. Blocking an already blocked denom is refused.
- **Unblock a denom** (`MsgUnblockDenom`, governance only): takes the denom off the blacklist. Any entry can be removed, whether or not the denom still has a supply. The guardian can never reopen a denom.
- **Disable message types** (`MsgDisableMessages`, guardian or governance): adds 1 to 100 type URLs (no duplicates) to the circuit breaker's disable list. Every URL is checked before anything is written; one refused URL refuses the whole message. Refused:
  - protected types, which would lock governance or the guardian out: every `/cosmos.gov.`, `/cosmos.circuit.`, `/cosmos.upgrade.` and `/bze.guardian.` message, and `/bze.dao.MsgSubmitProposal`, `/bze.dao.MsgVote`, `/bze.dao.MsgSettleProposal` (a DAO guardian needs them to act);
  - a URL that is not a message the app routes (`message type is not routed by the app`);
  - a type that is already disabled.
- **What can be disabled**: any other routed message type, including staking, bank or slashing messages (e.g. `/cosmos.staking.v1beta1.MsgDelegate`, `/cosmos.bank.v1beta1.MsgSend`, `/cosmos.slashing.v1beta1.MsgUnjail`), and stays disabled until governance resets it. A passed proposal that carries a disabled message fails on execution.
- **Re-enable message types** (governance only): a proposal with `x/circuit`'s `MsgResetCircuitBreaker`, signed by the governance authority. The guardian has no reset message and no `x/circuit` permission.
- **Halt denoms on the DEX** (`MsgHaltTradingDenoms`, guardian or governance): the same halt as the tradebin governance message `MsgHaltDenoms`, written to tradebin's halted denoms list. A halted denom gets no new order, fill, market, pool or liquidity add, no AMM swap involving it (user routes, module swaps, fee swaps) and is no longer accepted as a tx fee; cancelling orders, removing liquidity, bank sends and IBC transfers keep working. Up to 1000 valid denoms per message, no duplicates; the native denom is refused. A denom that is already halted is left as it is. **Turned back on only by governance**, with a proposal carrying tradebin's `MsgUnhaltDenoms`; the guardian cannot lift a halt.
- **Halt the chain** (`MsgHaltChain`, guardian or governance): the last resort. Schedules an x/upgrade plan named `emergency-halt-<height>` at the next block height (the earliest x/upgrade accepts). The `reason` (1 to 256 bytes) is emitted in `ChainHaltScheduledEvent` only; the plan info is always the fixed text `emergency halt` (tools such as Cosmovisor read the info as JSON with binary download links, so no signer-chosen text goes there). Every node stops at that height. There is no message to restart it, see [Chain halt runbook](#chain-halt-runbook).
- **Pause the protocol automation** (`MsgPauseAutomation`, guardian or governance): stops the txfeecollector EndBlock (both the gas fee pass and the community pool pass: no swap, no send, no community pool funding) and the burner's periodic burn (no burn, no lock, no liquidity added). Every coin stays where it is, in `txfeecollector`, `txfeecollector_cp` and `burner`. Pausing an already paused automation is refused. Raffle draws and the cleanup of finished raffles are user-facing and keep running.
- **Resume the protocol automation** (`MsgResumeAutomation`, governance only): lifts the pause. Nothing is lost while paused: the txfeecollector passes read the collectors' balances, so the first EndBlock after the resume handles the coins collected during the pause; a burning week that ended during the pause still marks the periodic burn pending, and the first EndBlock after the resume burns the whole burner balance instead of waiting for the next burning week. The guardian can never resume.


```bash
# As the guardian, disable swaps on the DEX
bzed tx guardian disable-messages /bze.tradebin.MsgMultiSwap --from guardian

# Governance re-enables them: a proposal with this message
#   {"@type": "/cosmos.circuit.v1.MsgResetCircuitBreaker",
#    "authority": "<gov module address>",
#    "msg_type_urls": ["/bze.tradebin.MsgMultiSwap"]}
bzed tx gov submit-proposal proposal.json --from mykey

# As the guardian, stop every IBC transfer of an IBC denom (exact on-chain spelling)
bzed tx guardian block-denom ibc/6490A7EAB61059BFC1CDDEB05917DD70BDF3A611654162A1A47DB930D40D8AF4 --from guardian

# As the guardian, halt two denoms on the DEX
bzed tx guardian halt-trading-denoms factory/bze1.../utoken ibc/6490A7EAB61059BFC1CDDEB05917DD70BDF3A611654162A1A47DB930D40D8AF4 --from guardian

# As the guardian, lower the ubze quotas on channel-0 to 3% send, 0% receive
bzed tx guardian tighten-rate-limit ubze channel-0 3 0 --from guardian

# As the guardian, stop the whole chain at the next block
bzed tx guardian halt-chain "incident under investigation" --from guardian

# As the guardian, freeze the txfeecollector conversions and the periodic burn
bzed tx guardian pause-automation --from guardian

# Governance resumes them (submits a proposal)
bzed tx guardian resume-automation --deposit 1000000ubze --title "Resume protocol automation" --summary "..." --from mykey
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
- `bzed query circuit disabled-list`: the disabled message types (gRPC `cosmos.circuit.v1.Query/DisabledList`, REST `/cosmos/circuit/v1/disable_list`).
- `bzed query tradebin halted-denoms` and `bzed query tradebin denom-halted [denom]`: the denoms halted on the DEX, by the guardian or by governance (one list).

- `bzed query guardian automation-paused`: whether the protocol automation is paused (`/bze/guardian/v1/automation_paused` over REST). The flag is part of the module genesis (`automation_paused`), so an export and import keeps it.

## Events
- `TightenRateLimitEvent` (signer, denom, channel, old and new percents), `DenomBlockedEvent` (`signer`, `denom`), `DenomUnblockedEvent` (`authority`, `denom`) and `MessagesDisabledEvent` (`signer`, `msg_type_urls`). A reset emits `x/circuit`'s `reset_circuit_breaker` event.
- `TradingDenomsHaltedEvent` (`signer`, `denoms`): one per `MsgHaltTradingDenoms`, listing the denoms it halted (not those already halted). Each of them also gets tradebin's `DenomHaltedEvent` (`denom`), the event a governance halt emits.
- `TightenRateLimitEvent` (signer, denom, channel, old and new percents), `DenomBlockedEvent` (`signer`, `denom`) and `DenomUnblockedEvent` (`authority`, `denom`).
- `ChainHaltScheduledEvent` (`signer`, `name`, `height`, `reason`).
- `AutomationPausedEvent` (`signer`) and `AutomationResumedEvent` (`authority`).

## Guardian as a DAO
The guardian address can be the account of a static-list `x/dao` DAO (e.g. five members with `expedite_threshold` 0.5 and `expedite_quorum` 0.6, so three yes votes pass). Static-list DAOs are exempt from `min_expedite_delay`: an expedited proposal carrying a guardian message is settled (and run) as soon as the third member has voted, with `MsgSettleProposal` in the next tx or in the same tx as the last vote.
