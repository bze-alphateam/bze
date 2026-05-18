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

// ----- Poll ID counter -----

// PeekNextPollID returns the next poll id for a DAO without advancing.
func (k Keeper) PeekNextPollID(ctx context.Context, daoID uint64) uint64 {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	bz := store.Get(types.PollIDCounterKey(daoID))
	if bz == nil {
		return 1
	}
	return sdk.BigEndianToUint64(bz)
}

// ConsumeNextPollID returns the next poll id and advances the counter.
func (k Keeper) ConsumeNextPollID(ctx context.Context, daoID uint64) uint64 {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	id := k.PeekNextPollID(ctx, daoID)
	store.Set(types.PollIDCounterKey(daoID), sdk.Uint64ToBigEndian(id+1))
	return id
}

// SetPollIDCounter overrides the counter. Genesis import only.
func (k Keeper) SetPollIDCounter(ctx context.Context, daoID, next uint64) {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store.Set(types.PollIDCounterKey(daoID), sdk.Uint64ToBigEndian(next))
}

// IteratePollIDCounters walks every per-DAO PollIDCounter row.
func (k Keeper) IteratePollIDCounters(ctx context.Context, cb func(daoID, counter uint64) (stop bool)) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)),
		types.PollIDCounterKeyPrefix)
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

// ----- Poll CRUD with status-index maintenance -----

// SetPollNew persists a brand-new poll AND its status-index row.
func (k Keeper) SetPollNew(ctx context.Context, p types.Poll) {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	bz := k.cdc.MustMarshal(&p)
	store.Set(types.PollKey(p.DaoId, p.PollId), bz)
	store.Set(types.PollByStatusKey(p.DaoId, p.Status, p.PollId), []byte{})
}

// UpdatePoll rewrites an existing poll, migrating the status-index row
// if the status changed. Mirrors UpdateProposal's contract.
func (k Keeper) UpdatePoll(ctx context.Context, p types.Poll) error {
	prev, ok := k.GetPoll(ctx, p.DaoId, p.PollId)
	if !ok {
		return fmt.Errorf("update on missing poll (dao=%d, id=%d)", p.DaoId, p.PollId)
	}
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	if prev.Status != p.Status {
		store.Delete(types.PollByStatusKey(p.DaoId, prev.Status, p.PollId))
		store.Set(types.PollByStatusKey(p.DaoId, p.Status, p.PollId), []byte{})
	}
	store.Set(types.PollKey(p.DaoId, p.PollId), k.cdc.MustMarshal(&p))
	return nil
}

// GetPoll returns a poll by (dao_id, poll_id).
func (k Keeper) GetPoll(ctx context.Context, daoID, pollID uint64) (types.Poll, bool) {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	bz := store.Get(types.PollKey(daoID, pollID))
	if bz == nil {
		return types.Poll{}, false
	}
	var p types.Poll
	k.cdc.MustUnmarshal(bz, &p)
	return p, true
}

// IterateAllPolls walks every poll in the store. Genesis export.
func (k Keeper) IterateAllPolls(ctx context.Context, cb func(types.Poll) (stop bool)) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)),
		types.PollKeyPrefix)
	iter := store.Iterator(nil, nil)
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		var p types.Poll
		k.cdc.MustUnmarshal(iter.Value(), &p)
		if cb(p) {
			return
		}
	}
}

// PaginatedPolls returns the DAO's polls, optionally filtered by status.
// status_filter == UNSPECIFIED → full per-DAO range; otherwise the
// status index drives the listing.
func (k Keeper) PaginatedPolls(ctx context.Context, daoID uint64, statusFilter types.PollStatus, pageReq *query.PageRequest) ([]types.Poll, *query.PageResponse, error) {
	root := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))

	if statusFilter == types.PollStatus_POLL_STATUS_UNSPECIFIED {
		store := prefix.NewStore(root, types.PollsIterationPrefix(daoID))
		var out []types.Poll
		pageRes, err := query.Paginate(store, pageReq, func(_, value []byte) error {
			var p types.Poll
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

	store := prefix.NewStore(root, types.PollsByStatusIterationPrefix(daoID, statusFilter))
	var out []types.Poll
	pageRes, err := query.Paginate(store, pageReq, func(key, _ []byte) error {
		pid := sdk.BigEndianToUint64(key)
		p, ok := k.GetPoll(ctx, daoID, pid)
		if !ok {
			return fmt.Errorf("poll status index points to missing poll (dao=%d, id=%d)", daoID, pid)
		}
		out = append(out, p)
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	return out, pageRes, nil
}

// ----- Expiring-poll queue -----

// pollDeadlineNs returns the unix-nano deadline the poll is currently
// enqueued against. Parallel to proposalDeadlineNs.
func pollDeadlineNs(p types.Poll) (uint64, bool) {
	switch p.Status {
	case types.PollStatus_POLL_STATUS_DEPOSIT_PERIOD:
		return uint64(p.DepositDeadline.UnixNano()), true
	case types.PollStatus_POLL_STATUS_VOTING:
		return uint64(p.VotingEnd.UnixNano()), true
	default:
		return 0, false
	}
}

// EnqueueExpiringPoll writes the end-blocker queue entry for a poll at
// the deadline implied by its CURRENT status. Status must be
// DEPOSIT_PERIOD or VOTING.
func (k Keeper) EnqueueExpiringPoll(ctx context.Context, p types.Poll) {
	unixNs, ok := pollDeadlineNs(p)
	if !ok {
		return
	}
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store.Set(types.ExpiringPollKey(unixNs, p.DaoId, p.PollId), []byte{})
}

// DequeueExpiringPoll removes the queue entry. Caller must pass the poll
// with its PRE-transition status so the deadline picker selects the
// right key.
func (k Keeper) DequeueExpiringPoll(ctx context.Context, p types.Poll) {
	unixNs, ok := pollDeadlineNs(p)
	if !ok {
		return
	}
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store.Delete(types.ExpiringPollKey(unixNs, p.DaoId, p.PollId))
}

// IterateExpiredPolls walks every queue entry with deadline <= now.
// Mirrors IterateExpiredProposals.
func (k Keeper) IterateExpiredPolls(ctx context.Context, now uint64, cb func(types.Poll) (stop bool)) {
	root := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	iter := root.Iterator(types.ExpiringPollKeyPrefix, types.ExpiringPollUntilPrefix(now))
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		key := iter.Key()
		// Key layout: prefix(1) | unixNs(8) | dao_id(8) | poll_id(8).
		if len(key) != 1+8+8+8 {
			continue
		}
		daoID := sdk.BigEndianToUint64(key[1+8 : 1+8+8])
		pollID := sdk.BigEndianToUint64(key[1+8+8:])
		p, ok := k.GetPoll(ctx, daoID, pollID)
		if !ok {
			continue
		}
		if cb(p) {
			return
		}
	}
}
