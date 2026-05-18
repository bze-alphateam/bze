package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/store/prefix"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"

	"github.com/bze-alphateam/bze/x/daodao/types"
)

// ----- Proposal counter -----

// PeekNextProposalID returns the next proposal id for a DAO without advancing
// the counter. Returns 1 if the counter is unset.
func (k Keeper) PeekNextProposalID(ctx context.Context, daoID uint64) uint64 {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	bz := store.Get(types.ProposalIDCounterKey(daoID))
	if bz == nil {
		return 1
	}
	return sdk.BigEndianToUint64(bz)
}

// ConsumeNextProposalID returns the next proposal id and advances the counter.
func (k Keeper) ConsumeNextProposalID(ctx context.Context, daoID uint64) uint64 {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	id := k.PeekNextProposalID(ctx, daoID)
	store.Set(types.ProposalIDCounterKey(daoID), sdk.Uint64ToBigEndian(id+1))
	return id
}

// SetProposalIDCounter overrides the next-proposal-id for a DAO. Used by
// genesis import; production code should go through ConsumeNextProposalID.
func (k Keeper) SetProposalIDCounter(ctx context.Context, daoID, next uint64) {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store.Set(types.ProposalIDCounterKey(daoID), sdk.Uint64ToBigEndian(next))
}

// IterateProposalIDCounters walks every per-DAO ProposalIDCounter row and
// invokes `cb` for each. Returning true from `cb` stops iteration. Used
// by genesis export to serialize the per-DAO map.
func (k Keeper) IterateProposalIDCounters(ctx context.Context, cb func(daoID, counter uint64) (stop bool)) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)),
		types.ProposalIDCounterKeyPrefix)
	iter := store.Iterator(nil, nil)
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		daoID := sdk.BigEndianToUint64(iter.Key())
		counter := sdk.BigEndianToUint64(iter.Value())
		if cb(daoID, counter) {
			return
		}
	}
}

// IterateAllProposals walks every proposal in the store (across all DAOs)
// in (dao_id, proposal_id) key order. Used by genesis export.
func (k Keeper) IterateAllProposals(ctx context.Context, cb func(types.Proposal) (stop bool)) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)),
		types.ProposalKeyPrefix)
	iter := store.Iterator(nil, nil)
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		var p types.Proposal
		k.cdc.MustUnmarshal(iter.Value(), &p)
		if cb(p) {
			return
		}
	}
}

// ----- Proposal CRUD with status-index maintenance -----

// SetProposalNew persists a brand-new proposal (called only at create time).
// Writes the Proposal record AND the status-index row.
func (k Keeper) SetProposalNew(ctx context.Context, p types.Proposal) {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	bz := k.cdc.MustMarshal(&p)
	store.Set(types.ProposalKey(p.DaoId, p.ProposalId), bz)
	store.Set(types.ProposalByStatusKey(p.DaoId, p.Status, p.ProposalId), []byte{})
}

// UpdateProposal rewrites an existing proposal. When `newStatus` differs from
// the proposal's previous on-disk status, the status-index row is migrated
// atomically (delete old, write new). Returns the persisted Proposal so the
// caller can avoid an extra read.
//
// Callers must pass the proposal with the new status already set. The
// previous status comes from the stored row, NOT from the in-memory copy
// passed in — this lets a caller mutate `p.Status` freely before calling.
func (k Keeper) UpdateProposal(ctx context.Context, p types.Proposal) error {
	prev, ok := k.GetProposal(ctx, p.DaoId, p.ProposalId)
	if !ok {
		return fmt.Errorf("update on missing proposal (dao=%d, id=%d)", p.DaoId, p.ProposalId)
	}
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	if prev.Status != p.Status {
		store.Delete(types.ProposalByStatusKey(p.DaoId, prev.Status, p.ProposalId))
		store.Set(types.ProposalByStatusKey(p.DaoId, p.Status, p.ProposalId), []byte{})
	}
	store.Set(types.ProposalKey(p.DaoId, p.ProposalId), k.cdc.MustMarshal(&p))
	return nil
}

// GetProposal returns a proposal by (dao_id, proposal_id).
func (k Keeper) GetProposal(ctx context.Context, daoID, proposalID uint64) (types.Proposal, bool) {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	bz := store.Get(types.ProposalKey(daoID, proposalID))
	if bz == nil {
		return types.Proposal{}, false
	}
	var p types.Proposal
	k.cdc.MustUnmarshal(bz, &p)
	return p, true
}

// PaginatedProposals returns the DAO's proposals (optionally filtered by
// status). When `statusFilter == PROPOSAL_STATUS_UNSPECIFIED` the listing
// is the full per-DAO range; otherwise it walks the status index and
// dereferences each entry back to its Proposal record.
func (k Keeper) PaginatedProposals(
	ctx context.Context,
	daoID uint64,
	statusFilter types.ProposalStatus,
	pageReq *query.PageRequest,
) ([]types.Proposal, *query.PageResponse, error) {
	root := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))

	if statusFilter == types.ProposalStatus_PROPOSAL_STATUS_UNSPECIFIED {
		store := prefix.NewStore(root, types.ProposalsIterationPrefix(daoID))
		var out []types.Proposal
		pageRes, err := query.Paginate(store, pageReq, func(_, value []byte) error {
			var p types.Proposal
			if err := k.cdc.Unmarshal(value, &p); err != nil {
				return err
			}
			out = append(out, p)
			return nil
		})
		if err != nil {
			return nil, nil, err
		}
		return out, pageRes, nil
	}

	// Status-filtered path: page over the status index, look each proposal up.
	store := prefix.NewStore(root, types.ProposalsByStatusIterationPrefix(daoID, statusFilter))
	var out []types.Proposal
	pageRes, err := query.Paginate(store, pageReq, func(key, _ []byte) error {
		pid := sdk.BigEndianToUint64(key)
		p, ok := k.GetProposal(ctx, daoID, pid)
		if !ok {
			return fmt.Errorf("status index points to missing proposal (dao=%d, id=%d)", daoID, pid)
		}
		out = append(out, p)
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	return out, pageRes, nil
}

// ----- Expiring-proposal queue -----

// proposalDeadlineNs returns the unix-nano deadline this proposal is
// currently enqueued against. Epic 4 introduces the DEPOSIT_PERIOD phase,
// so the relevant deadline depends on status:
//
//   DEPOSIT_PERIOD → deposit_deadline
//   VOTING         → voting_end
//
// Terminal statuses are not on the queue and have no relevant deadline;
// callers gate on status first. We don't store the deadline on the queue
// key itself (it's part of the key bytes), so the dequeue path needs to
// pass the SAME status the proposal had when it was enqueued.
func proposalDeadlineNs(p types.Proposal) (uint64, bool) {
	switch p.Status {
	case types.ProposalStatus_PROPOSAL_STATUS_DEPOSIT_PERIOD:
		return uint64(p.DepositDeadline.UnixNano()), true
	case types.ProposalStatus_PROPOSAL_STATUS_VOTING:
		return uint64(p.VotingEnd.UnixNano()), true
	default:
		return 0, false
	}
}

// EnqueueExpiringProposal writes the end-blocker finalization queue entry
// for a proposal at the deadline implied by its CURRENT status. Status
// must be DEPOSIT_PERIOD or VOTING — terminal proposals don't belong on
// the queue.
//
// Callers transitioning a proposal between phases must:
//   1. dequeue with the OLD status (proposalDeadlineNs uses status to
//      pick the right deadline);
//   2. update p.Status to the new phase;
//   3. enqueue with the NEW status.
func (k Keeper) EnqueueExpiringProposal(ctx context.Context, p types.Proposal) {
	unixNs, ok := proposalDeadlineNs(p)
	if !ok {
		return
	}
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store.Set(types.ExpiringProposalKey(unixNs, p.DaoId, p.ProposalId), []byte{})
}

// DequeueExpiringProposal removes the queue entry. Must be called with
// the proposal record reflecting its CURRENT (pre-transition) status so
// proposalDeadlineNs picks the correct deadline.
func (k Keeper) DequeueExpiringProposal(ctx context.Context, p types.Proposal) {
	unixNs, ok := proposalDeadlineNs(p)
	if !ok {
		return
	}
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store.Delete(types.ExpiringProposalKey(unixNs, p.DaoId, p.ProposalId))
}

// IterateExpiredProposals walks every queue entry with voting_end <= now
// (inclusive) and invokes `cb` for the resolved Proposal record. Returning
// true from `cb` stops iteration.
//
// `cb` must NOT mutate the queue while iterating; the keeper signals that
// by buffering deletes inside the end-blocker rather than calling
// DequeueExpiringProposal mid-iteration.
func (k Keeper) IterateExpiredProposals(ctx context.Context, now uint64, cb func(types.Proposal) (stop bool)) {
	root := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	iter := root.Iterator(types.ExpiringProposalKeyPrefix, types.ExpiringProposalUntilPrefix(now))
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		key := iter.Key()
		// Key layout: prefix(1) | unixNs(8) | dao_id(8) | proposal_id(8).
		// We only need (dao_id, proposal_id) to resolve the record.
		if len(key) != 1+8+8+8 {
			// Malformed entry — defensive skip. Should be unreachable given
			// the key helpers in types/keys.go.
			continue
		}
		daoID := sdk.BigEndianToUint64(key[1+8 : 1+8+8])
		proposalID := sdk.BigEndianToUint64(key[1+8+8:])
		p, ok := k.GetProposal(ctx, daoID, proposalID)
		if !ok {
			// Index points to a missing proposal — also defensive; should
			// be unreachable in normal flow.
			continue
		}
		if cb(p) {
			return
		}
	}
}
