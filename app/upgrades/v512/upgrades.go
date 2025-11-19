package v512

import (
	upgradetypes "cosmossdk.io/x/upgrade/types"
	"github.com/bze-alphateam/bze/app/upgrades"
)

const UpgradeName = "v5.1.2"

func CreateUpgradeHandler() upgradetypes.UpgradeHandler {
	return upgrades.EmptyUpgradeHandler()
}
