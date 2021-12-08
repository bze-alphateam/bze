package keeper_test

import (
	"strconv"
	"testing"

	keepertest "github.com/bze-alphateam/bzedgev5/testutil/keeper"
	"github.com/bze-alphateam/bzedgev5/x/scavenge/keeper"
	"github.com/bze-alphateam/bzedgev5/x/scavenge/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

// Prevent strconv unused error
var _ = strconv.IntSize

func createNCommit(keeper *keeper.Keeper, ctx sdk.Context, n int) []types.Commit {
	items := make([]types.Commit, n)
	for i := range items {
		items[i].Index = strconv.Itoa(i)

		keeper.SetCommit(ctx, items[i])
	}
	return items
}

func TestCommitGet(t *testing.T) {
	keeper, ctx := keepertest.ScavengeKeeper(t)
	items := createNCommit(keeper, ctx, 10)
	for _, item := range items {
		rst, found := keeper.GetCommit(ctx,
			item.Index,
		)
		require.True(t, found)
		require.Equal(t, item, rst)
	}
}
func TestCommitRemove(t *testing.T) {
	keeper, ctx := keepertest.ScavengeKeeper(t)
	items := createNCommit(keeper, ctx, 10)
	for _, item := range items {
		keeper.RemoveCommit(ctx,
			item.Index,
		)
		_, found := keeper.GetCommit(ctx,
			item.Index,
		)
		require.False(t, found)
	}
}

func TestCommitGetAll(t *testing.T) {
	keeper, ctx := keepertest.ScavengeKeeper(t)
	items := createNCommit(keeper, ctx, 10)
	require.ElementsMatch(t, items, keeper.GetAllCommit(ctx))
}
