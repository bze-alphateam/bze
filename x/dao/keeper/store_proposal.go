package keeper

import (
	"context"

	storetypes "cosmossdk.io/store/types"

	"github.com/bze-alphateam/bze/x/dao/types"
)

// GetProposal returns the proposal with the given id.
func (k Keeper) GetProposal(ctx context.Context, proposalId uint64) (proposal types.Proposal, found bool) {
	bz := k.getPrefixedStore(ctx, types.KeyPrefix(types.ProposalKeyPrefix)).Get(types.ProposalKey(proposalId))
	if bz == nil {
		return proposal, false
	}

	k.cdc.MustUnmarshal(bz, &proposal)

	return proposal, true
}

// IsOpenProposal reports whether the proposal is marked as in voting for its DAO.
func (k Keeper) IsOpenProposal(ctx context.Context, daoId, proposalId uint64) bool {
	return k.getPrefixedStore(ctx, types.KeyPrefix(types.OpenProposalKeyPrefix)).Has(types.DaoProposalKey(daoId, proposalId))
}

// HasDaoProposalIndex reports whether the DAO's proposal index holds the proposal.
func (k Keeper) HasDaoProposalIndex(ctx context.Context, daoId, proposalId uint64) bool {
	return k.getPrefixedStore(ctx, types.KeyPrefix(types.DaoProposalKeyPrefix)).Has(types.DaoProposalKey(daoId, proposalId))
}

// setNewProposal stores a new proposal in voting with its open marker and its
// DAO index entry.
func (k Keeper) setNewProposal(ctx context.Context, proposal types.Proposal) {
	key := types.DaoProposalKey(proposal.DaoId, proposal.Id)

	k.setProposal(ctx, proposal)
	k.getPrefixedStore(ctx, types.KeyPrefix(types.OpenProposalKeyPrefix)).Set(key, []byte{})
	k.getPrefixedStore(ctx, types.KeyPrefix(types.DaoProposalKeyPrefix)).Set(key, []byte{})
}

// setProposal writes a proposal record.
func (k Keeper) setProposal(ctx context.Context, proposal types.Proposal) {
	k.getPrefixedStore(ctx, types.KeyPrefix(types.ProposalKeyPrefix)).Set(types.ProposalKey(proposal.Id), k.cdc.MustMarshal(&proposal))
}

// countOpenProposals counts the DAO's proposals in voting, stopping at limit:
// the caller only needs to know whether the cap is reached, so the scan never
// passes it.
func (k Keeper) countOpenProposals(ctx context.Context, daoId, limit uint64) uint64 {
	store := k.getPrefixedStore(ctx, types.KeyPrefix(types.OpenProposalKeyPrefix))
	iterator := storetypes.KVStorePrefixIterator(store, types.DaoProposalsPrefix(daoId))
	defer iterator.Close()

	count := uint64(0)
	for ; iterator.Valid() && count < limit; iterator.Next() {
		count++
	}

	return count
}
