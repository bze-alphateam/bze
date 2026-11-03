package keeper

import (
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/bze-alphateam/bze/x/txfeecollector/types"
)

// getCollectorCursor returns the last denom the module account's collector pass looked at, "" when the next pass
// starts from the first denom.
func (k Keeper) getCollectorCursor(ctx sdk.Context, module string) string {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))

	return string(store.Get(types.CollectorCursorKey(module)))
}

// setCollectorCursor stores the module account's collector cursor; "" deletes it.
func (k Keeper) setCollectorCursor(ctx sdk.Context, module, denom string) {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	if denom == "" {
		store.Delete(types.CollectorCursorKey(module))

		return
	}

	store.Set(types.CollectorCursorKey(module), []byte(denom))
}

// readCollectorPage returns the balances the module account's collector pass looks at in this block: at most
// MaxBalanceIterations of them, the ones after the module's cursor, in the bank store's denom order. Every coin of the
// page counts towards the bound, whatever the pass does with it.
//
// It moves the cursor before the pass handles the page; the pass runs in a cached context (EndBlock runs it through
// bzeutils.ApplyFuncIfNoError), so a failed pass discards the move and the next block reads the same page again.
// The cursor wraps: a page shorter than the bound reached the end of the balances, so the cursor is cleared and the
// next pass starts from the first denom; a cursor past the last denom (its balance was spent) gives an empty page, read
// again from the first denom in the same pass. No denom is looked at twice in one pass.
func (k Keeper) readCollectorPage(ctx sdk.Context, module string, moduleAddr sdk.AccAddress) sdk.Coins {
	limit := int(k.GetParams(ctx).MaxBalanceIterations)
	cursor := k.getCollectorCursor(ctx, module)

	page, last := k.bankKeeper.GetBalancesPage(ctx, moduleAddr, cursor, limit)
	if len(page) == 0 && cursor != "" {
		page, last = k.bankKeeper.GetBalancesPage(ctx, moduleAddr, "", limit)
	}

	if len(page) < limit {
		last = ""
	}
	k.setCollectorCursor(ctx, module, last)

	return page
}
