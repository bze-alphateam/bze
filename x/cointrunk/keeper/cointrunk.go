package keeper

import (
	"crypto/md5"
	"encoding/binary"
	"github.com/bze-alphateam/bze/x/cointrunk/types"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/gobuffalo/packr/v2/file/resolver/encoding/hex"
)

const (
	CounterKey = "counter"
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

func (k Keeper) SetArticle(ctx sdk.Context, article types.Article) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.GenerateArticlePrefix(ctx)))
	hash := md5.Sum([]byte(article.Url))
	keyString := hex.EncodeToString(hash[:])
	record := k.cdc.MustMarshal(&article)
	store.Set(
		types.ArticleKey(keyString),
		record,
	)

	k.incrementCounter(ctx)
}

func (k Keeper) GetCounter(ctx sdk.Context) (no uint64) {
	counterStore := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.GenerateArticleCountPrefix(ctx)))
	counter := counterStore.Get(types.ArticleKey(CounterKey))
	if counter == nil {
		return 0
	}

	return binary.BigEndian.Uint64(counter)
}

func (k Keeper) incrementCounter(ctx sdk.Context) {
	no := k.GetCounter(ctx)
	no++
	counterStore := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.GenerateArticleCountPrefix(ctx)))
	record := make([]byte, 8)
	binary.BigEndian.PutUint64(record, no)

	counterStore.Set(
		types.ArticleKey(CounterKey),
		record,
	)
}
