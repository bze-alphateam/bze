package tradebin_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	keepertest "github.com/bze-alphateam/bze/testutil/keeper"
	tradebin "github.com/bze-alphateam/bze/x/tradebin/module"
	"github.com/bze-alphateam/bze/x/tradebin/testutil"
	"github.com/bze-alphateam/bze/x/tradebin/types"
	v2types "github.com/bze-alphateam/bze/x/tradebin/v2types"
)

// The owner-price index is not in the genesis: init rebuilds it from order_list (pointing at the
// lowest id when an owner has several orders at one price) and an export/import round trip rebuilds
// the same index.
func TestGenesis_RebuildsOwnerPriceIndex(t *testing.T) {
	const marketId = "stake/ubze"
	ownerA := "bze1m33n82r5x3eyjmjtwjkl82zzdlrnv8pevd8u9r"
	ownerB := "bze1w6x4jhpgdxyxkc4rj2ftk0xm8hn6g5c2ld2xya"
	orders := []types.Order{
		{Id: "000000000000000000000005", MarketId: marketId, OrderType: types.OrderTypeSell, Amount: "10", Price: "1", Owner: ownerA},
		{Id: "000000000000000000000002", MarketId: marketId, OrderType: types.OrderTypeSell, Amount: "10", Price: "1.000", Owner: ownerA},
		{Id: "000000000000000000000003", MarketId: marketId, OrderType: types.OrderTypeSell, Amount: "10", Price: "1", Owner: ownerB},
	}
	genesisState := types.GenesisState{
		Params:              v2types.DefaultParams(),
		OrderList:           orders,
		AggregatedOrderList: []types.AggregatedOrder{{MarketId: marketId, OrderType: types.OrderTypeSell, Amount: "30", Price: "1"}},
		OrderCounter:        6,
	}
	want := []types.OrderReference{
		{Id: "000000000000000000000002", MarketId: marketId, OrderType: types.OrderTypeSell},
		{Id: "000000000000000000000003", MarketId: marketId, OrderType: types.OrderTypeSell},
	}

	acc := testutil.NewMockAccountKeeper(gomock.NewController(t))
	k, ctx := keepertest.TradebinKeeper(t, nil, acc)
	tradebin.InitGenesis(ctx, &k, genesisState)

	ref, found := k.GetOwnerPriceOrder(ctx, ownerA, marketId, types.OrderTypeSell, "1")
	require.True(t, found)
	require.Equal(t, want[0], ref)
	ref, found = k.GetOwnerPriceOrder(ctx, ownerB, marketId, types.OrderTypeSell, "1")
	require.True(t, found)
	require.Equal(t, want[1], ref)
	require.Len(t, k.GetAllOwnerPriceOrder(ctx), 2)

	exported := tradebin.ExportGenesis(ctx, &k)
	require.ElementsMatch(t, orders, exported.OrderList)

	imported, importedCtx := keepertest.TradebinKeeper(t, nil, acc)
	tradebin.InitGenesis(importedCtx, &imported, *exported)
	require.Equal(t, k.GetAllOwnerPriceOrder(ctx), imported.GetAllOwnerPriceOrder(importedCtx))
}
