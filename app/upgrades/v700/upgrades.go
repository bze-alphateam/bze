package v700

import (
	burnermodule "github.com/bze-alphateam/bze/x/burner"
	burnermoduletypes "github.com/bze-alphateam/bze/x/burner/types"
	cointrunkmodule "github.com/bze-alphateam/bze/x/cointrunk"
	cointrunkmoduletypes "github.com/bze-alphateam/bze/x/cointrunk/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"
)

const UpgradeName = "v7.0.0"

func CreateUpgradeHandler(
	cfg module.Configurator,
	mm *module.Manager,
) upgradetypes.UpgradeHandler {

	return func(ctx sdk.Context, _plan upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		//add older modules not present in module_versions yet
		vm[burnermoduletypes.ModuleName] = burnermodule.AppModule{}.ConsensusVersion()
		vm[cointrunkmoduletypes.ModuleName] = cointrunkmodule.AppModule{}.ConsensusVersion()

		//run default migrations in order to init new module's genesis and to have them in vm
		return mm.RunMigrations(ctx, cfg, vm)
	}
}
