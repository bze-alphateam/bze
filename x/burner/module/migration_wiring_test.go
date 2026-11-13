package burner_test

import (
	"testing"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
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

	"github.com/bze-alphateam/bze/x/burner/keeper"
	burner "github.com/bze-alphateam/bze/x/burner/module"
	"github.com/bze-alphateam/bze/x/burner/testutil"
	"github.com/bze-alphateam/bze/x/burner/types"
)

// A consensus version 4 node stored params with periodic_burning_weeks only. Running the module migrations from
// version 4 through the module manager gives them the default limits of MsgMoveIbcLockedCoins and keeps the burning
// period.
func TestRunMigrations_BurnerV4ToV5_ThroughModuleManager(t *testing.T) {
	ctrl := gomock.NewController(t)
	storeKey := storetypes.NewKVStoreKey(types.StoreKey)
	db := dbm.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	require.NoError(t, stateStore.LoadLatestVersion())

	registry := codectypes.NewInterfaceRegistry()
	types.RegisterInterfaces(registry)
	cdc := codec.NewProtoCodec(registry)
	acc := testutil.NewMockAccountKeeper(ctrl)
	bank := testutil.NewMockBankKeeper(ctrl)
	epoch := testutil.NewMockEpochKeeper(ctrl)
	k := keeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(storeKey),
		log.NewNopLogger(),
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		bank,
		acc,
		epoch,
		testutil.NewMockTradeKeeper(ctrl),
		testutil.NewMockGuardianKeeper(ctrl),
	)
	ctx := sdk.NewContext(stateStore, cmtproto.Header{}, false, log.NewNopLogger())

	// the params record as a consensus version 4 node wrote it: field 1 (periodic_burning_weeks) only
	var bz []byte
	bz = protowire.AppendTag(bz, 1, protowire.VarintType)
	bz = protowire.AppendVarint(bz, 4)
	ctx.KVStore(storeKey).Set(types.ParamsKey, bz)

	msr := baseapp.NewMsgServiceRouter()
	msr.SetInterfaceRegistry(registry)
	qr := baseapp.NewGRPCQueryRouter()
	qr.SetInterfaceRegistry(registry)
	cfg := module.NewConfigurator(cdc, msr, qr)
	am := burner.NewAppModule(cdc, k, acc, bank, epoch, nil)
	mm := module.NewManager(am)
	require.NoError(t, mm.RegisterServices(cfg))
	require.Equal(t, uint64(5), am.ConsensusVersion())

	newVM, err := mm.RunMigrations(ctx, cfg, module.VersionMap{types.ModuleName: 4})
	require.NoError(t, err)
	require.Equal(t, uint64(5), newVM[types.ModuleName])

	got := k.GetParams(ctx)
	require.Equal(t, int64(4), got.PeriodicBurningWeeks)
	require.True(t, got.IbcMoveMaxReserveRatio.Equal(math.LegacyNewDecWithPrec(3, 2)), "ratio %s", got.IbcMoveMaxReserveRatio)
	require.Equal(t, uint64(1000), got.IbcMoveCooldownBlocks)
	require.NoError(t, got.Validate())
}
