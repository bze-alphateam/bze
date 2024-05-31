package v700

import (
	tokenfactorykeeper "github.com/bze-alphateam/bze/x/tokenfactory/keeper"
	tokenfactorytypes "github.com/bze-alphateam/bze/x/tokenfactory/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"
)

const UpgradeName = "v7.0.0"

func CreateUpgradeHandler(k *tokenfactorykeeper.Keeper) upgradetypes.UpgradeHandler {
	return func(ctx sdk.Context, _plan upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		//set tokenfactory module default params
		k.SetParams(ctx, tokenfactorytypes.DefaultParams())
		return vm, nil
	}
}
