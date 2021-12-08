package keeper

import (
	"github.com/bze-alphateam/bzedgev5/x/scavenge/types"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// SetScavenge set a specific scavenge in the store from its index
func (k Keeper) SetScavenge(ctx sdk.Context, scavenge types.Scavenge) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ScavengeKeyPrefix))
	b := k.cdc.MustMarshal(&scavenge)
	store.Set(types.ScavengeKey(
		scavenge.Index,
	), b)
}

// GetScavenge returns a scavenge from its index
func (k Keeper) GetScavenge(
	ctx sdk.Context,
	index string,

) (val types.Scavenge, found bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ScavengeKeyPrefix))

	b := store.Get(types.ScavengeKey(
		index,
	))
	if b == nil {
		return val, false
	}

	k.cdc.MustUnmarshal(b, &val)
	return val, true
}

// RemoveScavenge removes a scavenge from the store
func (k Keeper) RemoveScavenge(
	ctx sdk.Context,
	index string,

) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ScavengeKeyPrefix))
	store.Delete(types.ScavengeKey(
		index,
	))
}

// GetAllScavenge returns all scavenge
func (k Keeper) GetAllScavenge(ctx sdk.Context) (list []types.Scavenge) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ScavengeKeyPrefix))
	iterator := sdk.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.Scavenge
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}
