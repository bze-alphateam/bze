package keeper

import (
	storetypes "cosmossdk.io/store/types"
	"github.com/bze-alphateam/bze/x/burner/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// SetIbcMoveLock stores the height of the last successful MsgMoveIbcLockedCoins of lock.Denom.
func (k Keeper) SetIbcMoveLock(ctx sdk.Context, lock types.IbcMoveLock) {
	store := k.getPrefixedStore(ctx, types.KeyPrefix(types.IbcMoveLockKeyPrefix))
	store.Set([]byte(lock.Denom), k.cdc.MustMarshal(&lock))
}

// GetIbcMoveLock returns the lock of denom, if any of its locked coins was ever moved.
func (k Keeper) GetIbcMoveLock(ctx sdk.Context, denom string) (val types.IbcMoveLock, found bool) {
	store := k.getPrefixedStore(ctx, types.KeyPrefix(types.IbcMoveLockKeyPrefix))
	b := store.Get([]byte(denom))
	if b == nil {
		return val, false
	}

	k.cdc.MustUnmarshal(b, &val)
	return val, true
}

// GetAllIbcMoveLocks returns every stored lock, in denom order. Used by the genesis export.
func (k Keeper) GetAllIbcMoveLocks(ctx sdk.Context) (list []types.IbcMoveLock) {
	store := k.getPrefixedStore(ctx, types.KeyPrefix(types.IbcMoveLockKeyPrefix))
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.IbcMoveLock
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}
