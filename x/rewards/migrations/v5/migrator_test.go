package v5_test

import (
	"testing"

	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/testutil"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"

	v5 "github.com/bze-alphateam/bze/x/rewards/migrations/v5"
	rewards "github.com/bze-alphateam/bze/x/rewards/module"
	"github.com/bze-alphateam/bze/x/rewards/types"
)

// newMigratorStore builds an isolated prefixed module store + codec, mirroring the other
// migrations/vX package tests (see x/cointrunk/migrations/v3). v5.Migrate reads and writes
// the rewards Params under types.ParamsKey through this store.
func newMigratorStore(t *testing.T) (sdk.Context, prefix.Store, moduletestutil.TestEncodingConfig) {
	t.Helper()
	encCfg := moduletestutil.MakeTestEncodingConfig(rewards.AppModuleBasic{})

	storeKey := storetypes.NewKVStoreKey(types.ModuleName)
	tKey := storetypes.NewTransientStoreKey("transient_test")
	ctx := testutil.DefaultContext(storeKey, tKey)
	store := prefix.NewStore(ctx.KVStore(storeKey), []byte{})

	return ctx, store, encCfg
}

// v4Params is a realistic pre-upgrade (v8.1.1) params fixture: only the three params that
// existed at consensus version 4 carry values; the seven Denom Rewards fields are left at
// their proto zero values, exactly as v4-era stored bytes unmarshal under the v5 proto.
func v4Params() types.Params {
	return types.Params{
		CreateTradingRewardFee: sdk.NewInt64Coin("ubze", 111),
		CreateStakingRewardFee: sdk.NewInt64Coin("ubze", 222),
		ExtraGasForExitStake:   333,
	}
}

// TestMigrate_SetsDenomRewardDefaults_PreservesExisting is the core contract: the seven DR
// params are forced to their defaults while the three pre-existing params keep their stored
// values. Any garbage previously sitting in the DR fields is overwritten, not merged.
func TestMigrate_SetsDenomRewardDefaults_PreservesExisting(t *testing.T) {
	ctx, store, encCfg := newMigratorStore(t)
	cdc := encCfg.Codec

	// seed v4 params, then plant garbage in the DR fields to prove they are overwritten
	seed := v4Params()
	seed.CreateDenomRewardFee = sdk.NewInt64Coin("ugarbage", 1)
	seed.MaxPrizeDenomsPerDr = 99999
	seed.DenomRewardLock = 42
	seed.DenomRewardMinStake = 7
	store.Set(types.ParamsKey, cdc.MustMarshal(&seed))

	require.NoError(t, v5.Migrate(ctx, store, cdc))

	var got types.Params
	bz := store.Get(types.ParamsKey)
	require.NotNil(t, bz)
	require.NoError(t, cdc.Unmarshal(bz, &got))

	// the three pre-existing params are untouched
	require.Equal(t, seed.CreateTradingRewardFee, got.CreateTradingRewardFee)
	require.Equal(t, seed.CreateStakingRewardFee, got.CreateStakingRewardFee)
	require.Equal(t, seed.ExtraGasForExitStake, got.ExtraGasForExitStake)

	// the seven DR params are exactly the defaults
	require.Equal(t, types.DefaultCreateRewardFee, got.CreateDenomRewardFee)
	require.Equal(t, types.DefaultCreateRewardFee, got.CreateDenomRewardPrizeFee)
	require.Equal(t, types.DefaultAddDenomRewardScheduleFee, got.AddDenomRewardScheduleFee)
	require.Equal(t, types.DefaultMaxPrizeDenomsPerDr, got.MaxPrizeDenomsPerDr)
	require.Equal(t, types.DefaultExtraGasForDenomExit, got.ExtraGasForDenomExit)
	require.Equal(t, types.DefaultDenomRewardLock, got.DenomRewardLock)
	require.Equal(t, types.DefaultDenomRewardMinStake, got.DenomRewardMinStake)

	require.NoError(t, got.Validate())
}

// TestMigrate_Idempotent proves a second run is a no-op: re-applying the migration over
// already-migrated params yields byte-identical stored params. Guards against a re-run of
// the v8.2.0 upgrade handler (or a replayed migration) corrupting state.
func TestMigrate_Idempotent(t *testing.T) {
	ctx, store, encCfg := newMigratorStore(t)
	cdc := encCfg.Codec

	seed := v4Params()
	store.Set(types.ParamsKey, cdc.MustMarshal(&seed))

	require.NoError(t, v5.Migrate(ctx, store, cdc))
	first := store.Get(types.ParamsKey)
	firstCopy := append([]byte(nil), first...)

	require.NoError(t, v5.Migrate(ctx, store, cdc))
	second := store.Get(types.ParamsKey)

	require.Equal(t, firstCopy, second, "second migration must be a no-op")
}

// TestMigrate_MissingParams_FailsValidation documents the current contract when the params
// key is absent: Migrate starts from a zero-value Params, sets only the seven DR fields, and
// leaves the three pre-existing fee/gas params at their zero values — which fail
// Params.Validate (a zero-value fee Coin is invalid). Migrate therefore returns an error
// instead of writing partially-defaulted params. On a real chain the rewards params always
// exist by this point (they are set at InitGenesis), so this path is defensive only.
func TestMigrate_MissingParams_FailsValidation(t *testing.T) {
	ctx, store, encCfg := newMigratorStore(t)
	cdc := encCfg.Codec

	// no params written to the store at all
	require.Nil(t, store.Get(types.ParamsKey))

	err := v5.Migrate(ctx, store, cdc)
	require.Error(t, err, "missing params must not be silently defaulted")

	// nothing partially-migrated was persisted
	require.Nil(t, store.Get(types.ParamsKey))
}
