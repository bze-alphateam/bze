package rewards_test

import (
	"testing"

	"cosmossdk.io/log"
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
	"go.uber.org/mock/gomock"
	"google.golang.org/protobuf/encoding/protowire"

	"github.com/bze-alphateam/bze/x/rewards/keeper"
	rewards "github.com/bze-alphateam/bze/x/rewards/module"
	"github.com/bze-alphateam/bze/x/rewards/testutil"
	"github.com/bze-alphateam/bze/x/rewards/types"
)

// wiringFixture is a rewards keeper whose raw module store is reachable, so a v4-shaped params
// record can be planted exactly as a v8.1.1 node stored it (keeper.SetParams would refuse it:
// the zero Denom Rewards fee coins fail validation, which is precisely why the migration exists).
type wiringFixture struct {
	k        *keeper.Keeper
	ctx      sdk.Context
	storeKey *storetypes.KVStoreKey
	cdc      codec.Codec
	registry codectypes.InterfaceRegistry
	acc      *testutil.MockAccountKeeper
	bank     *testutil.MockBankKeeper
	epoch    *testutil.MockEpochKeeper
	trade    *testutil.MockTradingKeeper
}

func newWiringFixture(t *testing.T) wiringFixture {
	t.Helper()
	ctrl := gomock.NewController(t)

	storeKey := storetypes.NewKVStoreKey(types.StoreKey)
	db := dbm.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	require.NoError(t, stateStore.LoadLatestVersion())

	registry := codectypes.NewInterfaceRegistry()
	types.RegisterInterfaces(registry)
	cdc := codec.NewProtoCodec(registry)

	f := wiringFixture{
		storeKey: storeKey,
		cdc:      cdc,
		registry: registry,
		acc:      testutil.NewMockAccountKeeper(ctrl),
		bank:     testutil.NewMockBankKeeper(ctrl),
		epoch:    testutil.NewMockEpochKeeper(ctrl),
		trade:    testutil.NewMockTradingKeeper(ctrl),
	}
	k := keeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(storeKey),
		log.NewNopLogger(),
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		f.bank,
		f.epoch,
		f.trade,
		f.acc,
	)
	f.k = &k
	f.ctx = sdk.NewContext(stateStore, cmtproto.Header{}, false, log.NewNopLogger())

	return f
}

// plantV4Params writes the three consensus-version-4 params fields as raw protobuf, nothing else.
func (f wiringFixture) plantV4Params(stakingFee, tradingFee sdk.Coin, extraGas uint64) {
	coin := func(c sdk.Coin) []byte {
		var b []byte
		b = protowire.AppendTag(b, 1, protowire.BytesType)
		b = protowire.AppendString(b, c.Denom)
		b = protowire.AppendTag(b, 2, protowire.BytesType)
		b = protowire.AppendString(b, c.Amount.String())
		return b
	}
	var bz []byte
	bz = protowire.AppendTag(bz, 1, protowire.BytesType)
	bz = protowire.AppendBytes(bz, coin(stakingFee))
	bz = protowire.AppendTag(bz, 2, protowire.BytesType)
	bz = protowire.AppendBytes(bz, coin(tradingFee))
	bz = protowire.AppendTag(bz, 3, protowire.VarintType)
	bz = protowire.AppendVarint(bz, extraGas)

	f.ctx.KVStore(f.storeKey).Set(types.ParamsKey, bz)
}

// manager builds the real SDK configurator + module manager the app uses, with the rewards
// AppModule registered exactly as app.go does (through RegisterServices). RunMigrations on this
// manager is what the v8.2.0 upgrade handler executes; the returned msg service router is what
// BaseApp dispatches transactions through.
func (f wiringFixture) manager(t *testing.T) (*module.Manager, module.Configurator, *baseapp.MsgServiceRouter, rewards.AppModule) {
	t.Helper()

	msr := baseapp.NewMsgServiceRouter()
	msr.SetInterfaceRegistry(f.registry)
	qr := baseapp.NewGRPCQueryRouter()
	qr.SetInterfaceRegistry(f.registry)
	cfg := module.NewConfigurator(f.cdc, msr, qr)

	am := rewards.NewAppModule(f.cdc, f.k, f.acc, f.bank, f.trade, nil)
	mm := module.NewManager(am)
	require.NoError(t, mm.RegisterServices(cfg))

	return mm, cfg, msr, am
}

// TestRunMigrations_RewardsV4ToV5_ThroughModuleManager proves the piece the migrator unit tests
// cannot: the 4→5 migration is registered under the module's name and version in the
// configurator, so a module manager holding {rewards: 4} — the mainnet VersionMap the v8.2.0
// upgrade handler receives — lands on ConsensusVersion 5 with the Denom Rewards defaults set and
// the pre-existing params preserved. A missing or mis-numbered RegisterMigration would fail here
// the way it would fail the upgrade.
func TestRunMigrations_RewardsV4ToV5_ThroughModuleManager(t *testing.T) {
	f := newWiringFixture(t)

	stakingFee := sdk.NewInt64Coin("ubze", 25_000_000000)
	tradingFee := sdk.NewInt64Coin("ubze", 50_000_000000)
	f.plantV4Params(stakingFee, tradingFee, 1_000_000)

	mm, cfg, _, am := f.manager(t)
	require.Equal(t, uint64(5), am.ConsensusVersion())

	newVM, err := mm.RunMigrations(f.ctx, cfg, module.VersionMap{types.ModuleName: 4})
	require.NoError(t, err)
	require.Equal(t, uint64(5), newVM[types.ModuleName])

	got := f.k.GetParams(f.ctx)
	expected := types.DefaultParams()
	expected.CreateStakingRewardFee = stakingFee
	expected.CreateTradingRewardFee = tradingFee
	expected.ExtraGasForExitStake = 1_000_000
	require.True(t, got.Equal(&expected), "params after upgrade:\n got %v\nwant %v", got, expected)
	require.NoError(t, got.Validate())

	// a re-run from the new version is a no-op (no migration registered at 5), params unchanged
	again, err := mm.RunMigrations(f.ctx, cfg, newVM)
	require.NoError(t, err)
	require.Equal(t, uint64(5), again[types.ModuleName])
	after := f.k.GetParams(f.ctx)
	require.True(t, after.Equal(&expected))
}
