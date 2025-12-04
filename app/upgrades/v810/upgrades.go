package v810

import (
	"context"

	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
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
) upgradetypes.UpgradeHandler {

	return func(c context.Context, _plan upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		ctx := sdk.UnwrapSDKContext(c)

		// Migrate tradebin module parameters
		migrateTradebinParams(ctx, paramsKeeper)

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

	// Save updated parameters
	tradebinSubspace.SetParamSet(ctx, &params)

	ctx.Logger().Info("tradebin module parameters migrated successfully",
		"orderBookExtraGasWindow", params.OrderBookExtraGasWindow,
		"orderBookQueueExtraGas", params.OrderBookQueueExtraGas,
		"fillOrdersExtraGas", params.FillOrdersExtraGas,
	)
}

func GetStoreUpgrades() *storetypes.StoreUpgrades {
	return &storetypes.StoreUpgrades{
		Added:   []string{},
		Deleted: []string{},
	}
}
