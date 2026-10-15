package v830

import (
	"context"

	upgradetypes "cosmossdk.io/x/upgrade/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
)

const UpgradeName = "v8.3.0"

func CreateUpgradeHandler(
	cfg module.Configurator,
	mm *module.Manager,
) upgradetypes.UpgradeHandler {

	return func(c context.Context, _plan upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		ctx := sdk.UnwrapSDKContext(c)

		// RunMigrations triggers module-level migrations based on ConsensusVersion
		// changes: the ibc-go v10 module migrations. The cosmos-sdk v0.50 -> v0.53
		// bump changes no module's version.
		return mm.RunMigrations(ctx, cfg, vm)
	}
}
