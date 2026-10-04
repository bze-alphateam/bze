package keeper

import (
	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/bze-alphateam/bze/x/tradebin/types"
)

// haltedDenomValue is the value stored under a halted denom's key; only the key's presence matters.
var haltedDenomValue = []byte{1}

func (k Keeper) getHaltedDenomStore(ctx sdk.Context) prefix.Store {
	return k.getPrefixedStore(ctx, types.HaltedDenomPrefix())
}

// SetHaltedDenom marks denom as halted. Idempotent.
func (k Keeper) SetHaltedDenom(ctx sdk.Context, denom string) {
	k.getHaltedDenomStore(ctx).Set(types.HaltedDenomKey(denom), haltedDenomValue)
}

// RemoveHaltedDenom lifts the halt on denom. A denom that is not halted is a no-op.
func (k Keeper) RemoveHaltedDenom(ctx sdk.Context, denom string) {
	k.getHaltedDenomStore(ctx).Delete(types.HaltedDenomKey(denom))
}

// IsDenomHalted reports whether governance halted denom. A single store Has() on the denom's own key:
// exact match (a market id "base/quote" or a pool id "base_quote" never matches) at a flat gas cost
// that does not depend on how many denoms are halted.
func (k Keeper) IsDenomHalted(ctx sdk.Context, denom string) bool {
	return k.getHaltedDenomStore(ctx).Has(types.HaltedDenomKey(denom))
}

// GetAllHaltedDenoms returns every halted denom in store (byte) order. Used by genesis export; the
// paginated HaltedDenoms query iterates the store directly.
func (k Keeper) GetAllHaltedDenoms(ctx sdk.Context) (list []string) {
	store := k.getHaltedDenomStore(ctx)
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		list = append(list, string(iterator.Key()))
	}

	return list
}
