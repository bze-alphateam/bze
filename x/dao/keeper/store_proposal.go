package keeper

import (
	"context"
	"fmt"

	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

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

// removeOpenMarker takes the proposal out of its DAO's proposals in voting.
func (k Keeper) removeOpenMarker(ctx context.Context, proposal types.Proposal) {
	k.getPrefixedStore(ctx, types.KeyPrefix(types.OpenProposalKeyPrefix)).Delete(types.DaoProposalKey(proposal.DaoId, proposal.Id))
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

// removeProposal deletes a proposal record and its DAO index entry.
func (k Keeper) removeProposal(ctx context.Context, proposal types.Proposal) {
	k.getPrefixedStore(ctx, types.KeyPrefix(types.ProposalKeyPrefix)).Delete(types.ProposalKey(proposal.Id))
	k.getPrefixedStore(ctx, types.KeyPrefix(types.DaoProposalKeyPrefix)).Delete(types.DaoProposalKey(proposal.DaoId, proposal.Id))
}

// GetAllProposals returns every proposal, by id.
func (k Keeper) GetAllProposals(ctx context.Context) []types.Proposal {
	iterator := k.getPrefixedStore(ctx, types.KeyPrefix(types.ProposalKeyPrefix)).Iterator(nil, nil)
	defer iterator.Close()

	var proposals []types.Proposal
	for ; iterator.Valid(); iterator.Next() {
		var proposal types.Proposal
		k.cdc.MustUnmarshal(iterator.Value(), &proposal)
		proposals = append(proposals, proposal)
	}

	return proposals
}

// SetAllProposals imports proposal records (genesis): it stores each one with
// its DAO index entry, and a proposal in voting also with its open marker. The
// proposals must pass GenesisState.Validate.
func (k Keeper) SetAllProposals(ctx context.Context, proposals []types.Proposal) {
	for _, proposal := range proposals {
		if proposal.Status != types.ProposalStatus_VOTING {
			k.setProposal(ctx, proposal)
			k.getPrefixedStore(ctx, types.KeyPrefix(types.DaoProposalKeyPrefix)).Set(types.DaoProposalKey(proposal.DaoId, proposal.Id), []byte{})
			continue
		}

		k.setNewProposal(ctx, proposal)
	}
}

// CheckDepositEscrow checks I1 on import: the dao module account holds at
// least the deposits of the proposals in voting, the only ones not released
// yet, so each can still be paid out. A surplus is allowed: before the module
// existed its address was not blocked, so anyone could send it coins, and an
// exact match would let such a transfer halt the upgrade or any later import.
func (k Keeper) CheckDepositEscrow(ctx context.Context) error {
	escrowed := sdk.NewCoins()
	for _, proposal := range k.GetAllProposals(ctx) {
		if proposal.Status == types.ProposalStatus_VOTING && proposal.PinnedDeposit != nil {
			escrowed = escrowed.Add(*proposal.PinnedDeposit)
		}
	}

	if escrowed.IsZero() {
		return nil
	}

	balance := k.bankKeeper.GetAllBalances(ctx, authtypes.NewModuleAddress(types.ModuleName))
	if !balance.IsAllGTE(escrowed) {
		return fmt.Errorf("I1: dao module account holds %s, less than the deposits of the proposals in voting (%s)", balance, escrowed)
	}

	return nil
}
