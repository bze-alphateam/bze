package keeper_test

import (
	"testing"

	testkeeper "github.com/bze-alphateam/bze/testutil/keeper"
	"github.com/bze-alphateam/bze/x/tradebin/types"
	"github.com/stretchr/testify/require"
)

func TestGetParams(t *testing.T) {
	k, ctx := testkeeper.TradebinKeeper(t)
	params := types.DefaultParams()

	k.SetParams(ctx, params)

	require.EqualValues(t, params, k.GetParams(ctx))
	require.EqualValues(t, params.CreateMarketFee, k.CreateMarketFee(ctx))
	require.EqualValues(t, params.MarketMakerFee, k.MarketMakerFee(ctx))
	require.EqualValues(t, params.MarketTakerFee, k.MarketTakerFee(ctx))
	require.EqualValues(t, params.MakerFeeDestination, k.MakerFeeDestination(ctx))
	require.EqualValues(t, params.TakerFeeDestination, k.TakerFeeDestination(ctx))
}
