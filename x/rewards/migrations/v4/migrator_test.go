package v4_test

import (
	"testing"

	"cosmossdk.io/math"
	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	v4 "github.com/bze-alphateam/bze/x/rewards/migrations/v4"
	ct "github.com/bze-alphateam/bze/x/rewards/module"
	"github.com/bze-alphateam/bze/x/rewards/types"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	"github.com/stretchr/testify/require"
)

func TestMigrate_WithExistingParams(t *testing.T) {
	encCfg := moduletestutil.MakeTestEncodingConfig(ct.AppModuleBasic{})
	cdc := encCfg.Codec

	storeKey := storetypes.NewKVStoreKey(types.ModuleName)
	tKey := storetypes.NewTransientStoreKey("transient_test")
	ctx := testutil.DefaultContext(storeKey, tKey)

	store := prefix.NewStore(ctx.KVStore(storeKey), []byte{})

	// Set up existing params (simulating v3 state)
	existingParams := types.Params{
		CreateStakingRewardFee: sdk.NewCoin("ubze", math.NewInt(100)),
		CreateTradingRewardFee: sdk.NewCoin("ubze", math.NewInt(200)),
		ExtraGasForExitStake:   0, // Old default or not set
	}
	bz := cdc.MustMarshal(&existingParams)
	store.Set(types.ParamsKey, bz)

	// Run migration
	require.NoError(t, v4.Migrate(ctx, store, cdc))

	// Verify params were updated
	var res types.Params
	bz = store.Get(types.ParamsKey)
	require.NotNil(t, bz)
	require.NoError(t, cdc.Unmarshal(bz, &res))

	// Existing params should be preserved
	require.Equal(t, "100ubze", res.CreateStakingRewardFee.String())
	require.Equal(t, "200ubze", res.CreateTradingRewardFee.String())

	// New param should have default value
	require.Equal(t, types.DefaultExtraGasForExitStake, res.ExtraGasForExitStake)
}

func TestMigrate_WithEmptyStore(t *testing.T) {
	encCfg := moduletestutil.MakeTestEncodingConfig(ct.AppModuleBasic{})
	cdc := encCfg.Codec

	storeKey := storetypes.NewKVStoreKey(types.ModuleName)
	tKey := storetypes.NewTransientStoreKey("transient_test")
	ctx := testutil.DefaultContext(storeKey, tKey)

	store := prefix.NewStore(ctx.KVStore(storeKey), []byte{})

	// Run migration on empty store - should fail because empty params have nil coins
	// that fail validation. This is expected since the migration assumes v3 params exist.
	err := v4.Migrate(ctx, store, cdc)
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid")
}

func TestMigrate_PreservesExistingFees(t *testing.T) {
	encCfg := moduletestutil.MakeTestEncodingConfig(ct.AppModuleBasic{})
	cdc := encCfg.Codec

	storeKey := storetypes.NewKVStoreKey(types.ModuleName)
	tKey := storetypes.NewTransientStoreKey("transient_test")
	ctx := testutil.DefaultContext(storeKey, tKey)

	store := prefix.NewStore(ctx.KVStore(storeKey), []byte{})

	// Set up custom fees
	customParams := types.Params{
		CreateStakingRewardFee: sdk.NewCoin("uatom", math.NewInt(999)),
		CreateTradingRewardFee: sdk.NewCoin("uosmo", math.NewInt(888)),
		ExtraGasForExitStake:   500_000,
	}
	bz := cdc.MustMarshal(&customParams)
	store.Set(types.ParamsKey, bz)

	require.NoError(t, v4.Migrate(ctx, store, cdc))

	var res types.Params
	bz = store.Get(types.ParamsKey)
	require.NotNil(t, bz)
	require.NoError(t, cdc.Unmarshal(bz, &res))

	// Custom fees should be preserved
	require.Equal(t, "uatom", res.CreateStakingRewardFee.Denom)
	require.Equal(t, math.NewInt(999), res.CreateStakingRewardFee.Amount)
	require.Equal(t, "uosmo", res.CreateTradingRewardFee.Denom)
	require.Equal(t, math.NewInt(888), res.CreateTradingRewardFee.Amount)

	// ExtraGasForExitStake should be overwritten with default
	require.Equal(t, types.DefaultExtraGasForExitStake, res.ExtraGasForExitStake)
}
