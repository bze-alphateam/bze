package v700

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"
)

const UpgradeName = "v7.0.0"

func CreateUpgradeHandler() upgradetypes.UpgradeHandler {
	return func(ctx sdk.Context, _plan upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {

		return fromVM, nil
	}
}
