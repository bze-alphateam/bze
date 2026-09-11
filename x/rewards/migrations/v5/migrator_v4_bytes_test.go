package v5_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protowire"

	sdk "github.com/cosmos/cosmos-sdk/types"

	v5 "github.com/bze-alphateam/bze/x/rewards/migrations/v5"
	"github.com/bze-alphateam/bze/x/rewards/types"
)

// encodeV4Params hand-encodes a rewards Params record exactly as a v8.1.1 binary stored it:
// only the three fields that existed at consensus version 4 are on the wire —
//
//	1: createStakingRewardFee (cosmos.base.v1beta1.Coin)
//	2: createTradingRewardFee (cosmos.base.v1beta1.Coin)
//	3: extraGasForExitStake   (uint64)
//
// The v4Params() fixture used by the other tests is marshalled with the v5 proto, which always
// emits the seven non-nullable Denom Rewards fields (as empty coins / zero scalars), so those
// bytes are NOT what the migration meets on mainnet. These are.
func encodeV4Params(stakingFee, tradingFee sdk.Coin, extraGas uint64) []byte {
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

	return bz
}

// TestMigrate_GenuineV4Bytes_MainnetValues feeds Migrate the exact byte shape a v8.1.1 node has
// under the params key, carrying the mainnet values as of the v8.2.0 release
// (25,000 BZE / 50,000 BZE / 1,000,000 gas). The migrated params must be the mainnet values plus
// the Denom Rewards defaults from DefaultParams(), and must validate.
func TestMigrate_GenuineV4Bytes_MainnetValues(t *testing.T) {
	ctx, store, encCfg := newMigratorStore(t)
	cdc := encCfg.Codec

	stakingFee := sdk.NewInt64Coin("ubze", 25_000_000000)
	tradingFee := sdk.NewInt64Coin("ubze", 50_000_000000)
	const extraGas uint64 = 1_000_000

	raw := encodeV4Params(stakingFee, tradingFee, extraGas)

	// sanity: the fixture decodes under the v5 proto into "old fields set, DR fields zero"
	var decoded types.Params
	require.NoError(t, cdc.Unmarshal(raw, &decoded))
	require.Equal(t, stakingFee, decoded.CreateStakingRewardFee)
	require.Equal(t, tradingFee, decoded.CreateTradingRewardFee)
	require.Equal(t, extraGas, decoded.ExtraGasForExitStake)
	require.True(t, decoded.CreateDenomRewardFee.IsNil() || decoded.CreateDenomRewardFee.Amount.IsNil())
	require.Zero(t, decoded.MaxPrizeDenomsPerDr)
	require.Zero(t, decoded.DenomRewardLock)
	// and it is genuinely shorter than what the v5 proto would emit for the same values
	require.Less(t, len(raw), len(cdc.MustMarshal(&decoded)))

	store.Set(types.ParamsKey, raw)
	require.NoError(t, v5.Migrate(ctx, store, cdc))

	var got types.Params
	require.NoError(t, cdc.Unmarshal(store.Get(types.ParamsKey), &got))

	expected := types.DefaultParams()
	expected.CreateStakingRewardFee = stakingFee
	expected.CreateTradingRewardFee = tradingFee
	expected.ExtraGasForExitStake = extraGas
	require.True(t, got.Equal(&expected), "migrated params differ from mainnet values + DR defaults:\n got %v\nwant %v", got, expected)
	require.NoError(t, got.Validate())
}

// The migration's Denom Rewards values must be the same ones a fresh chain gets from
// DefaultParams(): a chain upgraded through v8.2.0 and a chain started at v8.2.0 must agree.
func TestMigrate_DenomRewardDefaultsMatchDefaultParams(t *testing.T) {
	ctx, store, encCfg := newMigratorStore(t)
	cdc := encCfg.Codec

	defaults := types.DefaultParams()
	store.Set(types.ParamsKey, encodeV4Params(defaults.CreateStakingRewardFee, defaults.CreateTradingRewardFee, defaults.ExtraGasForExitStake))
	require.NoError(t, v5.Migrate(ctx, store, cdc))

	var got types.Params
	require.NoError(t, cdc.Unmarshal(store.Get(types.ParamsKey), &got))
	require.True(t, got.Equal(&defaults), "migrated params must equal DefaultParams():\n got %v\nwant %v", got, defaults)
}
