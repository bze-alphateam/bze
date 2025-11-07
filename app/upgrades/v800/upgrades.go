package v800

import (
	"context"

	storetypes "cosmossdk.io/store/types"
	circuittypes "cosmossdk.io/x/circuit/types"
	"cosmossdk.io/x/nft"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	tradebintypes "github.com/bze-alphateam/bze/x/tradebin/types"
	txfeecollectormoduletypes "github.com/bze-alphateam/bze/x/txfeecollector/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	consensustypes "github.com/cosmos/cosmos-sdk/x/consensus/types"
	crisistypes "github.com/cosmos/cosmos-sdk/x/crisis/types"
	"github.com/cosmos/cosmos-sdk/x/group"
	ibcfeetypes "github.com/cosmos/ibc-go/v8/modules/apps/29-fee/types"
)

const UpgradeName = "v8.0.0-rc3"

func CreateUpgradeHandler(
	//cfg module.Configurator, //TODO: commented on testnet version (branch)
	//mm *module.Manager,
	//bank bankkeeper.Keeper,
	//distr distrkeeper.Keeper,
	acc *authkeeper.AccountKeeper,
	// mainDenom string,
	// paramsKeeper *paramskeeper.Keeper,
	// consensusParamsKeeper *consensuskeeper.Keeper,
) upgradetypes.UpgradeHandler {

	return func(c context.Context, _plan upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		ctx := sdk.UnwrapSDKContext(c)

		//TODO: COMMENTED only on testnet version (branch) to test module account permissions migration
		//baseAppLegacySS := paramsKeeper.Subspace(baseapp.Paramspace).WithKeyTable(paramstypes.ConsensusParamsKeyTable())
		//for _, subspace := range paramsKeeper.GetSubspaces() {
		//	s := subspace
		//	if s.HasKeyTable() {
		//		continue
		//	}
		//
		//	var keyTable paramstypes.KeyTable
		//	switch s.Name() {
		//	// sdk
		//	case authtypes.ModuleName:
		//		keyTable = authtypes.ParamKeyTable()
		//	case banktypes.ModuleName:
		//		keyTable = banktypes.ParamKeyTable()
		//	case stakingtypes.ModuleName:
		//		keyTable = stakingtypes.ParamKeyTable()
		//	case minttypes.ModuleName:
		//		keyTable = minttypes.ParamKeyTable()
		//	case distrtypes.ModuleName:
		//		keyTable = distrtypes.ParamKeyTable()
		//	case slashingtypes.ModuleName:
		//		keyTable = slashingtypes.ParamKeyTable()
		//	case govtypes.ModuleName:
		//		keyTable = govv1.ParamKeyTable()
		//	case crisistypes.ModuleName:
		//		keyTable = crisistypes.ParamKeyTable()
		//
		//	// ibc types
		//	case ibctransfertypes.ModuleName:
		//		keyTable = ibctransfertypes.ParamKeyTable()
		//	case icahosttypes.SubModuleName:
		//		keyTable = icahosttypes.ParamKeyTable()
		//	case icacontrollertypes.SubModuleName:
		//		keyTable = icacontrollertypes.ParamKeyTable()
		//
		//	//bze
		//	case cointrunkmoduletypes.ModuleName:
		//		keyTable = cointrunkmoduletypes.ParamKeyTable()
		//	case burnermoduletypes.ModuleName:
		//		keyTable = burnermoduletypes.ParamKeyTable()
		//	case tradebintypes.ModuleName:
		//		keyTable = tradebintypes.ParamKeyTable()
		//	case rTypes.ModuleName:
		//		keyTable = rv1Types.ParamKeyTable()
		//	case tokenfactorytypes.ModuleName:
		//		keyTable = tokenfactoryv1types.ParamKeyTable()
		//
		//	default:
		//		continue
		//	}
		//
		//	s.WithKeyTable(keyTable)
		//}
		//
		//err := baseapp.MigrateParams(ctx, baseAppLegacySS, consensusParamsKeeper.ParamsStore)
		//if err != nil {
		//	return nil, err
		//}
		//
		//newVm, err := mm.RunMigrations(ctx, cfg, vm)
		//if err != nil {
		//	return newVm, err
		//}
		//
		////we had a bug that sent staking reward fee to the Rewards module.
		////We need to move those funds from rewards module to community pool
		//rAcc := acc.GetModuleAddress(rTypes.ModuleName)
		////hardcoded denom to run it only for mainnet
		//rBal := bank.GetBalance(ctx, rAcc, mainDenom)
		//if rBal.Amount.GTE(math.NewInt(18_130_000000)) {
		//	toSend := sdk.NewCoins(sdk.NewInt64Coin(mainDenom, 18_130_000000))
		//	err = distr.FundCommunityPool(ctx, toSend, rAcc)
		//	if err != nil {
		//		ctx.Logger().Error("could not migrate funds from rewards module to community pool", "error", err)
		//	} else {
		//		ctx.Logger().Info("migrated funds from rewards module to community pool")
		//	}
		//}

		//migrate modules permissions
		//{Account: tradebinmoduletypes.ModuleName, Permissions: []string{authtypes.Burner, authtypes.Minter}},
		tradebinAcc := acc.GetModuleAccount(ctx, tradebintypes.ModuleName)
		if tradebinAcc != nil {
			if modAcc, ok := tradebinAcc.(*authtypes.ModuleAccount); ok {
				ctx.Logger().Info("migrating permissions for tradebin module account")
				modAcc.Permissions = []string{authtypes.Minter, authtypes.Burner}
				acc.SetModuleAccount(ctx, modAcc)
			} else {
				ctx.Logger().Error("could not migrate permissions for tradebin module account")
			}
		} else {
			ctx.Logger().Error("could not update tradebin module account permission. not found")
		}

		return vm, nil
	}
}

func GetStoreUpgrades() *storetypes.StoreUpgrades {
	return &storetypes.StoreUpgrades{
		Added:   []string{nft.ModuleName, group.ModuleName, circuittypes.ModuleName, consensustypes.ModuleName, crisistypes.ModuleName, ibcfeetypes.ModuleName, "icacontroller", "icahost", txfeecollectormoduletypes.ModuleName},
		Deleted: []string{"scavenge"},
	}
}
