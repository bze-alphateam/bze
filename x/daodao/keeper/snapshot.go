package keeper

import (
	"context"

	"cosmossdk.io/store/prefix"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/bze-alphateam/bze/x/daodao/types"
)

// Snapshot id counters are per-DAO and monotonic. Epic 3's MsgCreateProposal
// is the only intended caller of CreateSnapshot, which peeks the next id,
// runs the backend's SnapshotAll, and only advances the counter on success.
// Epic 2 only provides the counter and the reader helpers; no production
// code calls them yet.

// PeekNextSnapshotID returns the next snapshot id without consuming it.
// Returns 1 if the counter is unset (i.e. no snapshots yet).
func (k Keeper) PeekNextSnapshotID(ctx context.Context, daoID uint64) uint64 {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	bz := store.Get(types.SnapshotIDCounterKey(daoID))
	if bz == nil {
		return 1
	}
	return sdk.BigEndianToUint64(bz)
}

// ConsumeNextSnapshotID returns the next snapshot id and advances the
// counter.
func (k Keeper) ConsumeNextSnapshotID(ctx context.Context, daoID uint64) uint64 {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	id := k.PeekNextSnapshotID(ctx, daoID)
	store.Set(types.SnapshotIDCounterKey(daoID), sdk.Uint64ToBigEndian(id+1))
	return id
}

// SnapshotPower reads the captured per-address voting power at a given
// snapshot. Returns 0 if the address has no row at this snapshot.
//
// Used by Epic 3's vote/tally code.
func (k Keeper) SnapshotPower(ctx context.Context, daoID, snapshotID uint64, addr sdk.AccAddress) uint64 {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	bz := store.Get(types.SnapshotPowerKey(daoID, snapshotID, addr))
	if bz == nil {
		return 0
	}
	return sdk.BigEndianToUint64(bz)
}

// SnapshotTotal reads the captured total voting power at a given snapshot.
func (k Keeper) SnapshotTotal(ctx context.Context, daoID, snapshotID uint64) uint64 {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	bz := store.Get(types.SnapshotTotalKey(daoID, snapshotID))
	if bz == nil {
		return 0
	}
	return sdk.BigEndianToUint64(bz)
}

// SetSnapshotIDCounter overrides the next-snapshot-id for a DAO. Used by
// genesis import; production code goes through CreateSnapshot.
func (k Keeper) SetSnapshotIDCounter(ctx context.Context, daoID, next uint64) {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store.Set(types.SnapshotIDCounterKey(daoID), sdk.Uint64ToBigEndian(next))
}

// IterateSnapshotIDCounters walks every per-DAO SnapshotIDCounter row.
// Used by genesis export.
func (k Keeper) IterateSnapshotIDCounters(ctx context.Context, cb func(daoID, counter uint64) (stop bool)) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)),
		types.SnapshotIDCounterKeyPrefix)
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

// SetSnapshotPower writes (dao, snap, addr) → power. Used by genesis import;
// production callers use backend.SnapshotAll.
func (k Keeper) SetSnapshotPower(ctx context.Context, daoID, snapshotID uint64, addr sdk.AccAddress, power uint64) {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store.Set(types.SnapshotPowerKey(daoID, snapshotID, addr), sdk.Uint64ToBigEndian(power))
}

// SetSnapshotTotal writes (dao, snap) → total. Genesis import only.
func (k Keeper) SetSnapshotTotal(ctx context.Context, daoID, snapshotID, total uint64) {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store.Set(types.SnapshotTotalKey(daoID, snapshotID), sdk.Uint64ToBigEndian(total))
}

// IterateAllSnapshotPowers walks every SnapshotPowerKey row in (dao, snap,
// address-bytes) order. Used by genesis export.
//
// We can't decompose the key cleanly into a fixed-size suffix because
// AccAddress.Bytes() varies in length, so we parse it from the wire key.
func (k Keeper) IterateAllSnapshotPowers(ctx context.Context, cb func(daoID, snapshotID uint64, addr sdk.AccAddress, power uint64) (stop bool)) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)),
		types.SnapshotPowerKeyPrefix)
	iter := store.Iterator(nil, nil)
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		key := iter.Key()
		// Layout (after prefix.NewStore strips the 1-byte global prefix):
		//   dao_id (8) | snapshot_id (8) | addr.Bytes()
		if len(key) < 8+8+1 {
			continue
		}
		daoID := sdk.BigEndianToUint64(key[0:8])
		snapID := sdk.BigEndianToUint64(key[8:16])
		addr := sdk.AccAddress(key[16:])
		power := sdk.BigEndianToUint64(iter.Value())
		if cb(daoID, snapID, addr, power) {
			return
		}
	}
}

// IterateAllSnapshotTotals walks every SnapshotTotalKey row. Genesis export.
func (k Keeper) IterateAllSnapshotTotals(ctx context.Context, cb func(daoID, snapshotID, total uint64) (stop bool)) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)),
		types.SnapshotTotalKeyPrefix)
	iter := store.Iterator(nil, nil)
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		key := iter.Key()
		if len(key) != 8+8 {
			continue
		}
		daoID := sdk.BigEndianToUint64(key[0:8])
		snapID := sdk.BigEndianToUint64(key[8:16])
		total := sdk.BigEndianToUint64(iter.Value())
		if cb(daoID, snapID, total) {
			return
		}
	}
}

// CreateSnapshot allocates the next snapshot id for the DAO, asks the
// appropriate backend to populate the rows, and only advances the per-DAO
// counter once SnapshotAll has succeeded. Order matters: a msg-server
// caller would get tx-level rollback for free, but direct keeper-helper
// callers (end-blockers, upgrade handlers, tests) that swallow the error
// would otherwise leak a counter advance with no matching snapshot rows.
//
// Public so Epic 3's MsgCreateProposal can call it.
func (k Keeper) CreateSnapshot(ctx context.Context, dao types.Dao) (uint64, error) {
	backend, err := k.backendFor(dao)
	if err != nil {
		return 0, err
	}
	id := k.PeekNextSnapshotID(ctx, dao.Id)
	if err := backend.SnapshotAll(ctx, dao, id); err != nil {
		return 0, err
	}
	// Advance the counter only after a successful SnapshotAll. We write the
	// new value directly rather than re-using ConsumeNextSnapshotID so the
	// peek/write semantics are visibly atomic in this function.
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store.Set(types.SnapshotIDCounterKey(dao.Id), sdk.Uint64ToBigEndian(id+1))
	return id, nil
}
