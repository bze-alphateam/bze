package txfeecollector_test

import (
	"testing"

	"cosmossdk.io/log"
	sdkmath "cosmossdk.io/math"
	"cosmossdk.io/store"
	"cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/stretchr/testify/require"

	"github.com/bze-alphateam/bze/x/txfeecollector/keeper"
	txfeecollector "github.com/bze-alphateam/bze/x/txfeecollector/module"
	"github.com/bze-alphateam/bze/x/txfeecollector/types"
)

// TestRunMigrations_TxfeecollectorV2ToV3_ThroughModuleManager proves the piece the migrator
// unit tests cannot: the 2->3 migration is registered under the module's name and version in
// the configurator, so a module manager holding {txfeecollector: 2} - the mainnet VersionMap
// the v8.2.0 upgrade handler receives - lands on ConsensusVersion 3 with an empty
// BlockedIbcInbound param and the pre-existing params preserved. A missing or mis-numbered
// RegisterMigration would fail here the way it would fail the upgrade.
func TestRunMigrations_TxfeecollectorV2ToV3_ThroughModuleManager(t *testing.T) {
	storeKey := storetypes.NewKVStoreKey(types.StoreKey)
	db := dbm.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	require.NoError(t, stateStore.LoadLatestVersion())

	registry := codectypes.NewInterfaceRegistry()
	types.RegisterInterfaces(registry)
	cdc := codec.NewProtoCodec(registry)

	k := keeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(storeKey),
		log.NewNopLogger(),
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		nil,
		nil,
		nil,
		nil,
	)
	ctx := sdk.NewContext(stateStore, cmtproto.Header{}, false, log.NewNopLogger())

	// a consensus-version-2 params record: the field added in version 3 is absent, which is
	// exactly what a v8.1.1 node has in its store
	v2Params := types.Params{
		ValidatorMinGasFee:   sdk.NewDecCoinFromDec("ubze", sdkmath.LegacyNewDecWithPrec(5, 2)),
		MaxBalanceIterations: 42,
	}
	require.Nil(t, v2Params.BlockedIbcInbound)
	ctx.KVStore(storeKey).Set(types.ParamsKey, cdc.MustMarshal(&v2Params))

	msr := baseapp.NewMsgServiceRouter()
	msr.SetInterfaceRegistry(registry)
	qr := baseapp.NewGRPCQueryRouter()
	qr.SetInterfaceRegistry(registry)
	cfg := module.NewConfigurator(cdc, msr, qr)

	am := txfeecollector.NewAppModule(cdc, k, nil, nil, nil)
	mm := module.NewManager(am)
	require.NoError(t, mm.RegisterServices(cfg))
	require.Equal(t, uint64(3), am.ConsensusVersion())

	newVM, err := mm.RunMigrations(ctx, cfg, module.VersionMap{types.ModuleName: 2})
	require.NoError(t, err)
	require.Equal(t, uint64(3), newVM[types.ModuleName])

	got := k.GetParams(ctx)
	require.Equal(t, v2Params.ValidatorMinGasFee, got.ValidatorMinGasFee)
	require.Equal(t, v2Params.MaxBalanceIterations, got.MaxBalanceIterations)
	require.Empty(t, got.BlockedIbcInbound)
	require.NoError(t, got.Validate())

	// a re-run from the new version is a no-op (no migration registered at 3)
	again, err := mm.RunMigrations(ctx, cfg, newVM)
	require.NoError(t, err)
	require.Equal(t, uint64(3), again[types.ModuleName])
	after := k.GetParams(ctx)
	require.True(t, after.Equal(&got))
}
