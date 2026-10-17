package v830_test

import (
	"context"
	"errors"
	"testing"

	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdkruntime "github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/stretchr/testify/require"

	v830 "github.com/bze-alphateam/bze/app/upgrades/v830"
	"github.com/bze-alphateam/bze/x/dao/keeper"
	dao "github.com/bze-alphateam/bze/x/dao/module"
	daotypes "github.com/bze-alphateam/bze/x/dao/types"
)

const fakeModuleName = "fake"

// fakeModule is a module at consensus version 2 that registers a 1->2 migration,
// standing in for the SDK modules whose migrations the handler must run.
type fakeModule struct {
	migrationErr error
	migrated     *int
}

func (fakeModule) RegisterLegacyAminoCodec(*codec.LegacyAmino)                 {}
func (fakeModule) RegisterInterfaces(codectypes.InterfaceRegistry)             {}
func (fakeModule) RegisterGRPCGatewayRoutes(client.Context, *runtime.ServeMux) {}
func (fakeModule) IsOnePerModuleType()                                         {}
func (fakeModule) IsAppModule()                                                {}
func (fakeModule) Name() string                                                { return fakeModuleName }
func (fakeModule) ConsensusVersion() uint64                                    { return 2 }

func (m fakeModule) RegisterServices(cfg module.Configurator) {
	err := cfg.RegisterMigration(fakeModuleName, 1, func(sdk.Context) error {
		*m.migrated++
		return m.migrationErr
	})
	if err != nil {
		panic(err)
	}
}

func setup(t *testing.T, migrationErr error) (upgradetypes.UpgradeHandler, context.Context, *int) {
	t.Helper()

	migrated := 0
	mm := module.NewManager(fakeModule{migrationErr: migrationErr, migrated: &migrated})
	cfg := module.NewConfigurator(
		codec.NewProtoCodec(codectypes.NewInterfaceRegistry()),
		baseapp.NewMsgServiceRouter(),
		baseapp.NewGRPCQueryRouter(),
	)
	require.NoError(t, mm.RegisterServices(cfg))

	ctx := testutil.DefaultContext(storetypes.NewKVStoreKey("v830"), storetypes.NewTransientStoreKey("v830_t"))

	return v830.CreateUpgradeHandler(cfg, mm), ctx, &migrated
}

func TestUpgradeName(t *testing.T) {
	require.Equal(t, "v8.3.0", v830.UpgradeName)
}

func TestHandler_RunsModuleMigrations(t *testing.T) {
	handler, ctx, migrated := setup(t, nil)

	vm, err := handler(ctx, upgradetypes.Plan{Name: v830.UpgradeName}, module.VersionMap{fakeModuleName: 1})
	require.NoError(t, err)
	require.Equal(t, 1, *migrated)
	require.Equal(t, uint64(2), vm[fakeModuleName])
}

func TestHandler_UpToDateModuleIsNotMigrated(t *testing.T) {
	handler, ctx, migrated := setup(t, nil)

	vm, err := handler(ctx, upgradetypes.Plan{Name: v830.UpgradeName}, module.VersionMap{fakeModuleName: 2})
	require.NoError(t, err)
	require.Zero(t, *migrated)
	require.Equal(t, uint64(2), vm[fakeModuleName])
}

func TestHandler_MigrationErrorFailsTheUpgrade(t *testing.T) {
	handler, ctx, _ := setup(t, errors.New("boom"))

	_, err := handler(ctx, upgradetypes.Plan{Name: v830.UpgradeName}, module.VersionMap{fakeModuleName: 1})
	require.ErrorContains(t, err, "boom")
}

// The stores of the modules ibc-go v10 removed and of x/group and x/nft are
// deleted at the upgrade height, spelled out so a typo cannot leave one behind
// or delete a live store.
func TestGetStoreUpgrades_DeletesRemovedModuleStores(t *testing.T) {
	upgrades := v830.GetStoreUpgrades()
	require.Empty(t, upgrades.Renamed)
	require.ElementsMatch(t, []string{"capability", "feeibc", "group", "nft"}, upgrades.Deleted)
}

// The new x/dao module gets its store at the upgrade height, and only it.
func TestGetStoreUpgrades_AddsDaoStore(t *testing.T) {
	upgrades := v830.GetStoreUpgrades()
	require.Equal(t, []string{"dao"}, upgrades.Added)
	require.NotContains(t, upgrades.Deleted, "dao")
}

// x/dao has no VersionMap entry before the upgrade, so the handler initialises
// it from its default genesis and records its consensus version.
func TestHandler_InitialisesDaoModule(t *testing.T) {
	registry := codectypes.NewInterfaceRegistry()
	daotypes.RegisterInterfaces(registry)
	cdc := codec.NewProtoCodec(registry)
	storeKey := storetypes.NewKVStoreKey(daotypes.StoreKey)
	k := keeper.NewKeeper(
		cdc,
		sdkruntime.NewKVStoreService(storeKey),
		log.NewNopLogger(),
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		nil,
		nil,
	)

	msr := baseapp.NewMsgServiceRouter()
	msr.SetInterfaceRegistry(registry)
	mm := module.NewManager(dao.NewAppModule(cdc, &k, nil, nil))
	cfg := module.NewConfigurator(cdc, msr, baseapp.NewGRPCQueryRouter())
	require.NoError(t, mm.RegisterServices(cfg))

	ctx := testutil.DefaultContext(storeKey, storetypes.NewTransientStoreKey("v830_t"))

	vm, err := v830.CreateUpgradeHandler(cfg, mm)(ctx, upgradetypes.Plan{Name: v830.UpgradeName}, module.VersionMap{})
	require.NoError(t, err)
	require.Equal(t, uint64(dao.ConsensusVersion), vm[daotypes.ModuleName])
	require.Equal(t, daotypes.DefaultParams(), k.GetParams(ctx))
}
