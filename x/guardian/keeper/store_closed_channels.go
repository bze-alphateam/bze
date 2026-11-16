package keeper

import (
	"context"

	"cosmossdk.io/store/prefix"
	"github.com/cosmos/cosmos-sdk/runtime"

	"github.com/bze-alphateam/bze/x/guardian/types"
)

// closedChannelMarker is the value stored for a closed channel: only the key matters.
var closedChannelMarker = []byte{1}

func (k Keeper) closedChannelStore(ctx context.Context) prefix.Store {
	return prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.ClosedChannelKeyPrefix)
}

// IsChannelClosed reports whether every ICS-20 transfer on the channel is stopped.
// The transfer stack calls it for every packet sent or received.
func (k Keeper) IsChannelClosed(ctx context.Context, channelID string) bool {
	return k.closedChannelStore(ctx).Has([]byte(channelID))
}

// SetClosedChannel marks the channel closed. It checks nothing: the messages and
// the genesis validation do.
func (k Keeper) SetClosedChannel(ctx context.Context, channelID string) {
	k.closedChannelStore(ctx).Set([]byte(channelID), closedChannelMarker)
}

// RemoveClosedChannel lets transfers through the channel again.
func (k Keeper) RemoveClosedChannel(ctx context.Context, channelID string) {
	k.closedChannelStore(ctx).Delete([]byte(channelID))
}

// GetAllClosedChannels returns the closed channel ids in store order, nil when none.
func (k Keeper) GetAllClosedChannels(ctx context.Context) []string {
	iterator := k.closedChannelStore(ctx).Iterator(nil, nil)
	defer iterator.Close()

	var channels []string
	for ; iterator.Valid(); iterator.Next() {
		channels = append(channels, string(iterator.Key()))
	}

	return channels
}
