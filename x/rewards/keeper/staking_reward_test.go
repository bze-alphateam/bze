package keeper_test

import (
	"strconv"
	"testing"

	"github.com/bze-alphateam/bze/x/rewards/keeper"
	"github.com/bze-alphateam/bze/x/rewards/types"
	keepertest "github.com/bze-alphateam/bze/testutil/keeper"
	"github.com/bze-alphateam/bze/testutil/nullify"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

// Prevent strconv unused error
var _ = strconv.IntSize

func createNStakingReward(keeper *keeper.Keeper, ctx sdk.Context, n int) []types.StakingReward {
	items := make([]types.StakingReward, n)
	for i := range items {
		items[i].RewardId = strconv.Itoa(i)
        
		keeper.SetStakingReward(ctx, items[i])
	}
	return items
}

func TestStakingRewardGet(t *testing.T) {
	keeper, ctx := keepertest.RewardsKeeper(t)
	items := createNStakingReward(keeper, ctx, 10)
	for _, item := range items {
		rst, found := keeper.GetStakingReward(ctx,
		    item.RewardId,
            
		)
		require.True(t, found)
		require.Equal(t,
			nullify.Fill(&item),
			nullify.Fill(&rst),
		)
	}
}
func TestStakingRewardRemove(t *testing.T) {
	keeper, ctx := keepertest.RewardsKeeper(t)
	items := createNStakingReward(keeper, ctx, 10)
	for _, item := range items {
		keeper.RemoveStakingReward(ctx,
		    item.RewardId,
            
		)
		_, found := keeper.GetStakingReward(ctx,
		    item.RewardId,
            
		)
		require.False(t, found)
	}
}

func TestStakingRewardGetAll(t *testing.T) {
	keeper, ctx := keepertest.RewardsKeeper(t)
	items := createNStakingReward(keeper, ctx, 10)
	require.ElementsMatch(t,
		nullify.Fill(items),
		nullify.Fill(keeper.GetAllStakingReward(ctx)),
	)
}
