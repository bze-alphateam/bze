# Guardian Module – User Guide

The guardian module is the emergency path for the IBC rate limits of the `ratelimit` module (`cosmos/ibc-apps` rate-limiting). It can only make IBC transfers stricter; loosening them is left to governance.

## Concepts
- **Guardian**: one address, kept in the module params (empty means only governance). The guardian or the governance authority signs the emergency messages. Only governance changes the guardian (`MsgUpdateParams`).
- **Closed channel**: a transfer channel on which every ICS-20 transfer is stopped by x/guardian, in both directions, for every denom. It is a flag of this module, not an IBC channel close: the channel stays `OPEN` in IBC and the light clients keep updating.
- **Blacklist**: the `ratelimit` module's denom blacklist. Every ICS-20 transfer of a blacklisted denom is refused in both directions on every channel, with or without a rate limit (outgoing transfers fail as transactions, incoming ones are refunded on the sending chain).

## Messages
- **Tighten a rate limit** (`MsgTightenRateLimit`, guardian or governance): lowers the send and receive percents of an existing rate limit. A raise of either percent is refused, no limit is created, the window length is unchanged and the flow already counted in the current window is kept. `0`/`0` stops the pair.
- **Block a denom** (`MsgBlockDenom`, guardian or governance): puts the denom on the blacklist. The blacklist matches the exact denom string, so the denom must exist on chain (have a supply), written exactly as the chain stores it: for an IBC token that is `ibc/` followed by the upper-case hash. A lower-case hash, a base name such as `uusdc`, a trace path such as `transfer/channel-3/uusdc` or an unknown denom is refused with `denom does not exist on chain`. Blocking an already blocked denom is refused.
- **Unblock a denom** (`MsgUnblockDenom`, governance only): takes the denom off the blacklist. Any entry can be removed, whether or not the denom still has a supply. The guardian can never reopen a denom.
- **Close a channel** (`MsgCloseChannel`, guardian or governance): stops every ICS-20 transfer on an existing transfer channel (`transfer` port; an unknown channel is refused, closing a closed channel is refused). On the closed channel:
  - a transfer sent from BZE fails as a transaction: nothing is escrowed and no rate limit quota is counted;
  - a transfer arriving on BZE is answered with an error acknowledgement, so the sending chain refunds its sender when the acknowledgement is relayed. This includes BZE tokens coming back home;
  - acknowledgements and timeouts of transfers sent before the close still go through, so those transfers are completed or refunded as usual.
- **Reopen a channel** (`MsgReopenChannel`, governance only): lets transfers through the channel again. Reopening a channel that is not closed is refused. The guardian can never reopen a channel.

Only the ICS-20 transfer stack is covered. Other IBC applications (interchain accounts host and controller) are not affected by a closed channel.

```bash
# As the guardian, stop every IBC transfer on channel-0, in both directions
bzed tx guardian close-channel channel-0 --from guardian

# Reopen it through a governance proposal
bzed tx guardian reopen-channel channel-0 --deposit 1000000ubze --title "Reopen channel-0" --summary "..." --from mykey

# As the guardian, stop every IBC transfer of an IBC denom (exact on-chain spelling)
bzed tx guardian block-denom ibc/6490A7EAB61059BFC1CDDEB05917DD70BDF3A611654162A1A47DB930D40D8AF4 --from guardian

# As the guardian, lower the ubze quotas on channel-0 to 3% send, 0% receive
bzed tx guardian tighten-rate-limit ubze channel-0 3 0 --from guardian
```

## Queries
- `bzed query guardian params`: the guardian address.
- `bzed query guardian closed-channels`: the closed transfer channels.
- `bzed query guardian blocked-denoms`: the blacklisted denoms (the `ratelimit` module has no CLI for its blacklist; its gRPC `AllBlacklistedDenoms` works too).

## Events
- `TightenRateLimitEvent` (signer, denom, channel, old and new percents), `DenomBlockedEvent` (`signer`, `denom`) and `DenomUnblockedEvent` (`authority`, `denom`).
- `ChannelClosedEvent` (`signer`, `channel_id`) and `ChannelReopenedEvent` (`authority`, `channel_id`).
