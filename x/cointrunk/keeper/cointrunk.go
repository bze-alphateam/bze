package keeper

import (
	"encoding/binary"
	"fmt"
	"github.com/bze-alphateam/bze/x/cointrunk/types"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
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

func (k Keeper) GetAllArticles(ctx sdk.Context) (list []types.Article) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ArticleKeyPrefix))
	iterator := sdk.KVStorePrefixIterator(store, []byte{})
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.Article
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}

func (k Keeper) SetArticle(ctx sdk.Context, article types.Article) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ArticleKeyPrefix))
	if article.Id == 0 {
		article.Id = k.incrementCounter(ctx, types.ArticleCounterKeyPrefix)
	}
	keyString := fmt.Sprintf("%012d", article.Id)

	record := k.cdc.MustMarshal(&article)
	store.Set(
		types.ArticleKey(keyString),
		record,
	)

	if article.Paid {
		k.incrementMonthlyPaidArticleCounter(ctx)
	}
}

func (k Keeper) GetMonthlyPaidArticleCounter(ctx sdk.Context) uint64 {
	return k.getCounter(ctx, types.GenerateMonthlyPaidArticleCounterPrefix(ctx))
}

func (k Keeper) GetArticleCounter(ctx sdk.Context) uint64 {
	return k.getCounter(ctx, types.ArticleCounterKeyPrefix)
}

func (k Keeper) SetArticleCounter(ctx sdk.Context, counter uint64) {
	counterStore := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ArticleCounterKeyPrefix))
	record := make([]byte, 8)
	binary.BigEndian.PutUint64(record, counter)

	counterStore.Set(
		types.ArticleKey(CounterKey),
		record,
	)
}

func (k Keeper) getCounter(ctx sdk.Context, storePrefix string) uint64 {
	counterStore := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(storePrefix))
	counter := counterStore.Get(types.ArticleKey(CounterKey))
	if counter == nil {
		return 0
	}

	return binary.BigEndian.Uint64(counter)
}

func (k Keeper) incrementMonthlyPaidArticleCounter(ctx sdk.Context) uint64 {
	return k.incrementCounter(ctx, types.GenerateMonthlyPaidArticleCounterPrefix(ctx))
}

func (k Keeper) incrementCounter(ctx sdk.Context, storePrefix string) uint64 {
	no := k.getCounter(ctx, storePrefix)
	no++
	counterStore := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(storePrefix))
	record := make([]byte, 8)
	binary.BigEndian.PutUint64(record, no)

	counterStore.Set(
		types.ArticleKey(CounterKey),
		record,
	)

	return no
}
