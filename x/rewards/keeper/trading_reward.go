package keeper

import (
	"github.com/bze-alphateam/bze/x/rewards/types"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// SetTradingReward set a specific tradingReward in the store from its index
func (k Keeper) SetTradingReward(ctx sdk.Context, tradingReward types.TradingReward) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.TradingRewardKeyPrefix))
	b := k.cdc.MustMarshal(&tradingReward)
	store.Set(types.TradingRewardKey(tradingReward.RewardId), b)
}

// GetTradingReward returns a tradingReward from its index
func (k Keeper) GetTradingReward(ctx sdk.Context, rewardId string) (val types.TradingReward, found bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.TradingRewardKeyPrefix))

	b := store.Get(types.TradingRewardKey(rewardId))
	if b == nil {
		return val, false
	}

	k.cdc.MustUnmarshal(b, &val)
	return val, true
}

// RemoveTradingReward removes a tradingReward from the store
func (k Keeper) RemoveTradingReward(ctx sdk.Context, rewardId string) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.TradingRewardKeyPrefix))
	store.Delete(types.TradingRewardKey(rewardId))
}

// GetAllTradingReward returns all tradingReward
func (k Keeper) GetAllTradingReward(ctx sdk.Context) (list []types.TradingReward) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.TradingRewardKeyPrefix))
	iterator := sdk.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.TradingReward
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}
