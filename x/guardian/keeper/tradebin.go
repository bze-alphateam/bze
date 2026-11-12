package keeper

import (
	"github.com/bze-alphateam/bze/x/guardian/types"
)

// SetTradebinKeeper wires the tradebin keeper, used to halt denoms on the
// DEX. It is set by the app after the depinject build, like the rate-limiting
// keeper, so the guardian keeper's constructor does not change.
func (k *Keeper) SetTradebinKeeper(tradebinKeeper types.TradebinKeeper) {
	k.tradebinKeeper = tradebinKeeper
}
