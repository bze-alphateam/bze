package keeper

import (
	"github.com/bze-alphateam/bze/x/rewards/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// ProcessTradingRewardCandidatePurgeQueue removes the candidates of expired trading rewards, at most
// types.MaxTradingRewardCandidateRemovalsPerBlock rows per block across all queued rewards. Each call
// starts over at the beginning of the reward's candidates: the rows removed earlier are gone, so no
// cursor is needed. A queued reward can't get new rows, because candidates are only written for the
// reward its market currently points to, and that link is removed when the reward is paid.
func (k Keeper) ProcessTradingRewardCandidatePurgeQueue(ctx sdk.Context) {
	budget := types.MaxTradingRewardCandidateRemovalsPerBlock

	// keys are collected before any delete: a store must not be changed while an iterator is open
	for _, rewardId := range k.getBatchTradingRewardCandidatePurge(ctx, budget) {
		keys := k.getBatchTradingRewardCandidateKeys(ctx, rewardId, budget)
		store := k.getRewardCandidateStore(ctx)
		for _, key := range keys {
			store.Delete(key)
		}
		budget -= len(keys)

		if len(k.getBatchTradingRewardCandidateKeys(ctx, rewardId, 1)) == 0 {
			k.RemoveTradingRewardCandidatePurge(ctx, rewardId)
		}

		if budget <= 0 {
			break
		}
	}
}

// GetTradingRewardCandidatesForExport returns the candidates to write in an exported genesis: all of
// them except those of rewards waiting for their candidates to be removed.
func (k Keeper) GetTradingRewardCandidatesForExport(ctx sdk.Context) []types.TradingRewardCandidate {
	purging := make(map[string]struct{})
	for _, rewardId := range k.GetAllTradingRewardCandidatePurge(ctx) {
		purging[rewardId] = struct{}{}
	}

	all := k.GetAllTradingRewardCandidate(ctx)
	if len(purging) == 0 {
		return all
	}

	list := make([]types.TradingRewardCandidate, 0, len(all))
	for _, candidate := range all {
		if _, ok := purging[candidate.RewardId]; !ok {
			list = append(list, candidate)
		}
	}

	return list
}
