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

		BoostList: []types.Boost{
			{Id: "000000000001", RewardId: "000000000001", Denom: "ubze", DailyAmount: "1000", Duration: 5, Payouts: 1, DistributedStake: "0.2", Creator: "bze1creator"},
			{Id: "000000000002", RewardId: "000000000001", Denom: "uvdl", DailyAmount: "300", Duration: 3, Payouts: 3, DistributedStake: "0.5", Creator: "bze1creator"},
		},
		BoostParticipantList: []types.BoostParticipant{
			{Address: "bze1participant", RewardId: "000000000001", BoostId: "000000000001", JoinedAt: "0.1"},
		},
		BoostCounter: 2,

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

	require.ElementsMatch(t, genesisState.BoostList, got.BoostList)
	require.ElementsMatch(t, genesisState.BoostParticipantList, got.BoostParticipantList)
	require.Equal(t, genesisState.BoostCounter, got.BoostCounter)

	// this line is used by starport scaffolding # genesis/test/assert
}
