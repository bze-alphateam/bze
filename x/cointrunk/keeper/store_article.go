package keeper

import (
	"encoding/binary"
	"fmt"

	storetypes "cosmossdk.io/store/types"
	"github.com/bze-alphateam/bze/x/cointrunk/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k Keeper) GetAllArticles(ctx sdk.Context) (list []types.Article) {
	store := k.getPrefixedStore(ctx, types.KeyPrefix(types.ArticleKeyPrefix))
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.Article
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}

func (k Keeper) SetArticle(ctx sdk.Context, article types.Article) {
	store := k.getPrefixedStore(ctx, types.KeyPrefix(types.ArticleKeyPrefix))
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
	counterStore := k.getPrefixedStore(ctx, types.KeyPrefix(types.ArticleCounterKeyPrefix))
	record := make([]byte, 8)
	binary.BigEndian.PutUint64(record, counter)

	counterStore.Set(
		types.ArticleKey(CounterKey),
		record,
	)
}
