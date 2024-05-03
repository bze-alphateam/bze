package keeper

import (
	"encoding/binary"
	"github.com/bze-alphateam/bze/x/rewards/types"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k Keeper) getCounterStore(ctx sdk.Context) sdk.KVStore {
	return prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.CounterKey))
}

// GetCounter - returns a counter by key
func (k Keeper) GetCounter(ctx sdk.Context, key []byte) uint64 {
	counterStore := k.getCounterStore(ctx)
	counter := counterStore.Get(key)
	if counter == nil {
		return 0
	}

	return binary.BigEndian.Uint64(counter)
}

// SetCounter - sets counter in storage on specified key
func (k Keeper) SetCounter(ctx sdk.Context, key []byte, counter uint64) {
	counterStore := k.getCounterStore(ctx)
	record := make([]byte, 8)
	binary.BigEndian.PutUint64(record, counter)

	counterStore.Set(key, record)
}

// incrementCounter - increments the specified counter and returns the new value
func (k Keeper) incrementCounter(ctx sdk.Context, key []byte) uint64 {
	counter := k.GetCounter(ctx, key)
	counter++
	k.SetCounter(ctx, key, counter)

	return counter
}

func (k Keeper) GetStakingRewardsCounter(ctx sdk.Context) uint64 {
	return k.GetCounter(ctx, types.StakingRewardCounterKey())
}

func (k Keeper) SetStakingRewardsCounter(ctx sdk.Context, counter uint64) {
	k.SetCounter(ctx, types.StakingRewardCounterKey(), counter)
}

func (k Keeper) incrementStakingRewardsCounter(ctx sdk.Context) uint64 {
	return k.incrementCounter(ctx, types.StakingRewardCounterKey())
}

func (k Keeper) GetTradingRewardsCounter(ctx sdk.Context) uint64 {
	return k.GetCounter(ctx, types.TradingRewardCounterKey())
}

func (k Keeper) SetTradingRewardsCounter(ctx sdk.Context, counter uint64) {
	k.SetCounter(ctx, types.TradingRewardCounterKey(), counter)
}

func (k Keeper) incrementTradingRewardsCounter(ctx sdk.Context) uint64 {
	return k.incrementCounter(ctx, types.TradingRewardCounterKey())
}
