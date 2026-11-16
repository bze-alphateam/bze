package v4_test

import (
	"testing"

	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	v4 "github.com/bze-alphateam/bze/x/tokenfactory/migrations/v4"
	"github.com/bze-alphateam/bze/x/tokenfactory/types"
)

func authorityKey(denom string) []byte {
	return append(types.GetDenomPrefixStore(denom), []byte(types.DenomAuthorityMetadataKey)...)
}

// Admin-held denoms stay mintable, renounced denoms become non-mintable, and nothing else in the
// store is touched.
func TestMigrate(t *testing.T) {
	cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	storeKey := storetypes.NewKVStoreKey(types.ModuleName)
	ctx := testutil.DefaultContext(storeKey, storetypes.NewTransientStoreKey("transient_test"))
	store := ctx.KVStore(storeKey)

	admin := sdk.AccAddress("admin").String()
	held := "factory/" + admin + "/held"
	renounced := "factory/" + admin + "/renounced"
	nested := "factory/" + admin + "/nested/sub"

	// consensus-version-3 records: the mintable field does not exist yet, so it reads false
	store.Set(authorityKey(held), cdc.MustMarshal(&types.DenomAuthority{Admin: admin}))
	store.Set(authorityKey(renounced), cdc.MustMarshal(&types.DenomAuthority{Admin: ""}))
	store.Set(authorityKey(nested), cdc.MustMarshal(&types.DenomAuthority{Admin: admin}))

	// unrelated records next to the authorities
	creatorKey := append(types.GetCreatorPrefix(admin), []byte(held)...)
	store.Set(creatorKey, []byte(held))
	store.Set(types.ParamsKey, []byte("params"))

	mintable, nonMintable, err := v4.Migrate(store, cdc)
	require.NoError(t, err)
	require.Equal(t, 2, mintable)
	require.Equal(t, 1, nonMintable)

	expect := map[string]types.DenomAuthority{
		held:      {Admin: admin, Mintable: true},
		renounced: {Admin: "", Mintable: false},
		nested:    {Admin: admin, Mintable: true},
	}
	for denom, want := range expect {
		var got types.DenomAuthority
		require.NoError(t, cdc.Unmarshal(store.Get(authorityKey(denom)), &got))
		require.Equal(t, want, got, denom)
	}

	require.Equal(t, []byte(held), store.Get(creatorKey))
	require.Equal(t, []byte("params"), store.Get(types.ParamsKey))
}

// An empty store migrates without error.
func TestMigrate_Empty(t *testing.T) {
	cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	storeKey := storetypes.NewKVStoreKey(types.ModuleName)
	ctx := testutil.DefaultContext(storeKey, storetypes.NewTransientStoreKey("transient_test"))

	mintable, nonMintable, err := v4.Migrate(ctx.KVStore(storeKey), cdc)
	require.NoError(t, err)
	require.Zero(t, mintable)
	require.Zero(t, nonMintable)
}
