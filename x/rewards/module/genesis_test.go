package rewards_test

import (
	"github.com/bze-alphateam/bze/x/rewards/testutil"
	"go.uber.org/mock/gomock"
	"testing"

	keepertest "github.com/bze-alphateam/bze/testutil/keeper"
	"github.com/bze-alphateam/bze/testutil/nullify"
	rewards "github.com/bze-alphateam/bze/x/rewards/module"
	"github.com/bze-alphateam/bze/x/rewards/types"
	"github.com/stretchr/testify/require"
)

func TestGenesis(t *testing.T) {
	genesisState := types.GenesisState{
		Params: types.DefaultParams(),

		// this line is used by starport scaffolding # genesis/test/state
	}
	ctrl := gomock.NewController(t)
	acc := testutil.NewMockAccountKeeper(ctrl)

	k, ctx := keepertest.RewardsKeeper(t, nil, nil, nil, acc)
	rewards.InitGenesis(ctx, k, genesisState)
	got := rewards.ExportGenesis(ctx, k)
	require.NotNil(t, got)

	nullify.Fill(&genesisState)
	nullify.Fill(got)

	// this line is used by starport scaffolding # genesis/test/assert
}

// TestGenesis_IndexesBoostParticipants asserts InitGenesis builds the boost
// participant reverse index for genesis-imported participants, so a fresh-genesis
// chain matches the post-migration invariant (participants reachable by the
// boost finalization sweep).
func TestGenesis_IndexesBoostParticipants(t *testing.T) {
	genesisState := types.GenesisState{
		Params: types.DefaultParams(),
		StakingRewardParticipantList: []types.StakingRewardParticipant{
			{Address: "bze1aaa", RewardId: "000000000001", Amount: "100", JoinedAt: "0"},
			{Address: "bze1bbb", RewardId: "000000000001", Amount: "200", JoinedAt: "0"},
			{Address: "bze1aaa", RewardId: "000000000002", Amount: "300", JoinedAt: "0"},
		},
	}

	ctrl := gomock.NewController(t)
	acc := testutil.NewMockAccountKeeper(ctrl)
	k, ctx := keepertest.RewardsKeeper(t, nil, nil, nil, acc)

	rewards.InitGenesis(ctx, k, genesisState)

	for _, p := range genesisState.StakingRewardParticipantList {
		require.True(t, k.HasBoostParticipantIndex(ctx, p.RewardId, p.Address),
			"missing reverse-index entry for %s/%s", p.RewardId, p.Address)
	}

	require.ElementsMatch(t,
		[]string{"bze1aaa", "bze1bbb"},
		k.GetBoostParticipantsFromCursor(ctx, "000000000001", "", 0))
	require.Equal(t,
		[]string{"bze1aaa"},
		k.GetBoostParticipantsFromCursor(ctx, "000000000002", "", 0))
}
