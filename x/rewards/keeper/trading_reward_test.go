package keeper_test

import (
	"strconv"
	"testing"

	keepertest "github.com/bze-alphateam/bze/testutil/keeper"
	"github.com/bze-alphateam/bze/testutil/nullify"
	"github.com/bze-alphateam/bze/x/rewards/keeper"
	"github.com/bze-alphateam/bze/x/rewards/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

// Prevent strconv unused error
var _ = strconv.IntSize

func createNTradingReward(keeper *keeper.Keeper, ctx sdk.Context, n int) []types.TradingReward {
	items := make([]types.TradingReward, n)
	for i := range items {
		items[i].RewardId = strconv.Itoa(i)

		keeper.SetPendingTradingReward(ctx, items[i])
	}
	return items
}

func TestTradingRewardGet(t *testing.T) {
	keeper, ctx := keepertest.RewardsKeeper(t)
	items := createNTradingReward(keeper, ctx, 10)
	for _, item := range items {
		rst, found := keeper.GetPendingTradingReward(ctx,
			item.RewardId,
		)
		require.True(t, found)
		require.Equal(t,
			nullify.Fill(&item),
			nullify.Fill(&rst),
		)
	}
}
func TestTradingRewardRemove(t *testing.T) {
	keeper, ctx := keepertest.RewardsKeeper(t)
	items := createNTradingReward(keeper, ctx, 10)
	for _, item := range items {
		keeper.RemovePendingTradingReward(ctx,
			item.RewardId,
		)
		_, found := keeper.GetPendingTradingReward(ctx,
			item.RewardId,
		)
		require.False(t, found)
	}
}

func TestTradingRewardGetAll(t *testing.T) {
	keeper, ctx := keepertest.RewardsKeeper(t)
	items := createNTradingReward(keeper, ctx, 10)
	require.ElementsMatch(t,
		nullify.Fill(items),
		nullify.Fill(keeper.GetAllPendingTradingReward(ctx)),
	)
}
