package keeper_test

import (
	"strconv"
	"testing"

	keepertest "github.com/bzedgev5/testutil/keeper"
	"github.com/bzedgev5/x/scavenge/keeper"
	"github.com/bzedgev5/x/scavenge/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

// Prevent strconv unused error
var _ = strconv.IntSize

func createNScavenge(keeper *keeper.Keeper, ctx sdk.Context, n int) []types.Scavenge {
	items := make([]types.Scavenge, n)
	for i := range items {
		items[i].Index = strconv.Itoa(i)

		keeper.SetScavenge(ctx, items[i])
	}
	return items
}

func TestScavengeGet(t *testing.T) {
	keeper, ctx := keepertest.ScavengeKeeper(t)
	items := createNScavenge(keeper, ctx, 10)
	for _, item := range items {
		rst, found := keeper.GetScavenge(ctx,
			item.Index,
		)
		require.True(t, found)
		require.Equal(t, item, rst)
	}
}
func TestScavengeRemove(t *testing.T) {
	keeper, ctx := keepertest.ScavengeKeeper(t)
	items := createNScavenge(keeper, ctx, 10)
	for _, item := range items {
		keeper.RemoveScavenge(ctx,
			item.Index,
		)
		_, found := keeper.GetScavenge(ctx,
			item.Index,
		)
		require.False(t, found)
	}
}

func TestScavengeGetAll(t *testing.T) {
	keeper, ctx := keepertest.ScavengeKeeper(t)
	items := createNScavenge(keeper, ctx, 10)
	require.ElementsMatch(t, items, keeper.GetAllScavenge(ctx))
}
