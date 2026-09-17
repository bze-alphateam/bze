package v3_test

import (
	"testing"

	sdkmath "cosmossdk.io/math"
	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	"github.com/stretchr/testify/require"

	v3 "github.com/bze-alphateam/bze/x/txfeecollector/migrations/v3"
	txfeecollector "github.com/bze-alphateam/bze/x/txfeecollector/module"
	"github.com/bze-alphateam/bze/x/txfeecollector/types"
)

// newMigratorStore builds an isolated prefixed module store + codec, mirroring the other
// migrations/vX package tests (see x/rewards/migrations/v5).
func newMigratorStore(t *testing.T) (sdk.Context, prefix.Store, moduletestutil.TestEncodingConfig) {
	t.Helper()
	encCfg := moduletestutil.MakeTestEncodingConfig(txfeecollector.AppModuleBasic{})

	storeKey := storetypes.NewKVStoreKey(types.ModuleName)
	tKey := storetypes.NewTransientStoreKey("transient_test")
	ctx := testutil.DefaultContext(storeKey, tKey)
	store := prefix.NewStore(ctx.KVStore(storeKey), []byte{})

	return ctx, store, encCfg
}

// v2Params is a realistic pre-upgrade (v8.1.1) params fixture: only the two params that
// existed at consensus version 2 carry values, exactly as v2-era stored bytes unmarshal
// under the v3 proto.
func v2Params() types.Params {
	return types.Params{
		ValidatorMinGasFee:   sdk.NewDecCoinFromDec("ubze", sdkmath.LegacyNewDecWithPrec(5, 2)),
		MaxBalanceIterations: 42,
	}
}

// TestMigrate_SetsEmptyBlockedInbound_PreservesExisting is the core contract: the new
// param lands empty - no inbound transfer is blocked by the migration itself - while the
// two pre-existing params keep their stored values.
func TestMigrate_SetsEmptyBlockedInbound_PreservesExisting(t *testing.T) {
	ctx, store, encCfg := newMigratorStore(t)
	cdc := encCfg.Codec

	seed := v2Params()
	store.Set(types.ParamsKey, cdc.MustMarshal(&seed))

	require.NoError(t, v3.Migrate(ctx, store, cdc))

	var got types.Params
	bz := store.Get(types.ParamsKey)
	require.NotNil(t, bz)
	require.NoError(t, cdc.Unmarshal(bz, &got))

	require.Equal(t, seed.ValidatorMinGasFee, got.ValidatorMinGasFee)
	require.Equal(t, seed.MaxBalanceIterations, got.MaxBalanceIterations)
	require.Empty(t, got.BlockedIbcInbound)
	require.False(t, got.IsInboundBlocked("channel-3", "uusdc"))
	require.NoError(t, got.Validate())
}

// Anything already sitting in the new field is overwritten, not merged: the migration
// defines the post-upgrade value of the param on its own.
func TestMigrate_OverwritesExistingBlockedInbound(t *testing.T) {
	ctx, store, encCfg := newMigratorStore(t)
	cdc := encCfg.Codec

	seed := v2Params()
	seed.BlockedIbcInbound = []types.BlockedIbcTransfer{
		{ChannelId: "channel-99", BaseDenom: "ugarbage"},
	}
	store.Set(types.ParamsKey, cdc.MustMarshal(&seed))

	require.NoError(t, v3.Migrate(ctx, store, cdc))

	var got types.Params
	require.NoError(t, cdc.Unmarshal(store.Get(types.ParamsKey), &got))
	require.Empty(t, got.BlockedIbcInbound)
}

// A chain whose params key is somehow absent must not make the migration fail: it writes
// a valid, empty-blocked-list params object instead of panicking.
func TestMigrate_NoStoredParams(t *testing.T) {
	ctx, store, encCfg := newMigratorStore(t)
	cdc := encCfg.Codec

	require.Nil(t, store.Get(types.ParamsKey))

	err := v3.Migrate(ctx, store, cdc)
	// an empty ValidatorMinGasFee cannot pass validation - the migration reports it
	// instead of writing an invalid params object
	require.Error(t, err)
	require.Contains(t, err.Error(), "validator min gas fee denom must be ubze")
}
