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

// ---------- Counter ----------

// peekNextDaoID returns the next DAO id without consuming it. Used at create
// time to know what address.Module(...) derivation to use for validation
// before we commit any state.
func (k Keeper) peekNextDaoID(ctx context.Context) uint64 {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	bz := store.Get(types.DaoIDCounterKey)
	if bz == nil {
		return 1
	}
	return sdk.BigEndianToUint64(bz)
}

// consumeNextDaoID returns the next DAO id and advances the counter.
func (k Keeper) consumeNextDaoID(ctx context.Context) uint64 {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	id := k.peekNextDaoID(ctx)
	store.Set(types.DaoIDCounterKey, sdk.Uint64ToBigEndian(id+1))
	return id
}

// SetDaoIDCounter explicitly sets the counter. Used by genesis import.
func (k Keeper) SetDaoIDCounter(ctx context.Context, next uint64) {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store.Set(types.DaoIDCounterKey, sdk.Uint64ToBigEndian(next))
}

// GetDaoIDCounter reads the next-id counter (for genesis export).
func (k Keeper) GetDaoIDCounter(ctx context.Context) uint64 {
	return k.peekNextDaoID(ctx)
}

// ---------- DAO CRUD ----------

// SetDao writes a DAO record. Indices (DaoByAddress, DaoByCreator, SubDao)
// are NOT written here — callers do that via setDaoIndices on initial
// persistence and the SetDao path during updates assumes indices already
// exist.
func (k Keeper) SetDao(ctx context.Context, dao types.Dao) {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	bz := k.cdc.MustMarshal(&dao)
	store.Set(types.DaoKey(dao.Id), bz)
}

// GetDao returns the DAO with the given id and whether it exists.
func (k Keeper) GetDao(ctx context.Context, id uint64) (types.Dao, bool) {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	bz := store.Get(types.DaoKey(id))
	if bz == nil {
		return types.Dao{}, false
	}
	var dao types.Dao
	k.cdc.MustUnmarshal(bz, &dao)
	return dao, true
}

// GetDaoByAddress resolves a DAO by its on-chain account_address.
func (k Keeper) GetDaoByAddress(ctx context.Context, addr sdk.AccAddress) (types.Dao, bool) {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	bz := store.Get(types.DaoByAddressKey(addr))
	if bz == nil {
		return types.Dao{}, false
	}
	id := sdk.BigEndianToUint64(bz)
	return k.GetDao(ctx, id)
}

// SetDaoIndices writes the secondary indices for a DAO. Called once at
// creation (and on genesis import for each DAO).
func (k Keeper) SetDaoIndices(ctx context.Context, dao types.Dao) error {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))

	addr, err := sdk.AccAddressFromBech32(dao.AccountAddress)
	if err != nil {
		return fmt.Errorf("dao %d: invalid account_address: %w", dao.Id, err)
	}
	store.Set(types.DaoByAddressKey(addr), sdk.Uint64ToBigEndian(dao.Id))

	creator, err := sdk.AccAddressFromBech32(dao.Creator)
	if err != nil {
		return fmt.Errorf("dao %d: invalid creator: %w", dao.Id, err)
	}
	store.Set(types.DaoByCreatorKey(creator, dao.Id), []byte{})

	if dao.ParentDaoId != 0 {
		store.Set(types.SubDaoKey(dao.ParentDaoId, dao.Id), []byte{})
	}
	return nil
}

// ---------- Listing ----------

// IterateDaos calls cb for every DAO in id order. cb may return true to stop.
func (k Keeper) IterateDaos(ctx context.Context, cb func(types.Dao) (stop bool)) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.DaoKeyPrefix)
	iter := store.Iterator(nil, nil)
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		var dao types.Dao
		k.cdc.MustUnmarshal(iter.Value(), &dao)
		if cb(dao) {
			return
		}
	}
}

// PaginatedDaos returns a paginated slice of all DAOs.
func (k Keeper) PaginatedDaos(ctx context.Context, pageReq *query.PageRequest) ([]types.Dao, *query.PageResponse, error) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.DaoKeyPrefix)

	var daos []types.Dao
	pageRes, err := query.Paginate(store, pageReq, func(_, value []byte) error {
		var dao types.Dao
		if err := k.cdc.Unmarshal(value, &dao); err != nil {
			return err
		}
		daos = append(daos, dao)
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	return daos, pageRes, nil
}

// PaginatedDaosByCreator returns DAOs created by `creator`.
func (k Keeper) PaginatedDaosByCreator(ctx context.Context, creator sdk.AccAddress, pageReq *query.PageRequest) ([]types.Dao, *query.PageResponse, error) {
	indexStore := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)),
		types.DaoByCreatorIterationPrefix(creator))

	var daos []types.Dao
	pageRes, err := query.Paginate(indexStore, pageReq, func(key, _ []byte) error {
		// key (relative to the prefix) is uvarint(id)
		id := sdk.BigEndianToUint64(key)
		dao, ok := k.GetDao(ctx, id)
		if !ok {
			return fmt.Errorf("index points to missing dao id=%d", id)
		}
		daos = append(daos, dao)
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	return daos, pageRes, nil
}

// PaginatedSubDaos returns the children of `parentID`.
func (k Keeper) PaginatedSubDaos(ctx context.Context, parentID uint64, pageReq *query.PageRequest) ([]types.Dao, *query.PageResponse, error) {
	indexStore := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)),
		types.SubDaoIterationPrefix(parentID))

	var daos []types.Dao
	pageRes, err := query.Paginate(indexStore, pageReq, func(key, _ []byte) error {
		id := sdk.BigEndianToUint64(key)
		dao, ok := k.GetDao(ctx, id)
		if !ok {
			return fmt.Errorf("subdao index points to missing dao id=%d", id)
		}
		daos = append(daos, dao)
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	return daos, pageRes, nil
}

// ---------- Parent chain walks ----------

// hasParentCycle returns true if walking parent_dao_id upward from
// `proposedParentID` would ever reach `selfID`. Used to reject cyclic
// parent assignments. The walk also caps at a sane depth to prevent
// runaway iteration on corrupt state.
func (k Keeper) hasParentCycle(ctx context.Context, selfID, proposedParentID uint64) bool {
	const maxDepth = 100
	cur := proposedParentID
	for i := 0; i < maxDepth; i++ {
		if cur == 0 {
			return false
		}
		if cur == selfID {
			return true
		}
		parent, ok := k.GetDao(ctx, cur)
		if !ok {
			return false
		}
		cur = parent.ParentDaoId
	}
	// Hit max depth — treat as a cycle to be safe.
	return true
}
