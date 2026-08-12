package keeper

import (
	storetypes "cosmossdk.io/store/types"
	"github.com/bze-alphateam/bze/x/rewards/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// SetDenomRewardPrize sets a DenomRewardPrize (accumulator) in the store.
func (k Keeper) SetDenomRewardPrize(ctx sdk.Context, prize types.DenomRewardPrize) {
	store := k.getPrefixedStore(ctx, types.KeyPrefix(types.DenomRewardPrizeKeyPrefix))
	b := k.cdc.MustMarshal(&prize)
	store.Set(types.DenomRewardPrizeKey(prize.StakingDenom, prize.PrizeDenom), b)
}

// GetDenomRewardPrize returns a DenomRewardPrize from its (staking denom, prize denom) pair.
func (k Keeper) GetDenomRewardPrize(ctx sdk.Context, stakingDenom, prizeDenom string) (val types.DenomRewardPrize, found bool) {
	store := k.getPrefixedStore(ctx, types.KeyPrefix(types.DenomRewardPrizeKeyPrefix))

	b := store.Get(types.DenomRewardPrizeKey(stakingDenom, prizeDenom))
	if b == nil {
		return val, false
	}

	k.cdc.MustUnmarshal(b, &val)
	return val, true
}

// RemoveDenomRewardPrize removes a DenomRewardPrize from the store.
func (k Keeper) RemoveDenomRewardPrize(ctx sdk.Context, stakingDenom, prizeDenom string) {
	store := k.getPrefixedStore(ctx, types.KeyPrefix(types.DenomRewardPrizeKeyPrefix))
	store.Delete(types.DenomRewardPrizeKey(stakingDenom, prizeDenom))
}

// IterateDenomRewardPrizes iterates every prize (accumulator) of a staking denom in
// deterministic lexicographic order. Bounded by max_prize_denoms_per_dr.
func (k Keeper) IterateDenomRewardPrizes(ctx sdk.Context, stakingDenom string, msgHandler func(ctx sdk.Context, prize types.DenomRewardPrize) (stop bool)) {
	store := k.getPrefixedStore(ctx, types.DenomRewardPrizePrefix(stakingDenom))
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var prize types.DenomRewardPrize
		k.cdc.MustUnmarshal(iterator.Value(), &prize)
		if msgHandler(ctx, prize) {
			break
		}
	}
}

// GetAllDenomRewardPrizes returns every prize of a staking denom.
func (k Keeper) GetAllDenomRewardPrizes(ctx sdk.Context, stakingDenom string) (list []types.DenomRewardPrize) {
	k.IterateDenomRewardPrizes(ctx, stakingDenom, func(_ sdk.Context, prize types.DenomRewardPrize) bool {
		list = append(list, prize)
		return false
	})

	return
}

// CountDenomRewardPrizes counts the prizes of a staking denom (used to enforce the cap).
func (k Keeper) CountDenomRewardPrizes(ctx sdk.Context, stakingDenom string) uint32 {
	store := k.getPrefixedStore(ctx, types.DenomRewardPrizePrefix(stakingDenom))
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})
	defer iterator.Close()

	var count uint32
	for ; iterator.Valid(); iterator.Next() {
		count++
	}

	return count
}

// GetAllDenomRewardPrize returns all DenomRewardPrize records across every staking denom
// (genesis export).
func (k Keeper) GetAllDenomRewardPrize(ctx sdk.Context) (list []types.DenomRewardPrize) {
	store := k.getPrefixedStore(ctx, types.KeyPrefix(types.DenomRewardPrizeKeyPrefix))
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.DenomRewardPrize
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}
