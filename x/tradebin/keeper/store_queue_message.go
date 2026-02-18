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

	// Store the message with composite key: {market-id}/{zero-filled-id}
	// This allows efficient lookups by market while maintaining temporal order within each market
	store := k.getQueueMessageStore(ctx)
	b := k.cdc.MustMarshal(&qm)
	key := types.QueueMessageKey(qm.MarketId, qm.MessageId)
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

func (k Keeper) RemoveQueueMessage(ctx sdk.Context, marketId, messageId string) {
	store := k.getQueueMessageStore(ctx)
	key := types.QueueMessageKey(marketId, messageId)
	store.Delete(key)
}

func (k Keeper) IterateAllQueueMessages(ctx sdk.Context, msgHandler func(ctx sdk.Context, message types.QueueMessage) bool) {
	store := k.getPrefixedStore(ctx, types.KeyPrefix(types.QueueMessagePrefix))
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var msg types.QueueMessage
		k.cdc.MustUnmarshal(iterator.Value(), &msg)
		if !msgHandler(ctx, msg) {
			break
		}
	}
}

// HasQueueMessages checks if there are any messages in the queue
// Returns true if at least one message exists, false otherwise
// This is an O(1) operation as it stops at the first message
func (k Keeper) HasQueueMessages(ctx sdk.Context) bool {
	store := k.getPrefixedStore(ctx, types.KeyPrefix(types.QueueMessagePrefix))
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})
	defer iterator.Close()

	return iterator.Valid()
}

// GetQueueMessagesByMarket returns all queue messages for a specific market
// This uses the composite key with market ID prefix for O(M) performance
// where M is the number of messages for this market, instead of O(N) where N is total messages across all markets
func (k Keeper) getPendingCancelStore(ctx sdk.Context) prefix.Store {
	return k.getPrefixedStore(ctx, types.KeyPrefix(types.PendingCancelPrefix))
}

func (k Keeper) HasPendingCancel(ctx sdk.Context, marketId, orderType, orderId string) bool {
	store := k.getPendingCancelStore(ctx)
	key := types.PendingCancelKey(marketId, orderType, orderId)
	return store.Has(key)
}

func (k Keeper) SetPendingCancel(ctx sdk.Context, marketId, orderType, orderId string) {
	store := k.getPendingCancelStore(ctx)
	key := types.PendingCancelKey(marketId, orderType, orderId)
	store.Set(key, []byte{1})
}

func (k Keeper) RemovePendingCancel(ctx sdk.Context, marketId, orderType, orderId string) {
	store := k.getPendingCancelStore(ctx)
	key := types.PendingCancelKey(marketId, orderType, orderId)
	store.Delete(key)
}

func (k Keeper) GetQueueMessagesByMarket(ctx sdk.Context, marketId string) (list []types.QueueMessage) {
	store := k.getQueueMessageStore(ctx)

	// Create prefix for this market
	storePrefix := types.QueueMessageMarketPrefix(marketId)
	iterator := storetypes.KVStorePrefixIterator(store, storePrefix)
	defer iterator.Close()

	// Iterate through messages in order (by zero-filled message ID within this market)
	for ; iterator.Valid(); iterator.Next() {
		var msg types.QueueMessage
		k.cdc.MustUnmarshal(iterator.Value(), &msg)
		list = append(list, msg)
	}

	return
}
