package v4

import (
	"fmt"

	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	"github.com/bze-alphateam/bze/x/tradebin/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// MigrateOrderKeys migrates order keys from the old 24-char/10-decimal precision format
// to the new 32-char/18-decimal format. It uses a write-before-delete pattern to prevent
// data loss during migration.
func MigrateOrderKeys(
	ctx sdk.Context,
	orderStore prefix.Store,
	priceOrderStore prefix.Store,
	aggOrderStore prefix.Store,
	cdc codec.BinaryCodec,
) error {
	ctx.Logger().Info("starting order key migration to new precision format")

	if err := migratePriceOrders(ctx, orderStore, priceOrderStore, cdc); err != nil {
		return err
	}

	if err := migrateAggregatedOrders(ctx, aggOrderStore, cdc); err != nil {
		return err
	}

	ctx.Logger().Info("order key migration completed")
	return nil
}

func migratePriceOrders(ctx sdk.Context, orderStore, priceOrderStore prefix.Store, cdc codec.BinaryCodec) error {
	// Collect all price order references
	var allPriceOrders []types.OrderReference
	iterator := storetypes.KVStorePrefixIterator(priceOrderStore, []byte{})
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.OrderReference
		cdc.MustUnmarshal(iterator.Value(), &val)
		allPriceOrders = append(allPriceOrders, val)
	}

	priceOrdersCount := len(allPriceOrders)
	ctx.Logger().Info("starting price order migration",
		"totalOrders", priceOrdersCount,
	)

	for i, orderRef := range allPriceOrders {
		// Look up the full order to get the price
		orderKey := types.OrderKey(orderRef.MarketId, orderRef.OrderType, orderRef.Id)
		b := orderStore.Get(orderKey)
		if b == nil {
			ctx.Logger().Error("order not found during migration - data corruption detected",
				"marketId", orderRef.MarketId,
				"orderType", orderRef.OrderType,
				"orderId", orderRef.Id,
			)
			return fmt.Errorf("orphaned order reference found: marketId=%s orderType=%s orderId=%s",
				orderRef.MarketId, orderRef.OrderType, orderRef.Id)
		}

		var order types.Order
		cdc.MustUnmarshal(b, &order)

		// First, write using new key format (32 chars, 18 decimals)
		newKey := types.PriceOrderKey(order.MarketId, order.OrderType, order.Price, order.Id)
		bz := cdc.MustMarshal(&orderRef)
		priceOrderStore.Set(newKey, bz)

		// Only after successful write, delete the old key format (24 chars, 10 decimals)
		oldKey := oldPriceOrderKey(order.MarketId, order.OrderType, order.Price, order.Id)
		priceOrderStore.Delete(oldKey)

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

func migrateAggregatedOrders(ctx sdk.Context, aggOrderStore prefix.Store, cdc codec.BinaryCodec) error {
	// Collect all aggregated orders
	var allAggOrders []types.AggregatedOrder
	iterator := storetypes.KVStorePrefixIterator(aggOrderStore, []byte{})
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.AggregatedOrder
		cdc.MustUnmarshal(iterator.Value(), &val)
		allAggOrders = append(allAggOrders, val)
	}

	allAggOrdersCount := len(allAggOrders)
	ctx.Logger().Info("starting aggregated order migration",
		"totalOrders", allAggOrdersCount,
	)

	for i, aggOrder := range allAggOrders {
		// First, write using new key format (32 chars, 18 decimals)
		newKey := types.AggOrderKey(aggOrder.MarketId, aggOrder.OrderType, aggOrder.Price)
		bz := cdc.MustMarshal(&aggOrder)
		aggOrderStore.Set(newKey, bz)

		// Only after successful write, delete the old key format (24 chars, 10 decimals)
		oldKey := oldAggOrderKey(aggOrder.MarketId, aggOrder.OrderType, aggOrder.Price)
		aggOrderStore.Delete(oldKey)

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
