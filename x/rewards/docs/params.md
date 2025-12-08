# Rewards Module Parameters

- **`create_staking_reward_fee`** (`sdk.Coin`, default `250000000000ubze`): Fee collected when creating a staking reward. It is captured via the trade module (swapped to native if needed) and forwarded to the fee collector’s community-pool account.
- **`create_trading_reward_fee`** (`sdk.Coin`, default `250000000000ubze`): Same behavior for trading rewards—captured/swapped then sent to the fee collector community-pool account.

### How They’re Used
- `MsgCreateStakingReward` and `MsgCreateTradingReward` collect the corresponding fee from the creator (on top of the prize funds), swap to native denom if needed, and forward it to the fee collector. If you don’t have enough balance for both the prize and the fee, the transaction fails.

### Updating
- Only the module authority (typically governance) can change params via `MsgUpdateParams`. Supply both fees in the message; partial updates are rejected.
