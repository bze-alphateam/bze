# Guardian Module – User Guide

The guardian module holds the chain's emergency switches: the IBC rate limits of the `ratelimit` module (`cosmos/ibc-apps` rate-limiting) and the `x/circuit` message circuit breaker. Every switch follows one rule: the guardian or governance can switch something **off**, only governance switches it back **on**, and the guardian can never lock governance or itself out.

| Switch | Off (guardian or governance) | On (governance only) |
|---|---|---|
| Lower an IBC quota | `MsgTightenRateLimit` | `ratelimit` `MsgUpdateRateLimit` |
| Block a denom on IBC | `MsgBlockDenom` | `MsgUnblockDenom` |
| Disable message types | `MsgDisableMessages` | `x/circuit` `MsgResetCircuitBreaker` |

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

# As the guardian, lower the ubze quotas on channel-0 to 3% send, 0% receive
bzed tx guardian tighten-rate-limit ubze channel-0 3 0 --from guardian
```

## Queries
- `bzed query guardian params`: the guardian address.
- `bzed query guardian blocked-denoms`: the blacklisted denoms (the `ratelimit` module has no CLI for its blacklist; its gRPC `AllBlacklistedDenoms` works too).
- `bzed query circuit disabled-list`: the disabled message types (gRPC `cosmos.circuit.v1.Query/DisabledList`, REST `/cosmos/circuit/v1/disable_list`).

## Events
- `TightenRateLimitEvent` (signer, denom, channel, old and new percents), `DenomBlockedEvent` (`signer`, `denom`), `DenomUnblockedEvent` (`authority`, `denom`) and `MessagesDisabledEvent` (`signer`, `msg_type_urls`). A reset emits `x/circuit`'s `reset_circuit_breaker` event.
