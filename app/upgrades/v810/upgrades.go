package v810

import (
	"context"
	"fmt"

	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	tradebinkeeper "github.com/bze-alphateam/bze/x/tradebin/keeper"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	crisistypes "github.com/cosmos/cosmos-sdk/x/crisis/types"
)

const UpgradeName = "v8.1.0"

func CreateUpgradeHandler(
	cfg module.Configurator,
	mm *module.Manager,
	tradebinKeeper *tradebinkeeper.Keeper,
) upgradetypes.UpgradeHandler {

	return func(c context.Context, _plan upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		ctx := sdk.UnwrapSDKContext(c)

		// Migrate order keys to new precision format
		if err := migrateOrderKeys(ctx, tradebinKeeper); err != nil {
			return nil, err
		}

		// RunMigrations will trigger module-level migrations (param migrations for
		// tradebin v3->v4, rewards v3->v4, txfeecollector v1->v2) based on
		// ConsensusVersion changes.
		newVm, err := mm.RunMigrations(ctx, cfg, vm)
		if err != nil {
			return newVm, err
		}

		return newVm, nil
	}
}

func migrateOrderKeys(ctx sdk.Context, tradebinKeeper *tradebinkeeper.Keeper) error {
	ctx.Logger().Info("starting order key migration to new precision format")

	// Migrate PriceOrder keys
	if err := migratePriceOrders(ctx, tradebinKeeper); err != nil {
		return err
	}

	// Migrate AggregatedOrder keys
	if err := migrateAggregatedOrders(ctx, tradebinKeeper); err != nil {
		return err
	}

	ctx.Logger().Info("order key migration completed")
	return nil
}

func migratePriceOrders(ctx sdk.Context, k *tradebinkeeper.Keeper) error {
	// Get all price order references
	allPriceOrders := k.GetAllPriceOrder(ctx)
	priceOrdersCount := len(allPriceOrders)
	ctx.Logger().Info("starting price order migration",
		"totalOrders", priceOrdersCount,
	)

	// Get the price order store directly
	store := k.GetPriceOrderStoreForMigration(ctx)

	for i, orderRef := range allPriceOrders {
		// Get the actual order to retrieve the price
		order, found := k.GetOrder(ctx, orderRef.MarketId, orderRef.OrderType, orderRef.Id)
		if !found {
			// Orphaned order reference - this indicates data corruption
			ctx.Logger().Error("order not found during migration - data corruption detected",
				"marketId", orderRef.MarketId,
				"orderType", orderRef.OrderType,
				"orderId", orderRef.Id,
			)
			return fmt.Errorf("orphaned order reference found: marketId=%s orderType=%s orderId=%s",
				orderRef.MarketId, orderRef.OrderType, orderRef.Id)
		}

		// First, write using new key format (32 chars, 18 decimals)
		// This ensures we don't lose data if the write fails
		k.SetPriceOrder(ctx, orderRef, order.Price)

		// Only after successful write, delete the old key format (24 chars, 10 decimals)
		oldKey := oldPriceOrderKey(order.MarketId, order.OrderType, order.Price, order.Id)
		store.Delete(oldKey)

		if i%100 == 0 {
			ctx.Logger().Info("price order migration progress",
				"item_index", i,
				"total", priceOrdersCount,
			)
		}
	}

	ctx.Logger().Info("price order migration completed",
		"totalMigrated", priceOrdersCount,
	)

	return nil
}

func migrateAggregatedOrders(ctx sdk.Context, k *tradebinkeeper.Keeper) error {
	// Get all aggregated orders using the current store iteration
	// These will have the old key format
	allAggOrders := k.GetAllAggregatedOrder(ctx)
	allAggOrdersCount := len(allAggOrders)

	ctx.Logger().Info("starting aggregated order migration",
		"totalOrders", len(allAggOrders),
	)

	// Get the aggregated order store directly
	store := k.GetAggregatedOrderStoreForMigration(ctx)

	for i, aggOrder := range allAggOrders {
		// First, write using new key format (32 chars, 18 decimals)
		// This ensures we don't lose data if the write fails
		k.SetAggregatedOrder(ctx, aggOrder)

		// Only after successful write, delete the old key format (24 chars, 10 decimals)
		oldKey := oldAggOrderKey(aggOrder.MarketId, aggOrder.OrderType, aggOrder.Price)
		store.Delete(oldKey)

		if i%100 == 0 {
			ctx.Logger().Info("aggregated order migration progress",
				"item_index", i,
				"total", allAggOrdersCount,
			)
		}
	}

	ctx.Logger().Info("aggregated order migration completed",
		"totalMigrated", allAggOrdersCount,
	)

	return nil
}

func GetStoreUpgrades() *storetypes.StoreUpgrades {
	return &storetypes.StoreUpgrades{
		Added:   []string{},
		Deleted: []string{crisistypes.ModuleName},
	}
}
