# Guardian Module – User Guide

The guardian module is the emergency path for the IBC rate limits of the `ratelimit` module (`cosmos/ibc-apps` rate-limiting). It can only make IBC transfers stricter; loosening them is left to governance.

## Concepts
- **Guardian**: one address, kept in the module params (empty means only governance). The guardian or the governance authority signs the emergency messages. Only governance changes the guardian (`MsgUpdateParams`).
- **Blacklist**: the `ratelimit` module's denom blacklist. Every ICS-20 transfer of a blacklisted denom is refused in both directions on every channel, with or without a rate limit (outgoing transfers fail as transactions, incoming ones are refunded on the sending chain).

## Messages
- **Tighten a rate limit** (`MsgTightenRateLimit`, guardian or governance): lowers the send and receive percents of an existing rate limit. A raise of either percent is refused, no limit is created, the window length is unchanged and the flow already counted in the current window is kept. `0`/`0` stops the pair.
- **Block a denom** (`MsgBlockDenom`, guardian or governance): puts the denom on the blacklist. The blacklist matches the exact denom string, so the denom must exist on chain (have a supply), written exactly as the chain stores it: for an IBC token that is `ibc/` followed by the upper-case hash. A lower-case hash, a base name such as `uusdc`, a trace path such as `transfer/channel-3/uusdc` or an unknown denom is refused with `denom does not exist on chain`. Blocking an already blocked denom is refused.
- **Unblock a denom** (`MsgUnblockDenom`, governance only): takes the denom off the blacklist. Any entry can be removed, whether or not the denom still has a supply. The guardian can never reopen a denom.
- **Halt denoms on the DEX** (`MsgHaltTradingDenoms`, guardian or governance): the same halt as the tradebin governance message `MsgHaltDenoms`, written to tradebin's halted denoms list. A halted denom gets no new order, fill, market, pool or liquidity add, no AMM swap involving it (user routes, module swaps, fee swaps) and is no longer accepted as a tx fee; cancelling orders, removing liquidity, bank sends and IBC transfers keep working. Up to 1000 valid denoms per message, no duplicates; the native denom is refused. A denom that is already halted is left as it is. **Turned back on only by governance**, with a proposal carrying tradebin's `MsgUnhaltDenoms`; the guardian cannot lift a halt.

```bash
# As the guardian, stop every IBC transfer of an IBC denom (exact on-chain spelling)
bzed tx guardian block-denom ibc/6490A7EAB61059BFC1CDDEB05917DD70BDF3A611654162A1A47DB930D40D8AF4 --from guardian

# As the guardian, halt two denoms on the DEX
bzed tx guardian halt-trading-denoms factory/bze1.../utoken ibc/6490A7EAB61059BFC1CDDEB05917DD70BDF3A611654162A1A47DB930D40D8AF4 --from guardian

# As the guardian, lower the ubze quotas on channel-0 to 3% send, 0% receive
bzed tx guardian tighten-rate-limit ubze channel-0 3 0 --from guardian
```

## Queries
- `bzed query guardian params`: the guardian address.
- `bzed query guardian blocked-denoms`: the blacklisted denoms (the `ratelimit` module has no CLI for its blacklist; its gRPC `AllBlacklistedDenoms` works too).
- `bzed query tradebin halted-denoms` and `bzed query tradebin denom-halted [denom]`: the denoms halted on the DEX, by the guardian or by governance (one list).

## Events
- `TightenRateLimitEvent` (signer, denom, channel, old and new percents), `DenomBlockedEvent` (`signer`, `denom`) and `DenomUnblockedEvent` (`authority`, `denom`).
- `TradingDenomsHaltedEvent` (`signer`, `denoms`): one per `MsgHaltTradingDenoms`, listing the denoms it halted (not those already halted). Each of them also gets tradebin's `DenomHaltedEvent` (`denom`), the event a governance halt emits.
