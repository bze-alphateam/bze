package keeper

import (
	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"
	"github.com/bze-alphateam/bze/x/tradebin/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// The Safe* functions in this file wrap three protocol operations that trade on a pool at its live price:
// CaptureAndSwapUserFee, ModuleSwapForNativeDenom and ModuleAddLiquidityWithNativeDenom. Each one refuses to trade
// when the pool price moved too far from the pool's moving average (see GetPoolPriceAverage), then calls the
// original function unchanged.
//
// CaptureAndTryToSwapUserFeesOrSendItAsIs (maker/taker fees) intentionally has no Safe* variant: those fees are
// tiny and not worth protecting, and checking them would hurt order placement.

// poolPriceDrifted reports whether the price of the pool is further from its moving average than limit
// (relative to the average). Exactly at the limit is not drifted. A pool without a price, or without an average
// record, counts as drifted.
func (k Keeper) poolPriceDrifted(ctx sdk.Context, pool types.LiquidityPool, limit math.LegacyDec) bool {
	spot, ok := poolPrice(pool)
	if !ok {
		return true
	}

	avg, found := k.GetPoolPriceAverage(ctx, pool.Id)
	if !found || avg.IsNil() || !avg.IsPositive() {
		return true
	}

	// |spot - avg| / avg > limit, written as a product so there is no division
	diff := spot.Sub(avg).Abs()

	return diff.GT(limit.Mul(avg))
}

// anyPoolPriceDrifted checks the pool between the native denom and each non-native coin. A coin without a pool
// is skipped: nothing is traded at a price that does not exist, and the original function reports it.
func (k Keeper) anyPoolPriceDrifted(ctx sdk.Context, coins sdk.Coins, limit math.LegacyDec) bool {
	nativeDenom := k.getNativeDenom(ctx)
	if nativeDenom == "" {
		return false
	}

	for _, coin := range coins {
		if coin.Denom == nativeDenom {
			continue
		}

		_, _, poolId := k.CreatePoolId(nativeDenom, coin.Denom)
		pool, found := k.GetLiquidityPool(ctx, poolId)
		if !found {
			continue
		}

		if k.poolPriceDrifted(ctx, pool, limit) {
			return true
		}
	}

	return false
}

// SafeCaptureAndSwapUserFee is CaptureAndSwapUserFee that does not swap the fee through a pool whose price
// drifted more than the fee_max_price_drift param from its moving average: the fee is then charged in the native
// denom (the payer fails with the original's error if there is not enough of it). The swap is safe because the
// payer's input is never computed from a pool price that is far from its average.
func (k Keeper) SafeCaptureAndSwapUserFee(ctx sdk.Context, payer sdk.AccAddress, fee sdk.Coins, toModule string) (sdk.Coins, error) {
	nativeDenom := k.getNativeDenom(ctx)
	ctxDenom := k.getPayerCtxDenom(ctx, payer)
	if ctxDenom == "" || ctxDenom == nativeDenom {
		return k.CaptureAndSwapUserFee(ctx, payer, fee, toModule)
	}

	_, _, poolId := k.CreatePoolId(nativeDenom, ctxDenom)
	pool, found := k.GetLiquidityPool(ctx, poolId)
	if !found {
		return k.CaptureAndSwapUserFee(ctx, payer, fee, toModule)
	}

	if k.poolPriceDrifted(ctx, pool, k.GetParams(ctx).FeeMaxPriceDrift) {
		k.Logger().Debug("pool price drifted from its average, charging fee in native denom", "pool", poolId)

		return k.CaptureAndSwapUserFee(ctx.WithValue(CtxFeeDenomKey, nativeDenom), payer, fee, toModule)
	}

	return k.CaptureAndSwapUserFee(ctx, payer, fee, toModule)
}

// SafeModuleSwapForNativeDenom is ModuleSwapForNativeDenom that returns ErrPoolPriceDrift, before any coin moves,
// when the pool of any coin drifted more than the module_max_price_drift param from its moving average. The swap
// is safe because the module never trades at a pool price that is far from its average.
func (k Keeper) SafeModuleSwapForNativeDenom(ctx sdk.Context, toModule string, coins sdk.Coins) (sdk.Coin, error) {
	if k.anyPoolPriceDrifted(ctx, coins, k.GetParams(ctx).ModuleMaxPriceDrift) {
		return sdk.Coin{}, errorsmod.Wrap(types.ErrPoolPriceDrift, "module swap refused")
	}

	return k.ModuleSwapForNativeDenom(ctx, toModule, coins)
}

// SafeModuleAddLiquidityWithNativeDenom is ModuleAddLiquidityWithNativeDenom that returns ErrPoolPriceDrift,
// before any coin moves, when the pool of any coin drifted more than the module_max_price_drift param from its
// moving average. Adding liquidity is safe because the module never trades at a pool price that is far from its
// average.
func (k Keeper) SafeModuleAddLiquidityWithNativeDenom(ctx sdk.Context, fromModule string, coins sdk.Coins) (sdk.Coins, sdk.Coins, error) {
	if k.anyPoolPriceDrifted(ctx, coins, k.GetParams(ctx).ModuleMaxPriceDrift) {
		return nil, nil, errorsmod.Wrap(types.ErrPoolPriceDrift, "module add liquidity refused")
	}

	return k.ModuleAddLiquidityWithNativeDenom(ctx, fromModule, coins)
}
