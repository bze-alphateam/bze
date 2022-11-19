package keeper

import (
	"github.com/bze-alphateam/bze/x/cointrunk/types"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k Keeper) GetPublisher(ctx sdk.Context, index string) (publisher types.Publisher, found bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.PublisherKeyPrefix))
	record := store.Get(types.PublisherKey(index))
	if record == nil {
		return publisher, false
	}

	k.cdc.MustUnmarshal(record, &publisher)

	return publisher, true
}

func (k Keeper) GetAllPublisher(ctx sdk.Context) (list []types.Publisher) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.PublisherKeyPrefix))
	iterator := sdk.KVStorePrefixIterator(store, []byte{})
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.Publisher
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}

func (k Keeper) SetPublisher(ctx sdk.Context, publisher types.Publisher) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.PublisherKeyPrefix))
	val := k.cdc.MustMarshal(&publisher)
	store.Set(
		types.PublisherKey(publisher.Address),
		val,
	)
}

func (k Keeper) GetAcceptedDomain(ctx sdk.Context, index string) (acceptedDomain types.AcceptedDomain, found bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.AcceptedDomainKeyPrefix))
	record := store.Get(types.AcceptedDomainKey(index))
	if record == nil {
		return acceptedDomain, false
	}

	k.cdc.MustUnmarshal(record, &acceptedDomain)

	return acceptedDomain, true
}

func (k Keeper) GetAllAcceptedDomain(ctx sdk.Context) (list []types.AcceptedDomain) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.AcceptedDomainKeyPrefix))
	iterator := sdk.KVStorePrefixIterator(store, []byte{})
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.AcceptedDomain
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}

func (k Keeper) SetAcceptedDomain(ctx sdk.Context, acceptedDomain types.AcceptedDomain) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.AcceptedDomainKeyPrefix))
	record := k.cdc.MustMarshal(&acceptedDomain)
	store.Set(
		types.AcceptedDomainKey(acceptedDomain.Domain),
		record,
	)
}
