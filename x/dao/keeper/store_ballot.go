package keeper

import (
	"context"

	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/bze-alphateam/bze/x/dao/types"
)

// GetBallot returns the voter's ballot on the proposal.
func (k Keeper) GetBallot(ctx context.Context, proposalId uint64, voter sdk.AccAddress) (ballot types.Ballot, found bool) {
	bz := k.getPrefixedStore(ctx, types.KeyPrefix(types.BallotKeyPrefix)).Get(types.BallotKey(proposalId, voter))
	if bz == nil {
		return ballot, false
	}

	k.cdc.MustUnmarshal(bz, &ballot)

	return ballot, true
}

// HasVoteIndex reports whether the address's ballot on the proposal counts a
// share of the source.
func (k Keeper) HasVoteIndex(ctx context.Context, source []byte, addr sdk.AccAddress, proposalId uint64) bool {
	return k.getPrefixedStore(ctx, types.KeyPrefix(types.VoteIndexKeyPrefix)).Has(types.VoteIndexKey(source, addr, proposalId))
}

func (k Keeper) setBallot(ctx context.Context, voter sdk.AccAddress, ballot types.Ballot) {
	k.getPrefixedStore(ctx, types.KeyPrefix(types.BallotKeyPrefix)).Set(types.BallotKey(ballot.ProposalId, voter), k.cdc.MustMarshal(&ballot))
}

func (k Keeper) setVoteIndex(ctx context.Context, source []byte, addr sdk.AccAddress, proposalId uint64) {
	k.getPrefixedStore(ctx, types.KeyPrefix(types.VoteIndexKeyPrefix)).Set(types.VoteIndexKey(source, addr, proposalId), []byte{})
}

func (k Keeper) removeVoteIndex(ctx context.Context, source []byte, addr sdk.AccAddress, proposalId uint64) {
	k.getPrefixedStore(ctx, types.KeyPrefix(types.VoteIndexKeyPrefix)).Delete(types.VoteIndexKey(source, addr, proposalId))
}

// votedProposals returns the proposals whose ballot of the address counts a
// share of the source. The scan covers only this address's entries on this
// source, so its cost grows with the address's own ballots (D19).
func (k Keeper) votedProposals(ctx context.Context, source []byte, addr sdk.AccAddress) []uint64 {
	store := k.getPrefixedStore(ctx, types.KeyPrefix(types.VoteIndexKeyPrefix))
	iterator := storetypes.KVStorePrefixIterator(store, types.VoteIndexPrefix(source, addr))
	defer iterator.Close()

	var ids []uint64
	for ; iterator.Valid(); iterator.Next() {
		key := iterator.Key()
		ids = append(ids, sdk.BigEndianToUint64(key[len(key)-8:]))
	}

	return ids
}
