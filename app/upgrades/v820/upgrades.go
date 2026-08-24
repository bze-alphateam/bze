package v820

import (
	"context"

	upgradetypes "cosmossdk.io/x/upgrade/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
)

// UpgradeName is the name validators use in the software-upgrade proposal for v8.2.0.
const UpgradeName = "v8.2.0"

func CreateUpgradeHandler(
	cfg module.Configurator,
	mm *module.Manager,
) upgradetypes.UpgradeHandler {

	return func(c context.Context, _plan upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		ctx := sdk.UnwrapSDKContext(c)

		// RunMigrations triggers module-level migrations based on ConsensusVersion
		// changes: rewards v4->v5 sets the Denom Rewards parameter defaults. No store
		// keys are added or removed, so no store loader is needed.
		newVm, err := mm.RunMigrations(ctx, cfg, vm)
		if err != nil {
			return newVm, err
		}

		return newVm, nil
	}
}
