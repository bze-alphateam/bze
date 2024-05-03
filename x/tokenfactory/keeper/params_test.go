package keeper_test

import (
	"testing"

	testkeeper "github.com/bze-alphateam/bze/testutil/keeper"
	"github.com/bze-alphateam/bze/x/tokenfactory/types"
	"github.com/stretchr/testify/require"
)

func TestGetParams(t *testing.T) {
	k, ctx := testkeeper.TokenfactoryKeeper(t)
	params := types.DefaultParams()

	k.SetParams(ctx, params)

	require.EqualValues(t, params, k.GetParams(ctx))
	require.EqualValues(t, params.CreateDenomFee, k.CreateDenomFee(ctx))
}
