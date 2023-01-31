package v600

import (
	cointrunkkeeper "github.com/bze-alphateam/bze/x/cointrunk/keeper"
	cointrunktypes "github.com/bze-alphateam/bze/x/cointrunk/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"
)

const UpgradeName = "v6.0.0"

func CreateUpgradeHandler(k *cointrunkkeeper.Keeper) upgradetypes.UpgradeHandler {
	return func(ctx sdk.Context, _plan upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		//set cointrunk module default params
		k.SetParams(ctx, cointrunktypes.DefaultParams())
		return vm, nil
	}
}
