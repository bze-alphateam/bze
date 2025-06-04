package tokenfactory_test

import (
	"github.com/bze-alphateam/bze/x/tokenfactory/testutil"
	"go.uber.org/mock/gomock"
	"testing"

	keepertest "github.com/bze-alphateam/bze/testutil/keeper"
	"github.com/bze-alphateam/bze/testutil/nullify"
	tokenfactory "github.com/bze-alphateam/bze/x/tokenfactory/module"
	"github.com/bze-alphateam/bze/x/tokenfactory/types"
	"github.com/stretchr/testify/require"
)

func TestGenesis(t *testing.T) {
	genesisState := types.GenesisState{
		Params: types.DefaultParams(),

		// this line is used by starport scaffolding # genesis/test/state
	}

	ctrl := gomock.NewController(t)
	acc := testutil.NewMockAccountKeeper(ctrl)

	k, ctx := keepertest.TokenfactoryKeeper(t, nil, nil, acc)
	tokenfactory.InitGenesis(ctx, k, genesisState)
	got := tokenfactory.ExportGenesis(ctx, k)
	require.NotNil(t, got)

	nullify.Fill(&genesisState)
	nullify.Fill(got)

	// this line is used by starport scaffolding # genesis/test/assert
}
