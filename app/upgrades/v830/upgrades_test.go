package v830_test

import (
	"context"
	"errors"
	"testing"

	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/stretchr/testify/require"

	v830 "github.com/bze-alphateam/bze/app/upgrades/v830"
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

// The stores of the modules ibc-go v10 removed are deleted at the upgrade
// height, spelled out so a typo cannot leave one behind or delete a live store.
func TestGetStoreUpgrades_DeletesCapabilityAndFeeStores(t *testing.T) {
	upgrades := v830.GetStoreUpgrades()
	require.Empty(t, upgrades.Added)
	require.Empty(t, upgrades.Renamed)
	require.ElementsMatch(t, []string{"capability", "feeibc"}, upgrades.Deleted)
}
