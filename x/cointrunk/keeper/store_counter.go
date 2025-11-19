package keeper

import (
	"encoding/binary"

	"github.com/bze-alphateam/bze/x/cointrunk/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k Keeper) getCounter(ctx sdk.Context, storePrefix string) uint64 {
	counterStore := k.getPrefixedStore(ctx, types.KeyPrefix(storePrefix))
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
	counterStore := k.getPrefixedStore(ctx, types.KeyPrefix(storePrefix))
	record := make([]byte, 8)
	binary.BigEndian.PutUint64(record, no)

	counterStore.Set(
		types.ArticleKey(CounterKey),
		record,
	)

	return no
}
