package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/bze-alphateam/bze/x/tradebin/types"
)

// isMarketHalted reports whether the market's base or quote denom is halted.
func (k Keeper) isMarketHalted(ctx sdk.Context, market *types.Market) bool {
	return k.IsDenomHalted(ctx, market.GetBase()) || k.IsDenomHalted(ctx, market.GetQuote())
}

// isPoolHalted reports whether the pool's base or quote denom is halted. This is the check behind the
// no-swap invariant: a halted denom is never exchanged against anything, by anyone.
func (k Keeper) isPoolHalted(ctx sdk.Context, pool *types.LiquidityPool) bool {
	return k.IsDenomHalted(ctx, pool.GetBase()) || k.IsDenomHalted(ctx, pool.GetQuote())
}
