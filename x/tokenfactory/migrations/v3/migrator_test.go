package v3_test

import (
	ct "github.com/bze-alphateam/bze/x/tokenfactory/module"
	"testing"

	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	"github.com/stretchr/testify/require"

	"github.com/bze-alphateam/bze/x/tokenfactory/exported"
	v2 "github.com/bze-alphateam/bze/x/tokenfactory/migrations/v3"
	"github.com/bze-alphateam/bze/x/tokenfactory/types"
)

// mockSubspace implements the exported.Subspace interface for testing
type mockSubspace struct {
	ps types.Params
}

func newMockSubspace(ps types.Params) mockSubspace {
	return mockSubspace{ps: ps}
}

func (ms mockSubspace) GetParamSet(_ sdk.Context, ps exported.ParamSet) {
	*ps.(*types.Params) = ms.ps
}

// TestMigrate tests the successful migration of params from legacy subspace to module store
func TestMigrate(t *testing.T) {
	encCfg := moduletestutil.MakeTestEncodingConfig(ct.AppModuleBasic{})
	cdc := encCfg.Codec

	storeKey := storetypes.NewKVStoreKey(types.ModuleName)
	tKey := storetypes.NewTransientStoreKey("transient_test")
	ctx := testutil.DefaultContext(storeKey, tKey)

	store := prefix.NewStore(ctx.KVStore(storeKey), []byte{})

	// Create mock subspace with default params
	legacySubspace := newMockSubspace(types.DefaultParams())

	// Run migration
	require.NoError(t, v2.Migrate(ctx, store, legacySubspace, cdc))

	// Verify params were stored correctly
	var res types.Params
	bz := store.Get(types.ParamsKey)
	require.NotNil(t, bz, "params should be stored in the new location")
	require.NoError(t, cdc.Unmarshal(bz, &res))
	require.Equal(t, legacySubspace.ps, res)
}
