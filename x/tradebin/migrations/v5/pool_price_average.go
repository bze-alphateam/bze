package v5

import (
	"github.com/bze-alphateam/bze/x/tradebin/types"
	v2types "github.com/bze-alphateam/bze/x/tradebin/v2types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// PoolPriceAverageKeeper is the state MigratePoolPriceAverage reads and writes.
type PoolPriceAverageKeeper interface {
	ParamsKeeper
	GetAllLiquidityPool(ctx sdk.Context) []types.LiquidityPool
	SeedPoolPriceAverage(ctx sdk.Context, pool types.LiquidityPool)
}

// MigratePoolPriceAverage sets the pool price average params (half life, fee and module drift limits) to
// their defaults and seeds the price average of every existing pool with its current price at the upgrade
// height. The pools themselves are not rewritten.
func MigratePoolPriceAverage(ctx sdk.Context, k PoolPriceAverageKeeper) error {
	params := k.GetParams(ctx)
	params.PoolPriceHalfLifeBlocks = v2types.DefaultPoolPriceHalfLifeBlocks
	params.FeeMaxPriceDrift = v2types.DefaultFeeMaxPriceDrift
	params.ModuleMaxPriceDrift = v2types.DefaultModuleMaxPriceDrift
	if err := k.SetParams(ctx, params); err != nil {
		return err
	}

	for _, pool := range k.GetAllLiquidityPool(ctx) {
		k.SeedPoolPriceAverage(ctx, pool)
	}

	return nil
}
