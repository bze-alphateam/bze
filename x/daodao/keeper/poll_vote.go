package keeper

import (
	"context"

	"cosmossdk.io/store/prefix"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"

	"github.com/bze-alphateam/bze/x/daodao/types"
)

// SetPollVote writes (or replaces) a PollVote.
func (k Keeper) SetPollVote(ctx context.Context, v types.PollVote) error {
	addr, err := sdk.AccAddressFromBech32(v.Voter)
	if err != nil {
		return err
	}
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store.Set(types.PollVoteKey(v.DaoId, v.PollId, addr), k.cdc.MustMarshal(&v))
	return nil
}

// GetPollVote returns a voter's PollVote, or (zero, false) if absent.
func (k Keeper) GetPollVote(ctx context.Context, daoID, pollID uint64, voter sdk.AccAddress) (types.PollVote, bool) {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	bz := store.Get(types.PollVoteKey(daoID, pollID, voter))
	if bz == nil {
		return types.PollVote{}, false
	}
	var v types.PollVote
	k.cdc.MustUnmarshal(bz, &v)
	return v, true
}

// PaginatedPollVotes returns all votes for a poll, paginated.
func (k Keeper) PaginatedPollVotes(ctx context.Context, daoID, pollID uint64, pageReq *query.PageRequest) ([]types.PollVote, *query.PageResponse, error) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)),
		types.PollVotesIterationPrefix(daoID, pollID))

	var votes []types.PollVote
	pageRes, err := query.Paginate(store, pageReq, func(_, value []byte) error {
		var v types.PollVote
		if err := k.cdc.Unmarshal(value, &v); err != nil {
			return err
		}
		votes = append(votes, v)
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	return votes, pageRes, nil
}

// IterateAllPollVotes walks every PollVote in the store. Genesis export.
func (k Keeper) IterateAllPollVotes(ctx context.Context, cb func(types.PollVote) (stop bool)) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)),
		types.PollVoteKeyPrefix)
	iter := store.Iterator(nil, nil)
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		var v types.PollVote
		k.cdc.MustUnmarshal(iter.Value(), &v)
		if cb(v) {
			return
		}
	}
}
