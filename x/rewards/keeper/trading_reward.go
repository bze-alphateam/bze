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

// SetMarketIdRewardId save a reward id on a market id key
func (k Keeper) SetMarketIdRewardId(ctx sdk.Context, tradingReward types.TradingReward) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.MarketIdRewardIdKeyPrefix))
	val := []byte(tradingReward.RewardId)
	store.Set(types.MarketIdRewardIdKey(tradingReward.MarketId), val)
}

// GetMarketIdRewardId get a reward id for a market id key
func (k Keeper) GetMarketIdRewardId(ctx sdk.Context, marketId string) (string, bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.MarketIdRewardIdKeyPrefix))
	val := store.Get(types.MarketIdRewardIdKey(marketId))
	if val == nil {
		return "", false
	}

	return string(val), true
}

// RemoveMarketIdRewardId removes the reward id stored for a market id
func (k Keeper) RemoveMarketIdRewardId(ctx sdk.Context, marketId string) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.MarketIdRewardIdKeyPrefix))
	store.Delete(types.MarketIdRewardIdKey(marketId))
}

func (k Keeper) GetAllMarketIdRewardId(ctx sdk.Context) (list []string) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.MarketIdRewardIdKeyPrefix))
	iterator := sdk.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		list = append(list, string(iterator.Value()))
	}

	return
}

// SetTradingRewardExpiration save the reward id on the expiration key
func (k Keeper) SetTradingRewardExpiration(ctx sdk.Context, expiration types.TradingRewardExpiration) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.TradingRewardExpirationKeyPrefix))
	b := k.cdc.MustMarshal(&expiration)
	store.Set(types.TradingRewardExpirationKey(expiration.ExpireAt, expiration.RewardId), b)
}

func (k Keeper) RemoveTradingRewardExpiration(ctx sdk.Context, expireAt uint32, rewardId string) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.TradingRewardExpirationKeyPrefix))
	store.Delete(types.TradingRewardExpirationKey(expireAt, rewardId))
}

func (k Keeper) GetAllTradingRewardExpiration(ctx sdk.Context) (list []types.TradingRewardExpiration) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.TradingRewardExpirationKeyPrefix))
	iterator := sdk.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.TradingRewardExpiration
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}

func (k Keeper) GetAllTradingRewardExpirationByExpireAt(ctx sdk.Context, expireAt uint32) (list []types.TradingRewardExpiration) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.TradingRewardExpirationKeyPrefix))
	iterator := sdk.KVStorePrefixIterator(store, []byte(types.TradingRewardExpirationByExpireAtPrefix(expireAt)))

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.TradingRewardExpiration
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}
