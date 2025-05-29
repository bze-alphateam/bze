package v2_test

import (
	ct "github.com/bze-alphateam/bze/x/cointrunk/module"
	"testing"
	"time"

	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	"github.com/stretchr/testify/require"

	"github.com/bze-alphateam/bze/x/cointrunk/exported"
	v2 "github.com/bze-alphateam/bze/x/cointrunk/migrations/v2"
	"github.com/bze-alphateam/bze/x/cointrunk/types"
	"github.com/bze-alphateam/bze/x/cointrunk/v1types"
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

// TestMigrateParams tests the successful migration of params from legacy subspace to module store
func TestMigrateParams(t *testing.T) {
	encCfg := moduletestutil.MakeTestEncodingConfig(ct.AppModuleBasic{})
	cdc := encCfg.Codec

	storeKey := storetypes.NewKVStoreKey(types.ModuleName)
	tKey := storetypes.NewTransientStoreKey("transient_test")
	ctx := testutil.DefaultContext(storeKey, tKey)

	store := prefix.NewStore(ctx.KVStore(storeKey), []byte{})

	// Create mock subspace with default params
	legacySubspace := newMockSubspace(types.DefaultParams())

	// Run migration
	require.NoError(t, v2.MigrateParams(ctx, store, legacySubspace, cdc))

	// Verify params were stored correctly
	var res types.Params
	bz := store.Get(types.ParamsKey)
	require.NotNil(t, bz, "params should be stored in the new location")
	require.NoError(t, cdc.Unmarshal(bz, &res))
	require.Equal(t, legacySubspace.ps, res)
}

// TestMigratePublishers tests the migration of publishers from v1 to v2 format
func TestMigratePublishers(t *testing.T) {
	encCfg := moduletestutil.MakeTestEncodingConfig(ct.AppModuleBasic{})
	cdc := encCfg.Codec

	storeKey := storetypes.NewKVStoreKey(types.ModuleName)
	tKey := storetypes.NewTransientStoreKey("transient_test")
	ctx := testutil.DefaultContext(storeKey, tKey)

	store := prefix.NewStore(ctx.KVStore(storeKey), []byte{})

	// Create v1 publishers with int64 respect field
	v1Publishers := []v1types.Publisher{
		{
			Name:          "Publisher One",
			Address:       "address1",
			Active:        true,
			ArticlesCount: 5,
			CreatedAt:     time.Now().Unix(),
			Respect:       1000,
		},
		{
			Name:          "Publisher Two",
			Address:       "address2",
			Active:        false,
			ArticlesCount: 10,
			CreatedAt:     time.Now().Unix(),
			Respect:       -500,
		},
		{
			Name:          "Publisher Three",
			Address:       "address3",
			Active:        true,
			ArticlesCount: 0,
			CreatedAt:     time.Now().Unix(),
			Respect:       0,
		},
	}

	// Store v1 publishers in the store
	for _, pub := range v1Publishers {
		bz := cdc.MustMarshal(&pub)
		store.Set(types.PublisherKey(pub.Address), bz)
	}

	// Run migration
	require.NoError(t, v2.MigratePublishers(store, cdc))

	// Verify publishers were migrated correctly
	for _, v1Pub := range v1Publishers {
		// Get migrated publisher
		bz := store.Get(types.PublisherKey(v1Pub.Address))
		require.NotNil(t, bz, "publisher should exist after migration")

		var v2Pub types.Publisher
		require.NoError(t, cdc.Unmarshal(bz, &v2Pub))

		// Verify all fields were migrated correctly
		require.Equal(t, v1Pub.Name, v2Pub.Name)
		require.Equal(t, v1Pub.Address, v2Pub.Address)
		require.Equal(t, v1Pub.Active, v2Pub.Active)
		require.Equal(t, v1Pub.ArticlesCount, v2Pub.ArticlesCount)
		require.Equal(t, v1Pub.CreatedAt, v2Pub.CreatedAt)

		// Verify respect was converted from int64 to string
		expectedRespect := "1000"
		if v1Pub.Address == "address2" {
			expectedRespect = "-500"
		} else if v1Pub.Address == "address3" {
			expectedRespect = "0"
		}
		require.Equal(t, expectedRespect, v2Pub.Respect)
	}
}
