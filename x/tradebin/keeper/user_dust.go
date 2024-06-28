package keeper

import (
	"github.com/bze-alphateam/bze/x/tradebin/types"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k Keeper) getUserDustStore(ctx sdk.Context) sdk.KVStore {
	return prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.UserDustKeyPrefix))
}

func (k Keeper) GetAllUserDust(ctx sdk.Context) (list []types.UserDust) {
	store := k.getUserDustStore(ctx)
	iterator := sdk.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.UserDust
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}

func (k Keeper) GetUserDust(ctx sdk.Context, address, denom string) (ud types.UserDust, found bool) {
	store := k.getUserDustStore(ctx)

	key := types.UserDustKey(address, denom)
	b := store.Get(key)
	if b == nil {
		return ud, false
	}

	k.cdc.MustUnmarshal(b, &ud)

	return ud, true
}

func (k Keeper) SetUserDust(ctx sdk.Context, ud types.UserDust) {
	store := k.getUserDustStore(ctx)
	b := k.cdc.MustMarshal(&ud)
	key := types.UserDustKey(ud.Owner, ud.Denom)
	store.Set(key, b)
}

func (k Keeper) RemoveUserDust(ctx sdk.Context, ud types.UserDust) {
	store := k.getUserDustStore(ctx)
	key := types.UserDustKey(ud.Owner, ud.Denom)
	store.Delete(key)
}

func (k Keeper) GetUserDustByOwner(ctx sdk.Context, address string) (list []types.UserDust) {
	store := k.getUserDustStore(ctx)
	iterator := sdk.KVStorePrefixIterator(store, types.UserDustKeyAddressPrefix(address))

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.UserDust
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}
