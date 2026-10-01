package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/bze-alphateam/bze/x/tradebin/types"
)

// IsDenomHalted reports whether governance listed denom in the tradebin halted_denoms param.
// Exact string comparison against the list — never a substring match on market or pool ids.
func (k Keeper) IsDenomHalted(ctx sdk.Context, denom string) bool {
	return k.GetParams(ctx).IsDenomHalted(denom)
}

// isMarketHalted reports whether the market's base or quote denom is halted.
func (k Keeper) isMarketHalted(ctx sdk.Context, market *types.Market) bool {
	params := k.GetParams(ctx)

	return params.IsDenomHalted(market.GetBase()) || params.IsDenomHalted(market.GetQuote())
}

// isPoolHalted reports whether the pool's base or quote denom is halted. This is the check behind
// the no-swap invariant: a halted denom is never exchanged against anything, by anyone.
func (k Keeper) isPoolHalted(ctx sdk.Context, pool *types.LiquidityPool) bool {
	for _, denom := range k.GetParams(ctx).HaltedDenoms {
		if pool.HasDenom(denom) {
			return true
		}
	}

	return false
}
