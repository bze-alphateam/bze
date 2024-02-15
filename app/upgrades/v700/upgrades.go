package v700

import (
	tokenfactorykeeper "github.com/bze-alphateam/bze/x/tokenfactory/keeper"
	tokenfactorytypes "github.com/bze-alphateam/bze/x/tokenfactory/types"
	tradebinkeeper "github.com/bze-alphateam/bze/x/tradebin/keeper"
	tradebintypes "github.com/bze-alphateam/bze/x/tradebin/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"
)

const UpgradeName = "v7.0.0"

func CreateUpgradeHandler(factoryKeeper *tokenfactorykeeper.Keeper, tbinKeeper *tradebinkeeper.Keeper) upgradetypes.UpgradeHandler {
	return func(ctx sdk.Context, _plan upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		//set tokenfactory module default params
		factoryKeeper.SetParams(ctx, tokenfactorytypes.DefaultParams())
		tbinKeeper.SetParams(ctx, tradebintypes.DefaultParams())
		return vm, nil
	}
}
