package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/bze-alphateam/bze/x/dao/types"
)

// GetSourcePins returns how many DAOs and proposals in voting use the given
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

// GetAllPins returns the pin count of every pinned source, by source key.
func (k Keeper) GetAllPins(ctx context.Context) []types.PinCount {
	iterator := k.getPrefixedStore(ctx, types.KeyPrefix(types.PinKeyPrefix)).Iterator(nil, nil)
	defer iterator.Close()

	var pins []types.PinCount
	for ; iterator.Valid(); iterator.Next() {
		pins = append(pins, types.PinCount{Source: string(iterator.Key()), Count: sdk.BigEndianToUint64(iterator.Value())})
	}

	return pins
}

// SetAllPins imports the pin counts (genesis) as they are. They must pass
// GenesisState.Validate, which checks them against the DAOs and the proposals
// in voting (I4).
func (k Keeper) SetAllPins(ctx context.Context, pins []types.PinCount) {
	store := k.getPrefixedStore(ctx, types.KeyPrefix(types.PinKeyPrefix))
	for _, pin := range pins {
		store.Set([]byte(pin.Source), sdk.Uint64ToBigEndian(pin.Count))
	}
}
