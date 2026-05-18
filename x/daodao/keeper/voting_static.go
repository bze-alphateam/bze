package keeper

import (
	"context"
	"fmt"

	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/store/prefix"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"

	"github.com/bze-alphateam/bze/x/daodao/types"
)

// staticVotingBackend reads members directly from daodao's own store.
//
// Storage layout (see types/keys.go):
//
//	MemberKey(dao, addr) → uvarint(weight)
//	StaticTotalPowerKey(dao) → uvarint(sum of all weights)
//
// The cached total is kept in lock-step with the member-set mutations in
// applyStaticMembersInit / k.applyMemberUpdates so reads are O(1).
type staticVotingBackend struct {
	k Keeper
}

func (s staticVotingBackend) Power(ctx context.Context, dao types.Dao, addr sdk.AccAddress) (uint64, error) {
	store := runtime.KVStoreAdapter(s.k.storeService.OpenKVStore(ctx))
	bz := store.Get(types.MemberKey(dao.Id, addr))
	if bz == nil {
		return 0, nil
	}
	return sdk.BigEndianToUint64(bz), nil
}

func (s staticVotingBackend) TotalPower(ctx context.Context, dao types.Dao) (uint64, error) {
	return s.k.getStaticTotalPower(ctx, dao.Id), nil
}

// SnapshotAll iterates the DAO's member rows and writes one
// SnapshotPowerKey per member plus the SnapshotTotalKey total.
//
// Cost: O(N members). Bounded by MaxStaticMembers (= 10,000).
// Called from Epic 3's MsgCreateProposal.
func (s staticVotingBackend) SnapshotAll(ctx context.Context, dao types.Dao, snapshotID uint64) error {
	root := runtime.KVStoreAdapter(s.k.storeService.OpenKVStore(ctx))

	memberStore := prefix.NewStore(root, types.MembersIterationPrefix(dao.Id))
	iter := memberStore.Iterator(nil, nil)
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		addr := sdk.AccAddress(iter.Key())
		weight := sdk.BigEndianToUint64(iter.Value())
		root.Set(types.SnapshotPowerKey(dao.Id, snapshotID, addr), sdk.Uint64ToBigEndian(weight))
	}

	root.Set(types.SnapshotTotalKey(dao.Id, snapshotID),
		sdk.Uint64ToBigEndian(s.k.getStaticTotalPower(ctx, dao.Id)))
	return nil
}

// -------- Member-set mutation helpers --------

// applyStaticMembersInit writes the initial member list (from MsgCreateDao)
// and seeds StaticTotalPowerKey. Returns the computed total. Caller has
// already validated `members` via ValidateStaticMembers.
func (k Keeper) applyStaticMembersInit(ctx context.Context, daoID uint64, members []types.StaticMember) (uint64, error) {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	var total uint64
	for _, m := range members {
		addr, err := sdk.AccAddressFromBech32(m.Address)
		if err != nil {
			// ValidateStaticMembers should have caught this.
			return 0, fmt.Errorf("applyStaticMembersInit: invalid address %q: %w", m.Address, err)
		}
		store.Set(types.MemberKey(daoID, addr), sdk.Uint64ToBigEndian(m.Weight))
		// uint64 overflow check (per-DAO bound is 10k × max_uint64 weights —
		// realistic weights are small; guard anyway).
		next, ok := safeAddU64(total, m.Weight)
		if !ok {
			return 0, errorsmod.Wrap(types.ErrAmountOverflow, "static total weight overflow")
		}
		total = next
	}
	k.setStaticTotalPower(ctx, daoID, total)
	return total, nil
}

// applyMemberUpdates is the keeper-side worker for MsgUpdateMembers. It
// removes first, then upserts; maintains the cached total.
//
// Returns the post-update member count so the caller can enforce the
// non-empty invariant.
func (k Keeper) applyMemberUpdates(ctx context.Context, daoID uint64, add []types.StaticMember, remove []string) (postCount uint64, err error) {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	total := k.getStaticTotalPower(ctx, daoID)

	// Removes first. Reject if any removed address isn't a member; this is
	// stateful validation that ValidateBasic can't do.
	for _, a := range remove {
		addr, err := sdk.AccAddressFromBech32(a)
		if err != nil {
			return 0, errorsmod.Wrapf(types.ErrInvalidAddress, "remove %q: %s", a, err.Error())
		}
		key := types.MemberKey(daoID, addr)
		bz := store.Get(key)
		if bz == nil {
			return 0, errorsmod.Wrapf(types.ErrMemberNotFound, "address %s", addr.String())
		}
		w := sdk.BigEndianToUint64(bz)
		store.Delete(key)
		total -= w // safe: total was the sum that included w
	}

	// Then upserts. Each `add` is a full replacement (weight overwrites any
	// existing value). Validation already enforces weight > 0 and no
	// duplicates within `add`.
	for _, m := range add {
		addr, err := sdk.AccAddressFromBech32(m.Address)
		if err != nil {
			return 0, errorsmod.Wrapf(types.ErrInvalidAddress, "add %q: %s", m.Address, err.Error())
		}
		key := types.MemberKey(daoID, addr)
		// If the address already exists, subtract its old weight first.
		if bz := store.Get(key); bz != nil {
			total -= sdk.BigEndianToUint64(bz)
		}
		next, ok := safeAddU64(total, m.Weight)
		if !ok {
			return 0, errorsmod.Wrap(types.ErrAmountOverflow, "static total weight overflow")
		}
		total = next
		store.Set(key, sdk.Uint64ToBigEndian(m.Weight))
	}

	k.setStaticTotalPower(ctx, daoID, total)

	// Count post-update members (cheap; we just iterated removes and adds).
	count := uint64(0)
	mStore := prefix.NewStore(store, types.MembersIterationPrefix(daoID))
	it := mStore.Iterator(nil, nil)
	defer it.Close()
	for ; it.Valid(); it.Next() {
		count++
	}
	return count, nil
}

// -------- Static-total cache helpers --------

func (k Keeper) getStaticTotalPower(ctx context.Context, daoID uint64) uint64 {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	bz := store.Get(types.StaticTotalPowerKey(daoID))
	if bz == nil {
		return 0
	}
	return sdk.BigEndianToUint64(bz)
}

func (k Keeper) setStaticTotalPower(ctx context.Context, daoID uint64, total uint64) {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store.Set(types.StaticTotalPowerKey(daoID), sdk.Uint64ToBigEndian(total))
}

// -------- Paginated member listing for the Members query --------

func (k Keeper) PaginatedStaticMembers(ctx context.Context, daoID uint64, pageReq *query.PageRequest) ([]types.StaticMember, *query.PageResponse, error) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)),
		types.MembersIterationPrefix(daoID))

	var members []types.StaticMember
	pageRes, err := query.Paginate(store, pageReq, func(key, value []byte) error {
		members = append(members, types.StaticMember{
			Address: sdk.AccAddress(key).String(),
			Weight:  sdk.BigEndianToUint64(value),
		})
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	return members, pageRes, nil
}

// safeAddU64 returns a+b and a flag indicating no overflow.
func safeAddU64(a, b uint64) (uint64, bool) {
	s := a + b
	if s < a { // wrapped
		return 0, false
	}
	return s, true
}

// InitStaticMembers is the exported entry point for genesis import.
// Writes every member row in `members` for `daoID` and recomputes the
// cached StaticTotalPowerKey from the sum of weights. Mirrors what
// MsgCreateDao does at runtime for a freshly created STATIC DAO.
//
// Callers are responsible for batching: pass ALL of a DAO's members in
// one call so the cached total is correct. Calling it twice for the
// same DAO would re-set the cached total to the SECOND batch's sum,
// silently losing the first batch's contribution to the cache (the
// underlying rows would coexist if addresses don't collide).
func (k Keeper) InitStaticMembers(ctx context.Context, daoID uint64, members []types.StaticMember) error {
	_, err := k.applyStaticMembersInit(ctx, daoID, members)
	return err
}

// IterateAllStaticMembers walks every MemberKey row in the store (across
// all STATIC DAOs) and invokes cb for each. Returning true from cb stops
// iteration.
//
// Used by genesis export. The runtime never needs cross-DAO iteration —
// per-DAO `MembersIterationPrefix(daoID)` covers all production paths —
// but genesis needs the global view to emit a flat list.
//
// Key layout (after the global prefix is stripped by prefix.NewStore):
//
//	dao_id (8 bytes, big-endian) | addr.Bytes()
//
// Address bytes are variable-length (typically 20 for cosmos addresses)
// but always at least 1 byte. We split on the 8-byte boundary; anything
// shorter than 1+8 bytes after the prefix is malformed and skipped.
func (k Keeper) IterateAllStaticMembers(ctx context.Context, cb func(daoID uint64, addr sdk.AccAddress, weight uint64) (stop bool)) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)),
		types.MemberKeyPrefix)
	iter := store.Iterator(nil, nil)
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		key := iter.Key()
		if len(key) < 8+1 {
			continue
		}
		daoID := sdk.BigEndianToUint64(key[:8])
		addr := sdk.AccAddress(key[8:])
		weight := sdk.BigEndianToUint64(iter.Value())
		if cb(daoID, addr, weight) {
			return
		}
	}
}
