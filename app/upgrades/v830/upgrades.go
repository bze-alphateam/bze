package v830

import (
	"context"

	corestore "cosmossdk.io/core/store"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
)

const UpgradeName = "v8.3.0"

func CreateUpgradeHandler(
	cfg module.Configurator,
	mm *module.Manager,
	guardianKeeper GuardianKeeper,
	rateLimitKeeper RateLimitKeeper,
	bankKeeper BankKeeper,
	tradebinKeeper TradebinKeeper,
	metadataKeeper MetadataKeeper,
	rewardsKeeper RewardsKeeper,
	transferStore corestore.KVStoreService,
	transferKeeper TransferKeeper,
) upgradetypes.UpgradeHandler {

	return func(c context.Context, _plan upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		ctx := sdk.UnwrapSDKContext(c)

		sanitizeDenomTraces(ctx, transferStore, transferKeeper)

		// RunMigrations triggers module-level migrations based on ConsensusVersion
		// changes: the ibc-go v10 module migrations. The cosmos-sdk v0.50 -> v0.53
		// bump changes no module's version. x/dao is new: it has no VersionMap
		// entry, so RunMigrations runs its InitGenesis with the default genesis.
		// x/guardian is new too; its guardian address is set afterwards.
		newVM, err := mm.RunMigrations(ctx, cfg, vm)
		if err != nil {
			return nil, err
		}

		if err := setGuardian(ctx, guardianKeeper); err != nil {
			return nil, err
		}

		if err := addInitialRateLimits(ctx, rateLimitKeeper); err != nil {
			return nil, err
		}

		if err := sweepBurnerFeeCollector(ctx, bankKeeper); err != nil {
			return nil, err
		}

		backfillLpMetadata(ctx, tradebinKeeper, metadataKeeper)

		normaliseStakingRewardAmounts(ctx, rewardsKeeper)

		return newVM, nil
	}
}
