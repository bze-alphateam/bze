package v610

import (
	"github.com/bze-alphateam/bze/app/upgrades"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"
)

const UpgradeName = "v6.1.0"

func CreateUpgradeHandler() upgradetypes.UpgradeHandler {
	return upgrades.EmptyUpgradeHandler()
}
