package keeper

import (
	"cosmossdk.io/math"
	"github.com/bze-alphateam/bze/x/tradebin/types"
	v2types "github.com/bze-alphateam/bze/x/tradebin/v2types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// poolPriceFullDecayFactor is how many half-lives without an update make the old average negligible
// (0.5^20 is about 1e-6): past it the average is simply the start-of-block pool price.
const poolPriceFullDecayFactor = 20

// poolPrice returns quote reserve / base reserve. A pool with a zero reserve, or whose price rounds to zero
// at 18 decimals, has no price.
func poolPrice(pool types.LiquidityPool) (math.LegacyDec, bool) {
	if pool.ReserveBase.IsNil() || pool.ReserveQuote.IsNil() || !pool.ReserveBase.IsPositive() || !pool.ReserveQuote.IsPositive() {
		return math.LegacyDec{}, false
	}

	price := math.LegacyNewDecFromInt(pool.ReserveQuote).Quo(math.LegacyNewDecFromInt(pool.ReserveBase))
	if !price.IsPositive() {
		return math.LegacyDec{}, false
	}

	return price, true
}

// foldPoolPriceAverage brings the stored average of a pool up to the current block. The pool as it is stored
// (before any save of this block) gives the price that held since the last update.
//
// Returns false when the stored pool has no price, so there is nothing to fold with.
func (k Keeper) foldPoolPriceAverage(ctx sdk.Context, record types.PoolPriceAverage, params v2types.Params) (math.LegacyDec, bool) {
	if ctx.BlockHeight() <= record.LastHeight {
		return record.AvgPrice, true
	}

	stored, found := k.GetLiquidityPool(ctx, record.PoolId)
	if !found {
		return math.LegacyDec{}, false
	}

	price, ok := poolPrice(stored)
	if !ok {
		return math.LegacyDec{}, false
	}

	halfLife := params.PoolPriceHalfLifeBlocks
	if halfLife < 1 {
		halfLife = v2types.DefaultPoolPriceHalfLifeBlocks
	}
	if halfLife > v2types.MaxPoolPriceHalfLifeBlocks {
		halfLife = v2types.MaxPoolPriceHalfLifeBlocks
	}

	elapsed := uint64(ctx.BlockHeight() - record.LastHeight)
	if elapsed > poolPriceFullDecayFactor*halfLife {
		return price, true
	}

	// d = 0.5^(1/halfLife); the average moves toward the price by (1 - d^elapsed) of the gap
	d, err := math.LegacyNewDecWithPrec(5, 1).ApproxRoot(halfLife)
	if err != nil {
		return math.LegacyDec{}, false
	}

	return price.Add(record.AvgPrice.Sub(price).Mul(d.Power(elapsed))), true
}

// observePoolPrice updates the price average of a pool the first time the pool is saved in a block. It must be
// called before the pool is written.
func (k Keeper) observePoolPrice(ctx sdk.Context, pool types.LiquidityPool) {
	record, found := k.GetPoolPriceAverageRecord(ctx, pool.Id)
	if found && ctx.BlockHeight() <= record.LastHeight {
		return
	}

	if found {
		if avg, ok := k.foldPoolPriceAverage(ctx, record, k.GetParams(ctx)); ok {
			record.AvgPrice = avg
			record.LastHeight = ctx.BlockHeight()
			k.SetPoolPriceAverage(ctx, record)

			return
		}
	}

	// no record yet, or the stored pool has no price: start from the pool being saved
	price, ok := poolPrice(pool)
	if !ok {
		return
	}

	k.SetPoolPriceAverage(ctx, types.PoolPriceAverage{
		PoolId:     pool.Id,
		AvgPrice:   price,
		LastHeight: ctx.BlockHeight(),
	})
}

// GetPoolPriceAverage returns the moving average of a pool's price, brought up to the current block. It does
// not write. When the stored pool has a price, the next save of the pool stores the same value; when it has
// none, the next save restarts the average from the pool being saved while this view returns the stored record.
func (k Keeper) GetPoolPriceAverage(ctx sdk.Context, poolId string) (avg math.LegacyDec, found bool) {
	record, found := k.GetPoolPriceAverageRecord(ctx, poolId)
	if !found {
		return math.LegacyDec{}, false
	}

	if folded, ok := k.foldPoolPriceAverage(ctx, record, k.GetParams(ctx)); ok {
		return folded, true
	}

	return record.AvgPrice, true
}

// SeedPoolPriceAverage stores the current price of a pool as its average, as of the current block. Pools
// without a price are skipped.
func (k Keeper) SeedPoolPriceAverage(ctx sdk.Context, pool types.LiquidityPool) {
	price, ok := poolPrice(pool)
	if !ok {
		return
	}

	k.SetPoolPriceAverage(ctx, types.PoolPriceAverage{PoolId: pool.Id, AvgPrice: price, LastHeight: ctx.BlockHeight()})
}
