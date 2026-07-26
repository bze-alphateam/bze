package v5_test

import (
	"testing"

	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	"github.com/stretchr/testify/require"

	v5 "github.com/bze-alphateam/bze/x/rewards/migrations/v5"
	ct "github.com/bze-alphateam/bze/x/rewards/module"
	"github.com/bze-alphateam/bze/x/rewards/types"
)

func TestMigrate(t *testing.T) {
	encCfg := moduletestutil.MakeTestEncodingConfig(ct.AppModuleBasic{})
	cdc := encCfg.Codec

	storeKey := storetypes.NewKVStoreKey(types.ModuleName)
	tKey := storetypes.NewTransientStoreKey("transient_test")
	ctx := testutil.DefaultContext(storeKey, tKey)

	store := prefix.NewStore(ctx.KVStore(storeKey), []byte{})

	// pre-upgrade params: valid staking/trading/exit-gas values but the boost
	// params unset (zero value), the way they exist before consensus version 5.
	preParams := types.NewParams(
		sdk.NewInt64Coin("ubze", 100),
		sdk.NewInt64Coin("ubze", 200),
		types.DefaultExtraGasForExitStake,
	)
	store.Set(types.ParamsKey, cdc.MustMarshal(&preParams))

	// seed participants across multiple rewards and multiple addresses, each with
	// an empty snapshot map and no reverse-index entry.
	participantStore := prefix.NewStore(store, types.KeyPrefix(types.StakingRewardParticipantKeyPrefix))
	seeded := []types.StakingRewardParticipant{
		{Address: "bze1aaa", RewardId: "000000000001", Amount: "100", JoinedAt: "0"},
		{Address: "bze1bbb", RewardId: "000000000001", Amount: "200", JoinedAt: "0"},
		{Address: "bze1aaa", RewardId: "000000000002", Amount: "300", JoinedAt: "0"},
		{Address: "bze1ccc", RewardId: "000000000002", Amount: "400", JoinedAt: "0"},
	}
	for _, p := range seeded {
		require.Nil(t, p.BoostSnapshots, "seeded participant must have an empty snapshot map")
		participantStore.Set(types.StakingRewardParticipantKey(p.Address, p.RewardId), cdc.MustMarshal(&p))
	}

	// no reverse-index entries exist before the migration
	indexStore := prefix.NewStore(store, types.KeyPrefix(types.BoostParticipantIndexKeyPrefix))
	require.Equal(t, 0, countEntries(indexStore))

	// run migration
	require.NoError(t, v5.Migrate(ctx, store, cdc))

	// exactly one reverse-index entry per participant, at the expected key
	require.Equal(t, len(seeded), countEntries(indexStore))
	for _, p := range seeded {
		require.True(t,
			indexStore.Has(types.BoostParticipantIndexKey(p.RewardId, p.Address)),
			"missing reverse-index entry for %s/%s", p.RewardId, p.Address,
		)
	}

	// participant records are untouched (still decode, snapshot map still empty)
	for _, want := range seeded {
		bz := participantStore.Get(types.StakingRewardParticipantKey(want.Address, want.RewardId))
		require.NotNil(t, bz)
		var got types.StakingRewardParticipant
		require.NoError(t, cdc.Unmarshal(bz, &got))
		require.Equal(t, want.Amount, got.Amount)
		require.Empty(t, got.BoostSnapshots)
	}

	// boost params set to defaults, pre-existing params preserved
	var res types.Params
	require.NoError(t, cdc.Unmarshal(store.Get(types.ParamsKey), &res))
	require.NoError(t, res.Validate())
	require.Equal(t, types.DefaultCreateBoostFee, res.CreateBoostFee)
	require.Equal(t, types.DefaultMaxBoostsPerReward, res.MaxBoostsPerReward)
	require.Equal(t, "100ubze", res.CreateStakingRewardFee.String())
	require.Equal(t, "200ubze", res.CreateTradingRewardFee.String())

	// idempotent: a second run leaves the same entry count and params
	require.NoError(t, v5.Migrate(ctx, store, cdc))
	require.Equal(t, len(seeded), countEntries(indexStore))
}

func countEntries(store prefix.Store) int {
	it := storetypes.KVStorePrefixIterator(store, []byte{})
	defer it.Close()

	count := 0
	for ; it.Valid(); it.Next() {
		count++
	}

	return count
}
