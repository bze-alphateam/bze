package keeper

import (
	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	"github.com/bze-alphateam/bze/x/cointrunk/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k Keeper) getPrefixedStore(ctx sdk.Context, p []byte) prefix.Store {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))

	return prefix.NewStore(storeAdapter, p)
}

func (k Keeper) GetAcceptedDomain(ctx sdk.Context, index string) (acceptedDomain types.AcceptedDomain, found bool) {
	store := k.getPrefixedStore(ctx, types.KeyPrefix(types.AcceptedDomainKeyPrefix))
	record := store.Get(types.AcceptedDomainKey(index))
	if record == nil {
		return acceptedDomain, false
	}

	k.cdc.MustUnmarshal(record, &acceptedDomain)

	return acceptedDomain, true
}

func (k Keeper) GetAllAcceptedDomain(ctx sdk.Context) (list []types.AcceptedDomain) {
	store := k.getPrefixedStore(ctx, types.KeyPrefix(types.AcceptedDomainKeyPrefix))
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.AcceptedDomain
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}

func (k Keeper) SetAcceptedDomain(ctx sdk.Context, acceptedDomain types.AcceptedDomain) {
	store := k.getPrefixedStore(ctx, types.KeyPrefix(types.AcceptedDomainKeyPrefix))
	record := k.cdc.MustMarshal(&acceptedDomain)
	store.Set(
		types.AcceptedDomainKey(acceptedDomain.Domain),
		record,
	)
}
