package keeper_test

import (
	"strconv"
	"testing"

	keepertest "github.com/bze-alphateam/bze/testutil/keeper"
	"github.com/bze-alphateam/bze/testutil/nullify"
	"github.com/bze-alphateam/bze/x/tradebin/keeper"
	"github.com/bze-alphateam/bze/x/tradebin/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

// Prevent strconv unused error
var _ = strconv.IntSize

func createNMarket(keeper *keeper.Keeper, ctx sdk.Context, n int) []types.Market {
	items := make([]types.Market, n)
	for i := range items {
		items[i].Asset1 = strconv.Itoa(i)
		items[i].Asset2 = strconv.Itoa(i)

		keeper.SetMarket(ctx, items[i])
	}
	return items
}

func TestMarketGet(t *testing.T) {
	keeper, ctx := keepertest.TradebinKeeper(t)
	items := createNMarket(keeper, ctx, 10)
	for _, item := range items {
		rst, found := keeper.GetMarket(ctx,
			item.Asset1,
			item.Asset2,
		)
		require.True(t, found)
		require.Equal(t,
			nullify.Fill(&item),
			nullify.Fill(&rst),
		)
	}
}
func TestMarketRemove(t *testing.T) {
	keeper, ctx := keepertest.TradebinKeeper(t)
	items := createNMarket(keeper, ctx, 10)
	for _, item := range items {
		keeper.RemoveMarket(ctx,
			item.Asset1,
			item.Asset2,
		)
		_, found := keeper.GetMarket(ctx,
			item.Asset1,
			item.Asset2,
		)
		require.False(t, found)
	}
}

func TestMarketGetAll(t *testing.T) {
	keeper, ctx := keepertest.TradebinKeeper(t)
	items := createNMarket(keeper, ctx, 10)
	require.ElementsMatch(t,
		nullify.Fill(items),
		nullify.Fill(keeper.GetAllMarket(ctx)),
	)
}
