package keeper

import (
	"context"

	"cosmossdk.io/store/prefix"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"

	"github.com/bze-alphateam/bze/x/daodao/types"
)

// SetVote writes a Vote record. Used both for the initial vote and (when
// revote is allowed) for the in-place replacement.
func (k Keeper) SetVote(ctx context.Context, v types.Vote) error {
	addr, err := sdk.AccAddressFromBech32(v.Voter)
	if err != nil {
		return err
	}
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store.Set(types.VoteKey(v.DaoId, v.ProposalId, addr), k.cdc.MustMarshal(&v))
	return nil
}

// GetVote returns a voter's vote on a proposal, or false if no vote exists.
func (k Keeper) GetVote(ctx context.Context, daoID, proposalID uint64, voter sdk.AccAddress) (types.Vote, bool) {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	bz := store.Get(types.VoteKey(daoID, proposalID, voter))
	if bz == nil {
		return types.Vote{}, false
	}
	var v types.Vote
	k.cdc.MustUnmarshal(bz, &v)
	return v, true
}

// PaginatedVotes returns all votes for a proposal, paginated.
func (k Keeper) PaginatedVotes(ctx context.Context, daoID, proposalID uint64, pageReq *query.PageRequest) ([]types.Vote, *query.PageResponse, error) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)),
		types.VotesIterationPrefix(daoID, proposalID))

	var votes []types.Vote
	pageRes, err := query.Paginate(store, pageReq, func(_, value []byte) error {
		var v types.Vote
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

// IterateAllVotes walks every Vote in the store (across all proposals)
// in (dao_id, proposal_id, voter-bytes) key order. Used by genesis export.
func (k Keeper) IterateAllVotes(ctx context.Context, cb func(types.Vote) (stop bool)) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)),
		types.VoteKeyPrefix)
	iter := store.Iterator(nil, nil)
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		var v types.Vote
		k.cdc.MustUnmarshal(iter.Value(), &v)
		if cb(v) {
			return
		}
	}
}
