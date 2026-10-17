package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/bze-alphateam/bze/x/dao/types"
)

// GetSourcePins returns how many DAOs (and later open proposals) use the given
// electorate source (types.StakingRewardSource / types.DenomRewardSource).
func (k Keeper) GetSourcePins(ctx context.Context, source []byte) uint64 {
	bz := k.getPrefixedStore(ctx, types.KeyPrefix(types.PinKeyPrefix)).Get(source)
	if bz == nil {
		return 0
	}

	return sdk.BigEndianToUint64(bz)
}

// pinSource adds one user of the source.
func (k Keeper) pinSource(ctx context.Context, source []byte) {
	store := k.getPrefixedStore(ctx, types.KeyPrefix(types.PinKeyPrefix))
	store.Set(source, sdk.Uint64ToBigEndian(k.GetSourcePins(ctx, source)+1))
}

// unpinSource removes one user of the source and deletes the key at zero.
func (k Keeper) unpinSource(ctx context.Context, source []byte) {
	store := k.getPrefixedStore(ctx, types.KeyPrefix(types.PinKeyPrefix))
	pins := k.GetSourcePins(ctx, source)
	if pins <= 1 {
		store.Delete(source)
		return
	}

	store.Set(source, sdk.Uint64ToBigEndian(pins-1))
}
