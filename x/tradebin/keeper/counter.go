package keeper

import (
	"encoding/binary"
	"github.com/bze-alphateam/bze/x/tradebin/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k Keeper) GetQueueMessageCounter(ctx sdk.Context) uint64 {
	counterStore := k.getQueueMessageCounterStore(ctx)
	counter := counterStore.Get(types.QueueMessageCounterKey())
	if counter == nil {
		return 0
	}

	return binary.BigEndian.Uint64(counter)
}

func (k Keeper) SetQueueMessageCounter(ctx sdk.Context, counter uint64) {
	counterStore := k.getQueueMessageCounterStore(ctx)
	record := make([]byte, 8)
	binary.BigEndian.PutUint64(record, counter)

	counterStore.Set(
		types.QueueMessageCounterKey(),
		record,
	)
}

func (k Keeper) incrementQueueMessageCounter(ctx sdk.Context) uint64 {
	counter := k.GetQueueMessageCounter(ctx)
	counter++
	k.SetQueueMessageCounter(ctx, counter)

	return counter
}

func (k Keeper) ResetQueueMessageCounter(ctx sdk.Context) {
	k.SetQueueMessageCounter(ctx, 0)
}

func (k Keeper) GetOrderCounter(ctx sdk.Context) uint64 {
	counterStore := k.getOrderCounterStore(ctx)
	counter := counterStore.Get(types.OrderCounterKey())
	if counter == nil {
		return 0
	}

	return binary.BigEndian.Uint64(counter)
}

func (k Keeper) SetOrderCounter(ctx sdk.Context, counter uint64) {
	counterStore := k.getOrderCounterStore(ctx)
	record := make([]byte, 8)
	binary.BigEndian.PutUint64(record, counter)

	counterStore.Set(
		types.OrderCounterKey(),
		record,
	)
}

func (k Keeper) incrementOrderCounter(ctx sdk.Context) uint64 {
	counter := k.GetOrderCounter(ctx)
	counter++
	k.SetOrderCounter(ctx, counter)

	return counter
}
