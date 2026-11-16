package keeper

import (
	"bytes"

	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	"github.com/bze-alphateam/bze/x/rewards/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k Keeper) getTradingRewardCandidatePurgeStore(ctx sdk.Context) prefix.Store {
	return k.getPrefixedStore(ctx, types.KeyPrefix(types.TradingRewardCandidatePurgeKeyPrefix))
}

// EnqueueTradingRewardCandidatePurge marks the candidates of an expired trading reward for removal.
func (k Keeper) EnqueueTradingRewardCandidatePurge(ctx sdk.Context, rewardId string) {
	k.getTradingRewardCandidatePurgeStore(ctx).Set(types.TradingRewardKey(rewardId), []byte(rewardId))
}

// RemoveTradingRewardCandidatePurge drops a reward from the candidates removal queue.
func (k Keeper) RemoveTradingRewardCandidatePurge(ctx sdk.Context, rewardId string) {
	k.getTradingRewardCandidatePurgeStore(ctx).Delete(types.TradingRewardKey(rewardId))
}

// GetAllTradingRewardCandidatePurge returns the reward ids waiting for their candidates to be removed.
func (k Keeper) GetAllTradingRewardCandidatePurge(ctx sdk.Context) []string {
	return k.getBatchTradingRewardCandidatePurge(ctx, -1)
}

// getBatchTradingRewardCandidatePurge returns at most limit queued reward ids in key order. A negative
// limit returns all of them.
func (k Keeper) getBatchTradingRewardCandidatePurge(ctx sdk.Context, limit int) (list []string) {
	iterator := storetypes.KVStorePrefixIterator(k.getTradingRewardCandidatePurgeStore(ctx), []byte{})

	defer iterator.Close()

	for ; iterator.Valid() && (limit < 0 || len(list) < limit); iterator.Next() {
		list = append(list, string(iterator.Value()))
	}

	return
}

func (k Keeper) getRewardCandidateStore(ctx sdk.Context) prefix.Store {
	return k.getPrefixedStore(ctx, types.KeyPrefix(types.RewardCandidateKeyPrefix))
}

// getBatchTradingRewardCandidateKeys returns at most limit store keys of a reward's candidates.
func (k Keeper) getBatchTradingRewardCandidateKeys(ctx sdk.Context, rewardId string, limit int) (keys [][]byte) {
	iterator := storetypes.KVStorePrefixIterator(k.getRewardCandidateStore(ctx), []byte(rewardId+"/"))

	defer iterator.Close()

	for ; iterator.Valid() && len(keys) < limit; iterator.Next() {
		keys = append(keys, bytes.Clone(iterator.Key()))
	}

	return
}
