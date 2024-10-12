package v711

import (
	"github.com/bze-alphateam/bze/app/upgrades"
	"github.com/cosmos/cosmos-sdk/types/module"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"
)

const UpgradeName = "v7.1.1"

func CreateUpgradeHandler(
	_ module.Configurator,
	_ *module.Manager,
) upgradetypes.UpgradeHandler {
	return upgrades.EmptyUpgradeHandler()
}
