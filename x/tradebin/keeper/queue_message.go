package keeper

import (
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
	qm.MessageId = k.largeZeroFillId(counter)
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

func (k Keeper) RemoveQueueMessage(ctx sdk.Context, messageId string) {
	store := k.getQueueMessageStore(ctx)
	key := types.QueueMessageKey(messageId)
	store.Delete(key)
}

func (k Keeper) IterateAllQueueMessages(ctx sdk.Context, msgHandler func(ctx sdk.Context, message types.QueueMessage)) {
	store := ctx.KVStore(k.storeKey)
	iterator := sdk.KVStorePrefixIterator(store, types.KeyPrefix(types.QueueMessagePrefix))
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var msg types.QueueMessage
		k.cdc.MustUnmarshal(iterator.Value(), &msg)
		msgHandler(ctx, msg)
	}
}
