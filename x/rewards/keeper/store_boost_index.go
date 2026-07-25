package keeper

import (
	"strings"

	storetypes "cosmossdk.io/store/types"
	"github.com/bze-alphateam/bze/x/rewards/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// SetBoostParticipantIndex records that address is a participant of rewardId in
// the reverse index used by the boost finalization sweep. Set semantics: the
// value is a presence marker.
func (k Keeper) SetBoostParticipantIndex(ctx sdk.Context, rewardId, address string) {
	store := k.getPrefixedStore(ctx, types.KeyPrefix(types.BoostParticipantIndexKeyPrefix))
	store.Set(types.BoostParticipantIndexKey(rewardId, address), types.BoostParticipantIndexValue)
}

// RemoveBoostParticipantIndex removes a participant's reverse-index entry for a reward.
func (k Keeper) RemoveBoostParticipantIndex(ctx sdk.Context, rewardId, address string) {
	store := k.getPrefixedStore(ctx, types.KeyPrefix(types.BoostParticipantIndexKeyPrefix))
	store.Delete(types.BoostParticipantIndexKey(rewardId, address))
}

// HasBoostParticipantIndex reports whether a participant's reverse-index entry exists.
func (k Keeper) HasBoostParticipantIndex(ctx sdk.Context, rewardId, address string) bool {
	store := k.getPrefixedStore(ctx, types.KeyPrefix(types.BoostParticipantIndexKeyPrefix))
	return store.Has(types.BoostParticipantIndexKey(rewardId, address))
}

// GetBoostParticipantsFromCursor returns up to limit participant addresses of a
// reward, in key order, starting strictly after fromCursor. An empty fromCursor
// starts from the beginning; a limit <= 0 returns all remaining participants.
// The returned slice is the paginated input the finalization sweep walks.
func (k Keeper) GetBoostParticipantsFromCursor(ctx sdk.Context, rewardId, fromCursor string, limit int) []string {
	store := k.getPrefixedStore(ctx, types.KeyPrefix(types.BoostParticipantIndexKeyPrefix))
	rewardPrefix := types.BoostParticipantIndexRewardPrefix(rewardId)

	var startKey []byte
	if fromCursor != "" {
		// Start right after the cursor's key by appending 0x00.
		startKey = append(types.BoostParticipantIndexKey(rewardId, fromCursor), 0x00)
	} else {
		startKey = rewardPrefix
	}
	endKey := storetypes.PrefixEndBytes(rewardPrefix)

	iterator := store.Iterator(startKey, endKey)
	defer iterator.Close()

	var addresses []string
	for ; iterator.Valid() && (limit <= 0 || len(addresses) < limit); iterator.Next() {
		// key layout within the store: {reward_id}/{address}/
		addr := strings.TrimSuffix(string(iterator.Key()[len(rewardPrefix):]), "/")
		addresses = append(addresses, addr)
	}

	return addresses
}
