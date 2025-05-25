package keeper

import (
	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	"github.com/bze-alphateam/bze/x/tradebin/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k Keeper) getQueueMessageStore(ctx sdk.Context) prefix.Store {
	return k.getPrefixedStore(ctx, types.KeyPrefix(types.QueueMessagePrefix))
}

func (k Keeper) getQueueMessageCounterStore(ctx sdk.Context) prefix.Store {
	return k.getPrefixedStore(ctx, types.KeyPrefix(types.QueueMessageCounterPrefix))
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
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})

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
	store := k.getPrefixedStore(ctx, types.KeyPrefix(types.QueueMessagePrefix))
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var msg types.QueueMessage
		k.cdc.MustUnmarshal(iterator.Value(), &msg)
		msgHandler(ctx, msg)
	}
}
