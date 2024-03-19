package keeper

import (
	"github.com/bze-alphateam/bze/x/rewards/types"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// SetTradingRewardLeaderboard set a specific types.TradingRewardLeaderboard in the store from its index
func (k Keeper) SetTradingRewardLeaderboard(ctx sdk.Context, leaderboard types.TradingRewardLeaderboard) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.LeaderboardKeyPrefix))
	b := k.cdc.MustMarshal(&leaderboard)
	store.Set(types.TradingRewardKey(leaderboard.RewardId), b)
}

// GetTradingRewardLeaderboard returns a types.TradingRewardLeaderboard from its index
func (k Keeper) GetTradingRewardLeaderboard(ctx sdk.Context, rewardId string) (val types.TradingRewardLeaderboard, found bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.LeaderboardKeyPrefix))

	b := store.Get(types.TradingRewardKey(rewardId))
	if b == nil {
		return val, false
	}

	k.cdc.MustUnmarshal(b, &val)
	return val, true
}

// GetAllTradingRewardLeaderboard returns all []types.TradingRewardLeaderboard
func (k Keeper) GetAllTradingRewardLeaderboard(ctx sdk.Context) (list []types.TradingRewardLeaderboard) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.LeaderboardKeyPrefix))
	iterator := sdk.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.TradingRewardLeaderboard
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}

// SetTradingRewardCandidate set a specific types.TradingRewardCandidate in the store from its index
func (k Keeper) SetTradingRewardCandidate(ctx sdk.Context, entry types.TradingRewardCandidate) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.RewardCandidateKeyPrefix))
	b := k.cdc.MustMarshal(&entry)
	store.Set(types.TradingRewardCandidateKey(entry.RewardId, entry.Address), b)
}

func (k Keeper) GetTradingRewardCandidate(ctx sdk.Context, rewardId, address string) (val types.TradingRewardCandidate, found bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.RewardCandidateKeyPrefix))

	b := store.Get(types.TradingRewardCandidateKey(rewardId, address))
	if b == nil {
		return val, false
	}

	k.cdc.MustUnmarshal(b, &val)
	return val, true
}

// GetAllTradingRewardCandidate returns all []types.TradingRewardCandidate
func (k Keeper) GetAllTradingRewardCandidate(ctx sdk.Context) (list []types.TradingRewardCandidate) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.RewardCandidateKeyPrefix))
	iterator := sdk.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.TradingRewardCandidate
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}
