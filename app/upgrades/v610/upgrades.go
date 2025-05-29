package v610

import (
	upgradetypes "cosmossdk.io/x/upgrade/types"
	"github.com/bze-alphateam/bze/app/upgrades"
)

const UpgradeName = "v6.1.0"

func CreateUpgradeHandler() upgradetypes.UpgradeHandler {
	return upgrades.EmptyUpgradeHandler()
}
