package v810

import (
	"context"
	"fmt"

	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	rewardskeeper "github.com/bze-alphateam/bze/x/rewards/keeper"
	rewardstypes "github.com/bze-alphateam/bze/x/rewards/types"
	tradebinkeeper "github.com/bze-alphateam/bze/x/tradebin/keeper"
	tradebintypes "github.com/bze-alphateam/bze/x/tradebin/types"
	txfeecollectorkeeper "github.com/bze-alphateam/bze/x/txfeecollector/keeper"
	txfeecollectortypes "github.com/bze-alphateam/bze/x/txfeecollector/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
)

const UpgradeName = "v8.1.0"

func CreateUpgradeHandler(
	cfg module.Configurator,
	mm *module.Manager,
	tradebinKeeper *tradebinkeeper.Keeper,
	txfeecollectorKeeper *txfeecollectorkeeper.Keeper,
	rewardsKeeper *rewardskeeper.Keeper,
) upgradetypes.UpgradeHandler {

	return func(c context.Context, _plan upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		ctx := sdk.UnwrapSDKContext(c)

		// Migrate tradebin module parameters
		if err := migrateTradebinParams(ctx, tradebinKeeper); err != nil {
			return nil, err
		}

		// Migrate txfeecollector module parameters
		if err := migrateTxFeeCollectorParams(ctx, txfeecollectorKeeper); err != nil {
			return nil, err
		}

		// Migrate rewards module parameters
		if err := migrateRewardsParams(ctx, rewardsKeeper); err != nil {
			return nil, err
		}

		// Migrate order keys to new precision format
		if err := migrateOrderKeys(ctx, tradebinKeeper); err != nil {
			return nil, err
		}

		newVm, err := mm.RunMigrations(ctx, cfg, vm)
		if err != nil {
			return newVm, err
		}

		return newVm, nil
	}
}

func migrateTradebinParams(ctx sdk.Context, tradebinKeeper *tradebinkeeper.Keeper) error {
	// Get existing parameters
	params := tradebinKeeper.GetParams(ctx)

	// Set new parameters with default values
	params.OrderBookExtraGasWindow = tradebintypes.DefaultOrderBookExtraGasWindow
	params.OrderBookQueueExtraGas = tradebintypes.DefaultOrderBookQueueExtraGas
	params.FillOrdersExtraGas = tradebintypes.DefaultFillOrdersExtraGas
	params.MinNativeLiquidityForModuleSwap = tradebintypes.DefaultMinNativeLiquidityForModuleSwap
	params.OrderBookPerBlockMessages = tradebintypes.DefaultOrderBookPerBlockMessages
	params.OrderBookQueueMessageScanExtraGas = tradebintypes.DefaultOrderBookQueueMessageScanExtraGas

	// Save updated parameters
	if err := tradebinKeeper.SetParams(ctx, params); err != nil {
		ctx.Logger().Error("failed to migrate tradebin module parameters", "error", err)
		return err
	}

	ctx.Logger().Info("tradebin module parameters migrated successfully",
		"orderBookExtraGasWindow", params.OrderBookExtraGasWindow,
		"orderBookQueueExtraGas", params.OrderBookQueueExtraGas,
		"fillOrdersExtraGas", params.FillOrdersExtraGas,
		"minNativeLiquidityForModuleSwap", params.MinNativeLiquidityForModuleSwap,
		"orderBookPerBlockMessages", params.OrderBookPerBlockMessages,
		"orderBookQueueMessageScanExtraGas", params.OrderBookQueueMessageScanExtraGas,
	)

	return nil
}

func migrateTxFeeCollectorParams(ctx sdk.Context, txfeecollectorKeeper *txfeecollectorkeeper.Keeper) error {
	// Set default parameters (module is new or has empty params before this upgrade)
	defaultParams := txfeecollectortypes.DefaultParams()

	// Save parameters with default values
	if err := txfeecollectorKeeper.SetParams(ctx, defaultParams); err != nil {
		ctx.Logger().Error("failed to migrate txfeecollector module parameters", "error", err)
		return err
	}

	ctx.Logger().Info("txfeecollector module parameters migrated successfully",
		"validatorMinGasFee", defaultParams.ValidatorMinGasFee.String(),
	)

	return nil
}

func migrateRewardsParams(ctx sdk.Context, rewardsKeeper *rewardskeeper.Keeper) error {
	params := rewardsKeeper.GetParams(ctx)

	params.ExtraGasForExitStake = rewardstypes.DefaultExtraGasForExitStake

	if err := rewardsKeeper.SetParams(ctx, params); err != nil {
		ctx.Logger().Error("failed to migrate rewards module parameters", "error", err)
		return err
	}

	ctx.Logger().Info("rewards module parameters migrated successfully",
		"extraGasForExitStake", params.ExtraGasForExitStake,
	)

	return nil
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
		Deleted: []string{},
	}
}
