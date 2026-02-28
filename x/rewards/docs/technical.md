# Rewards Module – Technical Notes

## Staking Rewards
- **Lifecycle:** Created with prize funds escrowed in the module account; optional lock prevents immediate unstaking. Duration is in days (up to 10 years); reward IDs are zero-padded counters.
- **Participation:** Each join stores participant amount and the distributed stake height at join time. Claims/exit settle pending rewards proportionally to the participant’s share of the distributed stake.
- **Distribution:** Prize amounts can be topped up via `MsgDistributeStakingRewards`. Accrual is calculated against `DistributedStake`; creators extending duration via `MsgUpdateStakingReward` must fund additional prize coins up front.
- **Locks/Exit:** `MsgExitStaking` initiates withdrawal; locked stakes respect the configured `lock` period before release.

## Trading Rewards
- **Pending → Active:** Created rewards stay pending until `MsgActivateTradingReward` (authority) moves them to the active set. Expiration timestamps are refreshed on activation.
- **Market binding:** Only one active trading reward per `market_id`; creation fails if one already exists. The reward stores `slots` (payout rounds) and `duration` (hours) along with total prize to distribute.
- **Expiration/cleanup:** Expirations are stored to clear pending/active rewards when they time out.

## Fees and Community Pool
- Creation fees are calculated by `getRewardCreationFee` from params and sent to the distribution module community pool during message handling.

## Queries and Storage
- Rewards, participants, counters, leaderboards, and expirations are kept under prefixed stores; gRPC routes expose staking/trading rewards and params for light-client/REST usage.
