package keeper

import (
	"context"

	"github.com/bze-alphateam/bze/x/guardian/types"
)

// IsAutomationPaused reports whether the protocol automation is paused: while
// it is, the txfeecollector EndBlock conversions and the burner's periodic burn
// do nothing and every coin stays where it is.
func (k Keeper) IsAutomationPaused(ctx context.Context) bool {
	has, err := k.storeService.OpenKVStore(ctx).Has(types.AutomationPausedKey)
	if err != nil {
		panic(err)
	}

	return has
}

// SetAutomationPaused stores the pause flag. It does no authorization check,
// the messages do.
func (k Keeper) SetAutomationPaused(ctx context.Context, paused bool) {
	store := k.storeService.OpenKVStore(ctx)

	var err error
	if paused {
		err = store.Set(types.AutomationPausedKey, []byte{1})
	} else {
		err = store.Delete(types.AutomationPausedKey)
	}

	if err != nil {
		panic(err)
	}
}
