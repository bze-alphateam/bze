package rewards_test

import (
	"testing"

	keepertest "github.com/bze-alphateam/bze/testutil/keeper"
	"github.com/bze-alphateam/bze/testutil/nullify"
	"github.com/bze-alphateam/bze/x/rewards"
	"github.com/bze-alphateam/bze/x/rewards/types"
	"github.com/stretchr/testify/require"
)

func TestGenesis(t *testing.T) {
	genesisState := types.GenesisState{
		Params: types.DefaultParams(),

		StakingRewardList: []types.StakingReward{
		{
			RewardId: "0",
},
		{
			RewardId: "1",
},
	},
	TradingRewardList: []types.TradingReward{
		{
			RewardId: "0",
},
		{
			RewardId: "1",
},
	},
	// this line is used by starport scaffolding # genesis/test/state
	}

	k, ctx := keepertest.RewardsKeeper(t)
	rewards.InitGenesis(ctx, *k, genesisState)
	got := rewards.ExportGenesis(ctx, *k)
	require.NotNil(t, got)

	nullify.Fill(&genesisState)
	nullify.Fill(got)

	require.ElementsMatch(t, genesisState.StakingRewardList, got.StakingRewardList)
require.ElementsMatch(t, genesisState.TradingRewardList, got.TradingRewardList)
// this line is used by starport scaffolding # genesis/test/assert
}
