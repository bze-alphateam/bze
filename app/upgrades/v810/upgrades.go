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

		// Migrate order keys to new precision format
		migrateOrderKeys(ctx, tradebinKeeper)

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

func migrateOrderKeys(ctx sdk.Context, tradebinKeeper *tradebinkeeper.Keeper) {
	ctx.Logger().Info("starting order key migration to new precision format")

	// Migrate PriceOrder keys
	migratePriceOrders(ctx, tradebinKeeper)

	// Migrate AggregatedOrder keys
	migrateAggregatedOrders(ctx, tradebinKeeper)

	ctx.Logger().Info("order key migration completed")
}

func migratePriceOrders(ctx sdk.Context, k *tradebinkeeper.Keeper) {
	// Get all price order references
	allPriceOrders := k.GetAllPriceOrder(ctx)

	ctx.Logger().Info("starting price order migration",
		"totalOrders", len(allPriceOrders),
	)

	// Get the price order store directly
	store := k.GetPriceOrderStoreForMigration(ctx)

	migratedCount := 0
	for _, orderRef := range allPriceOrders {
		// Get the actual order to retrieve the price
		order, found := k.GetOrder(ctx, orderRef.MarketId, orderRef.OrderType, orderRef.Id)
		if !found {
			ctx.Logger().Error("order not found during migration",
				"marketId", orderRef.MarketId,
				"orderType", orderRef.OrderType,
				"orderId", orderRef.Id,
			)
			continue
		}

		// Delete using old key format (24 chars, 10 decimals)
		oldKey := oldPriceOrderKey(order.MarketId, order.OrderType, order.Price, order.Id)
		store.Delete(oldKey)

		// Set using new key format (32 chars, 18 decimals)
		k.SetPriceOrder(ctx, orderRef, order.Price)

		migratedCount++

		if migratedCount%100 == 0 {
			ctx.Logger().Info("price order migration progress",
				"migrated", migratedCount,
				"total", len(allPriceOrders),
			)
		}
	}

	ctx.Logger().Info("price order migration completed",
		"totalMigrated", migratedCount,
	)
}

func migrateAggregatedOrders(ctx sdk.Context, k *tradebinkeeper.Keeper) {
	// Get all aggregated orders using the current store iteration
	// These will have the old key format
	allAggOrders := k.GetAllAggregatedOrder(ctx)

	ctx.Logger().Info("starting aggregated order migration",
		"totalOrders", len(allAggOrders),
	)

	// Get the aggregated order store directly
	store := k.GetAggregatedOrderStoreForMigration(ctx)

	migratedCount := 0
	for _, aggOrder := range allAggOrders {
		// Delete using old key format (24 chars, 10 decimals)
		oldKey := oldAggOrderKey(aggOrder.MarketId, aggOrder.OrderType, aggOrder.Price)
		store.Delete(oldKey)

		// Set using new key format (32 chars, 18 decimals)
		k.SetAggregatedOrder(ctx, aggOrder)

		migratedCount++

		if migratedCount%100 == 0 {
			ctx.Logger().Info("aggregated order migration progress",
				"migrated", migratedCount,
				"total", len(allAggOrders),
			)
		}
	}

	ctx.Logger().Info("aggregated order migration completed",
		"totalMigrated", migratedCount,
	)
}

func GetStoreUpgrades() *storetypes.StoreUpgrades {
	return &storetypes.StoreUpgrades{
		Added:   []string{},
		Deleted: []string{},
	}
}
