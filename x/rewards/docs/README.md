# Rewards Module – User Guide

The rewards module runs two incentive types:
- **Staking rewards:** creators lock a prize pool that is distributed to users who stake a specific denom for a set duration.
- **Trading rewards:** creators fund rewards tied to a specific market; trading activity competes for the prize.

## Staking Rewards
What you can do:
- **Create a reward** (`MsgCreateStakingReward`): lock a prize amount/denom, choose staking denom, duration (days), minimum stake, and optional lock period. A creation fee may apply (it’s swapped to native via the trade module and forwarded to the fee collector).
- **Extend a reward** (`MsgUpdateStakingReward`): add duration by supplying extra prize funds.
- **Join** (`MsgJoinStaking`): stake the required denom into the reward.
- **Claim earnings** (`MsgClaimStakingRewards`): withdraw accrued prize tokens without exiting.
- **Exit** (`MsgExitStaking`): leave the reward; unlocked stake returns after the lock period (if any) and pending rewards are settled.
- **Distribute funds** (`MsgDistributeStakingRewards`): creator tops up prize amount during the campaign.

Example (CLI):
```bash
# Create a reward: 100 ubze prize, stake ubze for 30 days, min stake 10 ubze, lock 0 days
bzed tx rewards create-staking-reward \
  100000000ubze ubze 30 10000000 0 \
  --from mykey

# Join with 50 ubze
bzed tx rewards join-staking <reward-id> 50000000 --from mykey

# Claim pending prize
bzed tx rewards claim-staking-rewards <reward-id> --from mykey
```

## Trading Rewards
What you can do:
- **Create a reward** (`MsgCreateTradingReward`): fund a prize pool/denom, pick target `market_id`, set duration and number of slots (payout rounds). A creation fee may apply (swapped to native and sent to the fee collector module).
- **Activate** (`MsgActivateTradingReward`): governance/authority-only step that moves a pending reward into the active set for its market.
- Traders earn according to on-chain trading volume/logic of the module; leaderboard and distribution run automatically from the funded pool.

Example:
```bash
bzed tx rewards create-trading-reward \
  200000000ubze ubze 7 <market-id> 10 \
  --from mykey
# Activation is performed by the module authority (typically governance).
```

## Staking Reward Lock & Unlock
When a staking reward has a lock period and you exit:
- **Lock = 0 days**: your stake returns immediately.
- **Lock > 0 days**: your stake is queued as a `PendingUnlockParticipant`. The module processes unlock queues hourly, releasing up to 100 participants per block.

## Trading Reward Lifecycle
- Pending trading rewards must be activated by governance within 30 days. Unactivated rewards expire and their funds are sent to the burner module.
- Only one active trading reward per market is allowed; creation fails if one already exists.
- Traders per reward are tracked on a leaderboard (sized by the reward's `slots` parameter), sorted by volume with timestamp-based tie-breaking.
- Distribution happens automatically at the end of the reward period.

## Queries
- `bzed query rewards staking-reward <id>` – view a single staking reward.
- `bzed query rewards staking-rewards` – list all staking rewards (paginated).
- `bzed query rewards staking-reward-participant <address>` – view rewards a user participates in.
- `bzed query rewards all-staking-reward-participants` – all participants across all rewards (paginated).
- `bzed query rewards trading-reward <id>` – view a single trading reward.
- `bzed query rewards trading-rewards [--state pending|active]` – filter trading rewards by state (paginated).
- `bzed query rewards trading-reward-leaderboard <id>` – top traders for a reward.
- `bzed query rewards market-trading-reward --market-id <id>` – active reward for a specific market.
- `bzed query rewards all-pending-unlock-participants` – participants queued for stake unlock (paginated).
- `bzed query rewards params` – view current fees for creating rewards.

## Permissions
- `MsgActivateTradingReward` and `MsgUpdateParams` are authority-only (governance).
- All other messages are open to users who can fund the required amounts.

## Version History

### v8.1.0
- All reward operations moved from synchronous epoch hooks to bounded queue-based processing at EndBlock (up to 100 items/block): unlock participants, staking reward distribution, and trading reward expiration
- Added `ExtraGasForExitStake` parameter (default 1,000,000 gas) consumed when exiting a stake
- Removed `DistrKeeper` dependency — creation fees now routed to `txfeecollector` module via `CaptureAndSwapUserFee` instead of `FundCommunityPool`
- Expired pending trading rewards now send uncaptured tokens to the burner module instead of community pool
- Leaderboard sorting fix: old code compared the same value for both items; fixed to properly compare `a.Amount` vs `b.Amount`
- Trading hook now validates `tradedAmount.IsPositive()` before processing
- Small reward protection: if calculated reward truncates to zero, `JoinedAt` is not updated
- Maximum staking reward update duration increased from 10 years to 100 years
- ConsensusVersion bumped from 3 to 4
