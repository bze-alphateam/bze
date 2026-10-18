package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/bze-alphateam/bze/x/dao/types"
)

// GetDao returns the DAO with the given id.
func (k Keeper) GetDao(ctx context.Context, daoId uint64) (dao types.Dao, found bool) {
	bz := k.getPrefixedStore(ctx, types.KeyPrefix(types.DaoKeyPrefix)).Get(types.DaoKey(daoId))
	if bz == nil {
		return dao, false
	}

	k.cdc.MustUnmarshal(bz, &dao)

	return dao, true
}

// GetDaoIdByAddress returns the id of the DAO owning the given account, if any.
func (k Keeper) GetDaoIdByAddress(ctx context.Context, daoAddr sdk.AccAddress) (uint64, bool) {
	bz := k.getPrefixedStore(ctx, types.KeyPrefix(types.DaoAddressKeyPrefix)).Get(types.DaoAddressKey(daoAddr))
	if bz == nil {
		return 0, false
	}

	return sdk.BigEndianToUint64(bz), true
}

// HasDaoAdminIndex reports whether the admin index holds the given admin/dao pair.
func (k Keeper) HasDaoAdminIndex(ctx context.Context, admin sdk.AccAddress, daoId uint64) bool {
	return k.getPrefixedStore(ctx, types.KeyPrefix(types.DaoAdminKeyPrefix)).Has(types.DaoAdminKey(admin, daoId))
}

// GetAllDaos returns every DAO, by id.
func (k Keeper) GetAllDaos(ctx context.Context) []types.Dao {
	iterator := k.getPrefixedStore(ctx, types.KeyPrefix(types.DaoKeyPrefix)).Iterator(nil, nil)
	defer iterator.Close()

	var daos []types.Dao
	for ; iterator.Valid(); iterator.Next() {
		var dao types.Dao
		k.cdc.MustUnmarshal(iterator.Value(), &dao)
		daos = append(daos, dao)
	}

	return daos
}

// SetAllDaos imports DAO records (genesis): it stores each one with its address
// and admin index entries and pins its electorate sources. The DAOs must pass
// GenesisState.Validate.
func (k Keeper) SetAllDaos(ctx context.Context, daos []types.Dao) {
	for _, dao := range daos {
		k.setNewDao(ctx, dao)
		for _, source := range electorateSources(dao.Electorate) {
			k.pinSource(ctx, source)
		}
	}
}

// setNewDao stores a new DAO with its address and admin index entries. The
// DAO's address and admin must be valid bech32 addresses.
func (k Keeper) setNewDao(ctx context.Context, dao types.Dao) {
	daoAddr := sdk.MustAccAddressFromBech32(dao.Address)
	admin := sdk.MustAccAddressFromBech32(dao.Admin)

	k.getPrefixedStore(ctx, types.KeyPrefix(types.DaoKeyPrefix)).Set(types.DaoKey(dao.Id), k.cdc.MustMarshal(&dao))
	k.getPrefixedStore(ctx, types.KeyPrefix(types.DaoAddressKeyPrefix)).Set(types.DaoAddressKey(daoAddr), sdk.Uint64ToBigEndian(dao.Id))
	k.getPrefixedStore(ctx, types.KeyPrefix(types.DaoAdminKeyPrefix)).Set(types.DaoAdminKey(admin, dao.Id), []byte{})
}

// setDao rewrites the record of an existing DAO. Its address and admin must be
// unchanged; setAdmin changes the admin.
func (k Keeper) setDao(ctx context.Context, dao types.Dao) {
	k.getPrefixedStore(ctx, types.KeyPrefix(types.DaoKeyPrefix)).Set(types.DaoKey(dao.Id), k.cdc.MustMarshal(&dao))
}

// setAdmin makes newAdmin the DAO's admin, clears any pending nomination and
// moves the DAO's admin index entry, in one place. newAdmin must be a valid
// bech32 address. It returns the updated DAO.
func (k Keeper) setAdmin(ctx context.Context, dao types.Dao, newAdmin sdk.AccAddress) types.Dao {
	oldAdmin := sdk.MustAccAddressFromBech32(dao.Admin)

	adminIndex := k.getPrefixedStore(ctx, types.KeyPrefix(types.DaoAdminKeyPrefix))
	adminIndex.Delete(types.DaoAdminKey(oldAdmin, dao.Id))
	adminIndex.Set(types.DaoAdminKey(newAdmin, dao.Id), []byte{})

	dao.Admin = newAdmin.String()
	dao.PendingAdmin = ""
	k.setDao(ctx, dao)

	return dao
}
