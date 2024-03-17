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

func createNStakingRewardParticipant(keeper *keeper.Keeper, ctx sdk.Context, n int) []types.StakingRewardParticipant {
	items := make([]types.StakingRewardParticipant, n)
	for i := range items {
		items[i].Index = strconv.Itoa(i)

		keeper.SetStakingRewardParticipant(ctx, items[i])
	}
	return items
}

func TestStakingRewardParticipantGet(t *testing.T) {
	keeper, ctx := keepertest.RewardsKeeper(t)
	items := createNStakingRewardParticipant(keeper, ctx, 10)
	for _, item := range items {
		rst, found := keeper.GetStakingRewardParticipant(ctx,
			item.Index,
		)
		require.True(t, found)
		require.Equal(t,
			nullify.Fill(&item),
			nullify.Fill(&rst),
		)
	}
}
func TestStakingRewardParticipantRemove(t *testing.T) {
	keeper, ctx := keepertest.RewardsKeeper(t)
	items := createNStakingRewardParticipant(keeper, ctx, 10)
	for _, item := range items {
		keeper.RemoveStakingRewardParticipant(ctx,
			item.Index,
		)
		_, found := keeper.GetStakingRewardParticipant(ctx,
			item.Index,
		)
		require.False(t, found)
	}
}

func TestStakingRewardParticipantGetAll(t *testing.T) {
	keeper, ctx := keepertest.RewardsKeeper(t)
	items := createNStakingRewardParticipant(keeper, ctx, 10)
	require.ElementsMatch(t,
		nullify.Fill(items),
		nullify.Fill(keeper.GetAllStakingRewardParticipant(ctx)),
	)
}
