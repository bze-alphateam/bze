package tradebin_test

import (
	"cosmossdk.io/math"
	"github.com/bze-alphateam/bze/x/tradebin/testutil"
	"go.uber.org/mock/gomock"
	"testing"

	keepertest "github.com/bze-alphateam/bze/testutil/keeper"
	"github.com/bze-alphateam/bze/testutil/nullify"
	tradebin "github.com/bze-alphateam/bze/x/tradebin/module"
	"github.com/bze-alphateam/bze/x/tradebin/types"
	v2types "github.com/bze-alphateam/bze/x/tradebin/v2types"
	"github.com/stretchr/testify/require"
)

func TestGenesis(t *testing.T) {
	genesisState := types.GenesisState{
		Params: v2types.DefaultParams(),

		// this line is used by starport scaffolding # genesis/test/state
	}

	ctrl := gomock.NewController(t)
	acc := testutil.NewMockAccountKeeper(ctrl)

	k, ctx := keepertest.TradebinKeeper(t, nil, acc)
	tradebin.InitGenesis(ctx, &k, genesisState)
	got := tradebin.ExportGenesis(ctx, &k)
	require.NotNil(t, got)

	nullify.Fill(&genesisState)
	nullify.Fill(got)

	// this line is used by starport scaffolding # genesis/test/assert
}

// Records of the pool price average are not exported; importing the pools seeds them.
func TestGenesisSeedsPoolPriceAverages(t *testing.T) {
	ctrl := gomock.NewController(t)
	acc := testutil.NewMockAccountKeeper(ctrl)
	k, ctx := keepertest.TradebinKeeper(t, nil, acc)
	ctx = ctx.WithBlockHeight(9)

	genesisState := types.GenesisState{
		Params: v2types.DefaultParams(),
		LiquidityPools: []types.LiquidityPool{{
			Id: "abc/xyz", Base: "abc", Quote: "xyz", Fee: math.LegacyZeroDec(), FeeDest: &types.FeeDestination{},
			ReserveBase: math.NewInt(100), ReserveQuote: math.NewInt(300),
		}},
	}
	tradebin.InitGenesis(ctx, &k, genesisState)

	record, found := k.GetPoolPriceAverageRecord(ctx, "abc/xyz")
	require.True(t, found)
	require.Equal(t, math.LegacyMustNewDecFromStr("3"), record.AvgPrice)
	require.Equal(t, int64(9), record.LastHeight)

	exported := tradebin.ExportGenesis(ctx, &k)
	require.Len(t, exported.LiquidityPools, 1)
}
