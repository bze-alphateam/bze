package v800

import (
	"context"
	"cosmossdk.io/math"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	rTypes "github.com/bze-alphateam/bze/x/rewards/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	distrkeeper "github.com/cosmos/cosmos-sdk/x/distribution/keeper"
)

const UpgradeName = "v8.0.0"

func CreateUpgradeHandler(
	cfg module.Configurator,
	mm *module.Manager,
	bank bankkeeper.Keeper,
	distr distrkeeper.Keeper,
	acc authkeeper.AccountKeeper,
) upgradetypes.UpgradeHandler {

	return func(c context.Context, _plan upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		ctx := sdk.UnwrapSDKContext(c)

		//we had a bug that sent staking reward fee to the Rewards module.
		//We need to move those funds from rewards module to community pool
		rAcc := acc.GetModuleAddress(rTypes.ModuleName)
		//hardcoded denom to run it only for mainnet
		rBal := bank.GetBalance(ctx, rAcc, "ubze")
		if rBal.Amount.GTE(math.NewInt(50_000_000000)) {
			toSend := sdk.NewCoins(sdk.NewInt64Coin("ubze", 50_000_000000))
			err := distr.FundCommunityPool(ctx, toSend, rAcc)
			if err != nil {
				ctx.Logger().Error("could not migrate funds from rewards module to community pool", "error", err)
			} else {
				ctx.Logger().Info("migrated funds from rewards module to community pool")
			}
		}

		return mm.RunMigrations(ctx, cfg, vm)
	}
}
