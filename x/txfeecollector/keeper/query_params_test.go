package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	keepertest "github.com/bze-alphateam/bze/testutil/keeper"
	"github.com/bze-alphateam/bze/x/txfeecollector/types"
)

func TestParamsQuery(t *testing.T) {
	keeper, ctx := keepertest.TxfeecollectorKeeper(t)
	params := types.DefaultParams()
	require.NoError(t, keeper.SetParams(ctx, params))

	response, err := keeper.Params(ctx, &types.QueryParamsRequest{})
	require.NoError(t, err)
	require.Equal(t, &types.QueryParamsResponse{Params: params}, response)
}

func TestParamsQuery_NilRequest(t *testing.T) {
	keeper, ctx := keepertest.TxfeecollectorKeeper(t)

	response, err := keeper.Params(ctx, nil)
	require.Error(t, err)
	require.Nil(t, response)
	require.Contains(t, err.Error(), "invalid request")
}
