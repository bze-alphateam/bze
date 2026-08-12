package keeper

import (
	storetypes "cosmossdk.io/store/types"
	"github.com/bze-alphateam/bze/x/rewards/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// SetDenomReward sets a DenomReward in the store from its staking denom.
func (k Keeper) SetDenomReward(ctx sdk.Context, denomReward types.DenomReward) {
	store := k.getPrefixedStore(ctx, types.KeyPrefix(types.DenomRewardKeyPrefix))
	b := k.cdc.MustMarshal(&denomReward)
	store.Set(types.DenomRewardKey(denomReward.StakingDenom), b)
}

// GetDenomReward returns a DenomReward from its staking denom.
func (k Keeper) GetDenomReward(ctx sdk.Context, stakingDenom string) (val types.DenomReward, found bool) {
	store := k.getPrefixedStore(ctx, types.KeyPrefix(types.DenomRewardKeyPrefix))

	b := store.Get(types.DenomRewardKey(stakingDenom))
	if b == nil {
		return val, false
	}

	k.cdc.MustUnmarshal(b, &val)
	return val, true
}

// HasDenomReward returns true when a DenomReward exists for the staking denom.
func (k Keeper) HasDenomReward(ctx sdk.Context, stakingDenom string) bool {
	store := k.getPrefixedStore(ctx, types.KeyPrefix(types.DenomRewardKeyPrefix))

	return store.Has(types.DenomRewardKey(stakingDenom))
}

// RemoveDenomReward removes a DenomReward from the store.
func (k Keeper) RemoveDenomReward(ctx sdk.Context, stakingDenom string) {
	store := k.getPrefixedStore(ctx, types.KeyPrefix(types.DenomRewardKeyPrefix))
	store.Delete(types.DenomRewardKey(stakingDenom))
}

// GetAllDenomReward returns all DenomReward records.
func (k Keeper) GetAllDenomReward(ctx sdk.Context) (list []types.DenomReward) {
	store := k.getPrefixedStore(ctx, types.KeyPrefix(types.DenomRewardKeyPrefix))
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.DenomReward
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}

// IterateAllDenomRewards iterates every DenomReward, stopping early when the handler returns true.
func (k Keeper) IterateAllDenomRewards(ctx sdk.Context, msgHandler func(ctx sdk.Context, dr types.DenomReward) (stop bool)) {
	store := k.getPrefixedStore(ctx, types.KeyPrefix(types.DenomRewardKeyPrefix))
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var dr types.DenomReward
		k.cdc.MustUnmarshal(iterator.Value(), &dr)
		if msgHandler(ctx, dr) {
			break
		}
	}
}
