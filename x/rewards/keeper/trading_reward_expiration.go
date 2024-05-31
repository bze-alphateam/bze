package keeper

import (
	"github.com/bze-alphateam/bze/x/rewards/types"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k Keeper) getActiveTradingRewardExpirationStore(ctx sdk.Context) prefix.Store {
	return prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ActiveTradingRewardExpirationKeyPrefix))
}

func (k Keeper) getPendingTradingRewardExpirationStore(ctx sdk.Context) prefix.Store {
	return prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.PendingTradingRewardExpirationKeyPrefix))
}

func (k Keeper) setTradingRewardExpiration(store prefix.Store, expiration types.TradingRewardExpiration) {
	b := k.cdc.MustMarshal(&expiration)
	store.Set(types.TradingRewardExpirationKey(expiration.ExpireAt, expiration.RewardId), b)
}

func (k Keeper) removeTradingRewardExpiration(store prefix.Store, expireAt uint32, rewardId string) {
	store.Delete(types.TradingRewardExpirationKey(expireAt, rewardId))
}

func (k Keeper) getAllTradingRewardExpiration(store prefix.Store) (list []types.TradingRewardExpiration) {
	iterator := sdk.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.TradingRewardExpiration
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}

func (k Keeper) getAllTradingRewardExpirationByExpireAt(store prefix.Store, expireAt uint32) (list []types.TradingRewardExpiration) {
	iterator := sdk.KVStorePrefixIterator(store, []byte(types.TradingRewardExpirationByExpireAtPrefix(expireAt)))

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.TradingRewardExpiration
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}

// SetPendingTradingRewardExpiration save the reward id on the expiration key
func (k Keeper) SetPendingTradingRewardExpiration(ctx sdk.Context, expiration types.TradingRewardExpiration) {
	store := k.getPendingTradingRewardExpirationStore(ctx)
	k.setTradingRewardExpiration(store, expiration)
}

func (k Keeper) RemovePendingTradingRewardExpiration(ctx sdk.Context, expireAt uint32, rewardId string) {
	store := k.getPendingTradingRewardExpirationStore(ctx)
	k.removeTradingRewardExpiration(store, expireAt, rewardId)
}

func (k Keeper) GetAllPendingTradingRewardExpiration(ctx sdk.Context) []types.TradingRewardExpiration {
	store := k.getPendingTradingRewardExpirationStore(ctx)

	return k.getAllTradingRewardExpiration(store)
}

func (k Keeper) GetAllPendingTradingRewardExpirationByExpireAt(ctx sdk.Context, expireAt uint32) []types.TradingRewardExpiration {
	store := k.getPendingTradingRewardExpirationStore(ctx)

	return k.getAllTradingRewardExpirationByExpireAt(store, expireAt)
}

func (k Keeper) SetActiveTradingRewardExpiration(ctx sdk.Context, expiration types.TradingRewardExpiration) {
	store := k.getActiveTradingRewardExpirationStore(ctx)
	k.setTradingRewardExpiration(store, expiration)
}

func (k Keeper) RemoveActiveTradingRewardExpiration(ctx sdk.Context, expireAt uint32, rewardId string) {
	store := k.getActiveTradingRewardExpirationStore(ctx)
	k.removeTradingRewardExpiration(store, expireAt, rewardId)
}

func (k Keeper) GetAllActiveTradingRewardExpiration(ctx sdk.Context) []types.TradingRewardExpiration {
	store := k.getActiveTradingRewardExpirationStore(ctx)

	return k.getAllTradingRewardExpiration(store)
}

func (k Keeper) GetAllActiveTradingRewardExpirationByExpireAt(ctx sdk.Context, expireAt uint32) []types.TradingRewardExpiration {
	store := k.getActiveTradingRewardExpirationStore(ctx)

	return k.getAllTradingRewardExpirationByExpireAt(store, expireAt)
}
