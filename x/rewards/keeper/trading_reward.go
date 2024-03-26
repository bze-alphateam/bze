package keeper

import (
	"github.com/bze-alphateam/bze/x/rewards/types"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k Keeper) getActiveTradingRewardStore(ctx sdk.Context) prefix.Store {
	return prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ActiveTradingRewardKeyPrefix))
}

func (k Keeper) getPendingTradingRewardStore(ctx sdk.Context) prefix.Store {
	return prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.PendingTradingRewardKeyPrefix))
}

func (k Keeper) setTradingReward(store prefix.Store, tradingReward types.TradingReward) {
	b := k.cdc.MustMarshal(&tradingReward)
	store.Set(types.TradingRewardKey(tradingReward.RewardId), b)
}

func (k Keeper) getTradingReward(store prefix.Store, rewardId string) (val types.TradingReward, found bool) {
	b := store.Get(types.TradingRewardKey(rewardId))
	if b == nil {
		return val, false
	}

	k.cdc.MustUnmarshal(b, &val)

	return val, true
}

// RemovePendingTradingReward removes a tradingReward from the store
func (k Keeper) removeTradingReward(store prefix.Store, rewardId string) {
	store.Delete(types.TradingRewardKey(rewardId))
}

func (k Keeper) getAllTradingReward(store prefix.Store) (list []types.TradingReward) {
	iterator := sdk.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.TradingReward
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}

// SetPendingTradingReward set a specific tradingReward in the store from its index
func (k Keeper) SetPendingTradingReward(ctx sdk.Context, tradingReward types.TradingReward) {
	store := k.getPendingTradingRewardStore(ctx)
	k.setTradingReward(store, tradingReward)
}

// GetPendingTradingReward returns a tradingReward from its index
func (k Keeper) GetPendingTradingReward(ctx sdk.Context, rewardId string) (val types.TradingReward, found bool) {
	store := k.getPendingTradingRewardStore(ctx)

	return k.getTradingReward(store, rewardId)
}

// RemovePendingTradingReward removes a tradingReward from the store
func (k Keeper) RemovePendingTradingReward(ctx sdk.Context, rewardId string) {
	store := k.getPendingTradingRewardStore(ctx)
	k.removeTradingReward(store, rewardId)
}

// GetAllPendingTradingReward returns all tradingReward
func (k Keeper) GetAllPendingTradingReward(ctx sdk.Context) []types.TradingReward {
	store := k.getPendingTradingRewardStore(ctx)

	return k.getAllTradingReward(store)
}

// SetActiveTradingReward set a specific tradingReward in the store from its index
func (k Keeper) SetActiveTradingReward(ctx sdk.Context, tradingReward types.TradingReward) {
	store := k.getActiveTradingRewardStore(ctx)
	k.setTradingReward(store, tradingReward)
}

// GetActiveTradingReward returns a tradingReward from its index
func (k Keeper) GetActiveTradingReward(ctx sdk.Context, rewardId string) (val types.TradingReward, found bool) {
	store := k.getActiveTradingRewardStore(ctx)

	return k.getTradingReward(store, rewardId)
}

// RemoveActiveTradingReward removes a tradingReward from the store
func (k Keeper) RemoveActiveTradingReward(ctx sdk.Context, rewardId string) {
	store := k.getActiveTradingRewardStore(ctx)
	k.removeTradingReward(store, rewardId)
}

// GetAllActiveTradingReward returns all tradingReward
func (k Keeper) GetAllActiveTradingReward(ctx sdk.Context) []types.TradingReward {
	store := k.getActiveTradingRewardStore(ctx)

	return k.getAllTradingReward(store)
}

// SetMarketIdRewardId save a reward id on a market id key
func (k Keeper) SetMarketIdRewardId(ctx sdk.Context, marketIdRewardId types.MarketIdTradingRewardId) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.MarketIdRewardIdKeyPrefix))
	val := k.cdc.MustMarshal(&marketIdRewardId)
	store.Set(types.MarketIdRewardIdKey(marketIdRewardId.MarketId), val)
}

// GetMarketIdRewardId get a reward id for a market id key
func (k Keeper) GetMarketIdRewardId(ctx sdk.Context, marketId string) (val types.MarketIdTradingRewardId, found bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.MarketIdRewardIdKeyPrefix))
	valStorage := store.Get(types.MarketIdRewardIdKey(marketId))
	if valStorage == nil {
		return val, false
	}

	k.cdc.MustUnmarshal(valStorage, &val)

	return val, true
}

// RemoveMarketIdRewardId removes the reward id stored for a market id
func (k Keeper) RemoveMarketIdRewardId(ctx sdk.Context, marketId string) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.MarketIdRewardIdKeyPrefix))
	store.Delete(types.MarketIdRewardIdKey(marketId))
}

func (k Keeper) GetAllMarketIdRewardId(ctx sdk.Context) (list []types.MarketIdTradingRewardId) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.MarketIdRewardIdKeyPrefix))
	iterator := sdk.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.MarketIdTradingRewardId
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}
