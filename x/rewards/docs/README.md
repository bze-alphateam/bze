# Rewards Module – User Guide

The rewards module runs two incentive types:
- **Staking rewards:** creators lock a prize pool that is distributed to users who stake a specific denom for a set duration.
- **Trading rewards:** creators fund rewards tied to a specific market; trading activity competes for the prize.

## Staking Rewards
What you can do:
- **Create a reward** (`MsgCreateStakingReward`): lock a prize amount/denom, choose staking denom, duration (days), minimum stake, and optional lock period. A creation fee may apply.
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
- **Create a reward** (`MsgCreateTradingReward`): fund a prize pool/denom, pick target `market_id`, set duration and number of slots (payout rounds). A creation fee may apply.
- **Activate** (`MsgActivateTradingReward`): governance/authority-only step that moves a pending reward into the active set for its market.
- Traders earn according to on-chain trading volume/logic of the module; leaderboard and distribution run automatically from the funded pool.

Example:
```bash
bzed tx rewards create-trading-reward \
  200000000ubze ubze 7 <market-id> 10 \
  --from mykey
# Activation is performed by the module authority (typically governance).
```

## Queries
- `bzed query rewards staking-reward <id>` / `staking-rewards` – view a reward and participants.
- `bzed query rewards trading-reward <id>` / `trading-rewards` – view pending/active trading rewards and expirations.
- `bzed query rewards params` – view current fees for creating rewards.

## Permissions
- `MsgActivateTradingReward` and `MsgUpdateParams` are authority-only (governance).
- All other messages are open to users who can fund the required amounts.
