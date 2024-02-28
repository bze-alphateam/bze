package keeper

import (
	"encoding/binary"
	"github.com/bze-alphateam/bze/x/tradebin/types"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k Keeper) getQueueMessageStore(ctx sdk.Context) sdk.KVStore {
	return prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.QueueMessagePrefix))
}

func (k Keeper) getQueueMessageCounterStore(ctx sdk.Context) sdk.KVStore {
	return prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.QueueMessageCounterPrefix))
}

// SetQueueMessage set a specific market in the store from its index
func (k Keeper) SetQueueMessage(ctx sdk.Context, qm types.QueueMessage) {
	counter := k.GetQueueMessageCounter(ctx)
	qm.MessageId = k.zeroFillId(counter)
	qm.CreatedAt = ctx.BlockHeader().Time.Unix()

	store := k.getQueueMessageStore(ctx)
	b := k.cdc.MustMarshal(&qm)
	key := types.QueueMessageKey(qm.MessageId)
	store.Set(key, b)
	k.incrementQueueMessageCounter(ctx)
}

// GetAllQueueMessage returns all queue messages
func (k Keeper) GetAllQueueMessage(ctx sdk.Context) (list []types.QueueMessage) {
	store := k.getQueueMessageStore(ctx)
	iterator := sdk.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.QueueMessage
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}

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

func (k Keeper) decrementQueueMessageCounter(ctx sdk.Context) uint64 {
	counter := k.GetQueueMessageCounter(ctx)
	counter--
	k.SetQueueMessageCounter(ctx, counter)

	return counter
}
