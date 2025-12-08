# Rewards Module Parameters

- **`create_staking_reward_fee`** (`sdk.Coin`, default `250000000000ubze`): Fee sent to the community pool when creating a staking reward.
- **`create_trading_reward_fee`** (`sdk.Coin`, default `250000000000ubze`): Fee sent to the community pool when creating a trading reward.

### How They’re Used
- `MsgCreateStakingReward` and `MsgCreateTradingReward` collect the corresponding fee from the creator (on top of the prize funds) and forward it to the community pool. If you don’t have enough balance for both the prize and the fee, the transaction fails.

### Updating
- Only the module authority (typically governance) can change params via `MsgUpdateParams`. Supply both fees in the message; partial updates are rejected.
