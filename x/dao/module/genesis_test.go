package dao_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	keepertest "github.com/bze-alphateam/bze/testutil/keeper"
	dao "github.com/bze-alphateam/bze/x/dao/module"
	"github.com/bze-alphateam/bze/x/dao/types"
)

func TestGenesis_RoundTrip(t *testing.T) {
	genesisState := types.GenesisState{
		Params: types.DefaultParams(),
	}

	k, ctx := keepertest.DaoKeeper(t, keepertest.DaoCodec(), nil, nil, nil, nil)
	dao.InitGenesis(ctx, &k, genesisState)
	got := dao.ExportGenesis(ctx, &k)

	require.NotNil(t, got)
	require.Equal(t, genesisState, *got)
}
