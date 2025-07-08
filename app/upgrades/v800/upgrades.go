package v800

import (
	"context"
	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	circuittypes "cosmossdk.io/x/circuit/types"
	"cosmossdk.io/x/nft"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	burnermoduletypes "github.com/bze-alphateam/bze/x/burner/types"
	cointrunkmoduletypes "github.com/bze-alphateam/bze/x/cointrunk/types"
	rTypes "github.com/bze-alphateam/bze/x/rewards/types"
	tokenfactorytypes "github.com/bze-alphateam/bze/x/tokenfactory/types"
	tradebintypes "github.com/bze-alphateam/bze/x/tradebin/types"
	txfeecollectormoduletypes "github.com/bze-alphateam/bze/x/txfeecollector/types"
	"github.com/cosmos/cosmos-sdk/baseapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	consensuskeeper "github.com/cosmos/cosmos-sdk/x/consensus/keeper"
	consensustypes "github.com/cosmos/cosmos-sdk/x/consensus/types"
	crisistypes "github.com/cosmos/cosmos-sdk/x/crisis/types"
	distrkeeper "github.com/cosmos/cosmos-sdk/x/distribution/keeper"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	govv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	"github.com/cosmos/cosmos-sdk/x/group"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	paramskeeper "github.com/cosmos/cosmos-sdk/x/params/keeper"
	paramstypes "github.com/cosmos/cosmos-sdk/x/params/types"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	icacontrollertypes "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/controller/types"
	icahosttypes "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/host/types"
	ibcfeetypes "github.com/cosmos/ibc-go/v8/modules/apps/29-fee/types"
	ibctransfertypes "github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
)

const UpgradeName = "v8.0.0"

func CreateUpgradeHandler(
	cfg module.Configurator,
	mm *module.Manager,
	bank bankkeeper.Keeper,
	distr distrkeeper.Keeper,
	acc authkeeper.AccountKeeper,
	mainDenom string,
	paramsKeeper *paramskeeper.Keeper,
	consensusParamsKeeper *consensuskeeper.Keeper,
) upgradetypes.UpgradeHandler {

	return func(c context.Context, _plan upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		ctx := sdk.UnwrapSDKContext(c)
		vm[burnermoduletypes.ModuleName] = 1
		baseAppLegacySS := paramsKeeper.Subspace(baseapp.Paramspace).WithKeyTable(paramstypes.ConsensusParamsKeyTable())
		for _, subspace := range paramsKeeper.GetSubspaces() {
			s := subspace
			if s.HasKeyTable() {
				continue
			}

			var keyTable paramstypes.KeyTable
			switch s.Name() {
			// sdk
			case authtypes.ModuleName:
				keyTable = authtypes.ParamKeyTable()
			case banktypes.ModuleName:
				keyTable = banktypes.ParamKeyTable()
			case stakingtypes.ModuleName:
				keyTable = stakingtypes.ParamKeyTable()
			case minttypes.ModuleName:
				keyTable = minttypes.ParamKeyTable()
			case distrtypes.ModuleName:
				keyTable = distrtypes.ParamKeyTable()
			case slashingtypes.ModuleName:
				keyTable = slashingtypes.ParamKeyTable()
			case govtypes.ModuleName:
				keyTable = govv1.ParamKeyTable()
			case crisistypes.ModuleName:
				keyTable = crisistypes.ParamKeyTable()

			// ibc types
			case ibctransfertypes.ModuleName:
				keyTable = ibctransfertypes.ParamKeyTable()
			case icahosttypes.SubModuleName:
				keyTable = icahosttypes.ParamKeyTable()
			case icacontrollertypes.SubModuleName:
				keyTable = icacontrollertypes.ParamKeyTable()

			//bze
			case cointrunkmoduletypes.ModuleName:
				keyTable = cointrunkmoduletypes.ParamKeyTable()
			case burnermoduletypes.ModuleName:
				keyTable = burnermoduletypes.ParamKeyTable()
			case tradebintypes.ModuleName:
				keyTable = tradebintypes.ParamKeyTable()
			case rTypes.ModuleName:
				keyTable = rTypes.ParamKeyTable()
			case tokenfactorytypes.ModuleName:
				keyTable = tokenfactorytypes.ParamKeyTable()

			default:
				continue
			}

			s.WithKeyTable(keyTable)
		}

		err := baseapp.MigrateParams(ctx, baseAppLegacySS, consensusParamsKeeper.ParamsStore)
		if err != nil {
			return nil, err
		}

		newVm, err := mm.RunMigrations(ctx, cfg, vm)
		if err != nil {
			return newVm, err
		}

		//we had a bug that sent staking reward fee to the Rewards module.
		//We need to move those funds from rewards module to community pool
		rAcc := acc.GetModuleAddress(rTypes.ModuleName)
		//hardcoded denom to run it only for mainnet
		rBal := bank.GetBalance(ctx, rAcc, mainDenom)
		if rBal.Amount.GTE(math.NewInt(50_000_000000)) {
			toSend := sdk.NewCoins(sdk.NewInt64Coin(mainDenom, 50_000_000000))
			err = distr.FundCommunityPool(ctx, toSend, rAcc)
			if err != nil {
				ctx.Logger().Error("could not migrate funds from rewards module to community pool", "error", err)
			} else {
				ctx.Logger().Info("migrated funds from rewards module to community pool")
			}
		}

		return newVm, nil
	}
}

func GetStoreUpgrades() *storetypes.StoreUpgrades {
	return &storetypes.StoreUpgrades{
		Added:   []string{nft.ModuleName, group.ModuleName, circuittypes.ModuleName, consensustypes.ModuleName, crisistypes.ModuleName, ibcfeetypes.ModuleName, "icacontroller", "icahost", txfeecollectormoduletypes.ModuleName},
		Deleted: []string{"scavenge"},
	}
}
