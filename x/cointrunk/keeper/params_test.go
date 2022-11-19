package keeper_test

import (
	"testing"

	testkeeper "github.com/bze-alphateam/bze/testutil/keeper"
	"github.com/bze-alphateam/bze/x/cointrunk/types"
	"github.com/stretchr/testify/require"
)

func TestGetParams(t *testing.T) {
	k, ctx := testkeeper.CointrunkKeeper(t)
	params := types.DefaultParams()

	k.SetParams(ctx, params)

	require.EqualValues(t, params, k.GetParams(ctx))
	require.EqualValues(t, params.AnonArticleLimit, k.AnonArticleLimit(ctx))
	require.EqualValues(t, params.AnonArticleCost, k.AnonArticleCost(ctx))
}
