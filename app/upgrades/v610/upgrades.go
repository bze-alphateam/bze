package v610

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"
)

const UpgradeName = "v6.1.0"

func CreateUpgradeHandler() upgradetypes.UpgradeHandler {
	return func(ctx sdk.Context, _plan upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		// nothing to do
		return vm, nil
	}
}
