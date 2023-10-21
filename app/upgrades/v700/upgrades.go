package v700

import (
	"github.com/cosmos/cosmos-sdk/baseapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	consensusparamkeeper "github.com/cosmos/cosmos-sdk/x/consensus/keeper"
	paramskeeper "github.com/cosmos/cosmos-sdk/x/params/keeper"
	paramstypes "github.com/cosmos/cosmos-sdk/x/params/types"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"
)

const UpgradeName = "v7.0.0"

func CreateUpgradeHandler(paramsKeeper *paramskeeper.Keeper, consensusKeeper *consensusparamkeeper.Keeper) upgradetypes.UpgradeHandler {
	baseAppLegacySS := paramsKeeper.Subspace(baseapp.Paramspace).WithKeyTable(paramstypes.ConsensusParamsKeyTable())

	return func(ctx sdk.Context, _plan upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
		// Migrate Tendermint consensus parameters from x/params module to a
		// dedicated x/consensus module.
		baseapp.MigrateParams(ctx, baseAppLegacySS, consensusKeeper)

		return fromVM, nil
	}
}
