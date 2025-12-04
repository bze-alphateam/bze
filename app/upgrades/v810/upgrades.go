package v810

import (
	"context"

	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	tradebinkeeper "github.com/bze-alphateam/bze/x/tradebin/keeper"
	tradebintypes "github.com/bze-alphateam/bze/x/tradebin/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	paramskeeper "github.com/cosmos/cosmos-sdk/x/params/keeper"
)

const UpgradeName = "v8.1.0"

func CreateUpgradeHandler(
	cfg module.Configurator,
	mm *module.Manager,
	paramsKeeper *paramskeeper.Keeper,
	tradebinKeeper *tradebinkeeper.Keeper,
) upgradetypes.UpgradeHandler {

	return func(c context.Context, _plan upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		ctx := sdk.UnwrapSDKContext(c)

		// Migrate tradebin module parameters
		migrateTradebinParams(ctx, paramsKeeper)

		// Create initial snapshots for all existing liquidity pools
		snapshotExistingLiquidityPools(ctx, tradebinKeeper)

		newVm, err := mm.RunMigrations(ctx, cfg, vm)
		if err != nil {
			return newVm, err
		}

		return newVm, nil
	}
}

func migrateTradebinParams(ctx sdk.Context, paramsKeeper *paramskeeper.Keeper) {
	tradebinSubspace, found := paramsKeeper.GetSubspace(tradebintypes.ModuleName)
	if !found {
		ctx.Logger().Error("tradebin subspace not found during parameter migration")
		return
	}

	// Get existing parameters
	var params tradebintypes.Params
	tradebinSubspace.GetParamSet(ctx, &params)

	// Set new parameters with default values
	params.OrderBookExtraGasWindow = tradebintypes.DefaultOrderBookExtraGasWindow
	params.OrderBookQueueExtraGas = tradebintypes.DefaultOrderBookQueueExtraGas
	params.FillOrdersExtraGas = tradebintypes.DefaultFillOrdersExtraGas
	params.MinNativeLiquidityForModuleSwap = tradebintypes.DefaultMinNativeLiquidityForModuleSwap

	// Save updated parameters
	tradebinSubspace.SetParamSet(ctx, &params)

	ctx.Logger().Info("tradebin module parameters migrated successfully",
		"orderBookExtraGasWindow", params.OrderBookExtraGasWindow,
		"orderBookQueueExtraGas", params.OrderBookQueueExtraGas,
		"fillOrdersExtraGas", params.FillOrdersExtraGas,
		"minNativeLiquidityForModuleSwap", params.MinNativeLiquidityForModuleSwap,
	)
}

func snapshotExistingLiquidityPools(ctx sdk.Context, tradebinKeeper *tradebinkeeper.Keeper) {
	// Get all existing liquidity pools
	allPools := tradebinKeeper.GetAllLiquidityPool(ctx)

	ctx.Logger().Info("starting liquidity pool snapshots migration",
		"totalPools", len(allPools),
	)

	// Create snapshots for all pools
	snapshotCount := 0
	for _, pool := range allPools {
		tradebinKeeper.SetLiquidityPoolSnapshot(ctx, pool)
		snapshotCount++

		ctx.Logger().Info("liquidity pool snapshot created",
			"poolId", pool.Id,
			"base", pool.Base,
			"quote", pool.Quote,
			"reserveBase", pool.ReserveBase.String(),
			"reserveQuote", pool.ReserveQuote.String(),
		)
	}

	ctx.Logger().Info("liquidity pool snapshots migration completed",
		"totalPools", len(allPools),
		"snapshotsCreated", snapshotCount,
	)
}

func GetStoreUpgrades() *storetypes.StoreUpgrades {
	return &storetypes.StoreUpgrades{
		Added:   []string{},
		Deleted: []string{},
	}
}
