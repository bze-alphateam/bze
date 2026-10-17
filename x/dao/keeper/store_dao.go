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

// setNewDao stores a new DAO with its address and admin index entries. The
// DAO's address and admin must be valid bech32 addresses.
func (k Keeper) setNewDao(ctx context.Context, dao types.Dao) {
	daoAddr := sdk.MustAccAddressFromBech32(dao.Address)
	admin := sdk.MustAccAddressFromBech32(dao.Admin)

	k.getPrefixedStore(ctx, types.KeyPrefix(types.DaoKeyPrefix)).Set(types.DaoKey(dao.Id), k.cdc.MustMarshal(&dao))
	k.getPrefixedStore(ctx, types.KeyPrefix(types.DaoAddressKeyPrefix)).Set(types.DaoAddressKey(daoAddr), sdk.Uint64ToBigEndian(dao.Id))
	k.getPrefixedStore(ctx, types.KeyPrefix(types.DaoAdminKeyPrefix)).Set(types.DaoAdminKey(admin, dao.Id), []byte{})
}
