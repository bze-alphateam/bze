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

// proposalBallots returns up to limit ballots of the proposal, in key order,
// starting after the ballot of the voter named by cursor (from the first one
// when cursor is empty), and whether more ballots follow them.
func (k Keeper) proposalBallots(ctx context.Context, proposalId uint64, cursor string, limit uint64) (ballots []types.Ballot, more bool, err error) {
	store := k.getPrefixedStore(ctx, types.KeyPrefix(types.BallotKeyPrefix))
	start := types.ProposalBallotsPrefix(proposalId)
	if cursor != "" {
		voter, err := sdk.AccAddressFromBech32(cursor)
		if err != nil {
			return nil, false, err
		}
		// the smallest key after the cursor's ballot
		start = append(types.BallotKey(proposalId, voter), 0)
	}

	iterator := store.Iterator(start, storetypes.PrefixEndBytes(types.ProposalBallotsPrefix(proposalId)))
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		if uint64(len(ballots)) == limit {
			return ballots, true, nil
		}

		var ballot types.Ballot
		k.cdc.MustUnmarshal(iterator.Value(), &ballot)
		ballots = append(ballots, ballot)
	}

	return ballots, false, nil
}

// removeBallot deletes a ballot and the vote-index entry of each of its shares.
func (k Keeper) removeBallot(ctx context.Context, ballot types.Ballot) {
	voter := sdk.MustAccAddressFromBech32(ballot.Voter)
	for _, share := range ballot.Shares {
		k.removeVoteIndex(ctx, []byte(share.Source), voter, ballot.ProposalId)
	}

	k.getPrefixedStore(ctx, types.KeyPrefix(types.BallotKeyPrefix)).Delete(types.BallotKey(ballot.ProposalId, voter))
}
