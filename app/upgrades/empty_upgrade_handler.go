package upgrades

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"
)

// EmptyUpgradeHandler - should be used to create upgrade handler for gov software upgrades that require no migration
func EmptyUpgradeHandler() upgradetypes.UpgradeHandler {
	return func(ctx sdk.Context, _plan upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		//do nothing on purpose
		return vm, nil
	}
}
