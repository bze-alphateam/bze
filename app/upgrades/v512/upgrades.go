package v512

import (
	"github.com/bze-alphateam/bze/app/upgrades"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"
)

const UpgradeName = "v5.1.2"

func CreateUpgradeHandler() upgradetypes.UpgradeHandler {
	return upgrades.EmptyUpgradeHandler()
}
