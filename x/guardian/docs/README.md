# Guardian Module – User Guide

The guardian module is the emergency path for the IBC rate limits of the `ratelimit` module (`cosmos/ibc-apps` rate-limiting) and for the protocol's own automatic fund movements. The guardian can only stop things; loosening or restarting them is left to governance.

## Concepts
- **Guardian**: one address, kept in the module params (empty means only governance). The guardian or the governance authority signs the emergency messages. Only governance changes the guardian (`MsgUpdateParams`).
- **Blacklist**: the `ratelimit` module's denom blacklist. Every ICS-20 transfer of a blacklisted denom is refused in both directions on every channel, with or without a rate limit (outgoing transfers fail as transactions, incoming ones are refunded on the sending chain).

## Messages
- **Tighten a rate limit** (`MsgTightenRateLimit`, guardian or governance): lowers the send and receive percents of an existing rate limit. A raise of either percent is refused, no limit is created, the window length is unchanged and the flow already counted in the current window is kept. `0`/`0` stops the pair.
- **Block a denom** (`MsgBlockDenom`, guardian or governance): puts the denom on the blacklist. The blacklist matches the exact denom string, so the denom must exist on chain (have a supply), written exactly as the chain stores it: for an IBC token that is `ibc/` followed by the upper-case hash. A lower-case hash, a base name such as `uusdc`, a trace path such as `transfer/channel-3/uusdc` or an unknown denom is refused with `denom does not exist on chain`. Blocking an already blocked denom is refused.
- **Unblock a denom** (`MsgUnblockDenom`, governance only): takes the denom off the blacklist. Any entry can be removed, whether or not the denom still has a supply. The guardian can never reopen a denom.
- **Pause the protocol automation** (`MsgPauseAutomation`, guardian or governance): stops the txfeecollector EndBlock (both the gas fee pass and the community pool pass: no swap, no send, no community pool funding) and the burner's periodic burn (no burn, no lock, no liquidity added). Every coin stays where it is, in `txfeecollector`, `txfeecollector_cp` and `burner`. Pausing an already paused automation is refused. Raffle draws and the cleanup of finished raffles are user-facing and keep running.
- **Resume the protocol automation** (`MsgResumeAutomation`, governance only): lifts the pause. Nothing is lost while paused: the txfeecollector passes read the collectors' balances, so the first EndBlock after the resume handles the coins collected during the pause; a burning week that ended during the pause still marks the periodic burn pending, and the first EndBlock after the resume burns the whole burner balance instead of waiting for the next burning week. The guardian can never resume.

## Switches

| Switch | Off (stop) | On (restart) |
|---|---|---|
| IBC rate limit quota | guardian or governance (`MsgTightenRateLimit`) | governance (`ratelimit` `MsgUpdateRateLimit`) |
| IBC denom block | guardian or governance (`MsgBlockDenom`) | governance (`MsgUnblockDenom`) |
| Protocol automation (txfeecollector EndBlock, burner periodic burn) | guardian or governance (`MsgPauseAutomation`) | governance (`MsgResumeAutomation`) |

```bash
# As the guardian, stop every IBC transfer of an IBC denom (exact on-chain spelling)
bzed tx guardian block-denom ibc/6490A7EAB61059BFC1CDDEB05917DD70BDF3A611654162A1A47DB930D40D8AF4 --from guardian

# As the guardian, lower the ubze quotas on channel-0 to 3% send, 0% receive
bzed tx guardian tighten-rate-limit ubze channel-0 3 0 --from guardian

# As the guardian, freeze the txfeecollector conversions and the periodic burn
bzed tx guardian pause-automation --from guardian

# Governance resumes them (submits a proposal)
bzed tx guardian resume-automation --deposit 1000000ubze --title "Resume protocol automation" --summary "..." --from mykey
```

## Queries
- `bzed query guardian params`: the guardian address.
- `bzed query guardian blocked-denoms`: the blacklisted denoms (the `ratelimit` module has no CLI for its blacklist; its gRPC `AllBlacklistedDenoms` works too).
- `bzed query guardian automation-paused`: whether the protocol automation is paused (`/bze/guardian/v1/automation_paused` over REST). The flag is part of the module genesis (`automation_paused`), so an export and import keeps it.

## Events
- `TightenRateLimitEvent` (signer, denom, channel, old and new percents), `DenomBlockedEvent` (`signer`, `denom`) and `DenomUnblockedEvent` (`authority`, `denom`), `AutomationPausedEvent` (`signer`) and `AutomationResumedEvent` (`authority`).
