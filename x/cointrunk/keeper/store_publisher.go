package keeper

import (
	storetypes "cosmossdk.io/store/types"
	"github.com/bze-alphateam/bze/x/cointrunk/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	CounterKey = "counter"
)

func (k Keeper) GetPublisher(ctx sdk.Context, addr string) (publisher types.Publisher, found bool) {
	store := k.getPrefixedStore(ctx, types.KeyPrefix(types.PublisherKeyPrefix))
	record := store.Get(types.PublisherKey(addr))
	if record == nil {
		return publisher, false
	}

	k.cdc.MustUnmarshal(record, &publisher)

	return publisher, true
}

func (k Keeper) GetAllPublisher(ctx sdk.Context) (list []types.Publisher) {
	store := k.getPrefixedStore(ctx, types.KeyPrefix(types.PublisherKeyPrefix))
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.Publisher
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}

func (k Keeper) SetPublisher(ctx sdk.Context, publisher types.Publisher) {
	store := k.getPrefixedStore(ctx, types.KeyPrefix(types.PublisherKeyPrefix))
	val := k.cdc.MustMarshal(&publisher)
	store.Set(
		types.PublisherKey(publisher.Address),
		val,
	)
}
