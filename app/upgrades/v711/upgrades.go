package v711

import (
	upgradetypes "cosmossdk.io/x/upgrade/types"
	"github.com/bze-alphateam/bze/app/upgrades"
	"github.com/cosmos/cosmos-sdk/types/module"
)

const UpgradeName = "v7.1.1"

func CreateUpgradeHandler(
	_ module.Configurator,
	_ *module.Manager,
) upgradetypes.UpgradeHandler {
	return upgrades.EmptyUpgradeHandler()
}
