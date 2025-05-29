package burner_test

import (
	"github.com/bze-alphateam/bze/x/burner/testutil"
	"go.uber.org/mock/gomock"
	"testing"

	keepertest "github.com/bze-alphateam/bze/testutil/keeper"
	"github.com/bze-alphateam/bze/testutil/nullify"
	burner "github.com/bze-alphateam/bze/x/burner/module"
	"github.com/bze-alphateam/bze/x/burner/types"
	"github.com/stretchr/testify/require"
)

func TestGenesis(t *testing.T) {
	genesisState := types.GenesisState{
		Params: types.DefaultParams(),

		// this line is used by starport scaffolding # genesis/test/state
	}

	ctrl := gomock.NewController(t)
	acc := testutil.NewMockAccountKeeper(ctrl)
	k, ctx := keepertest.BurnerKeeper(t, nil, acc, nil)

	acc.EXPECT().GetModuleAccount(gomock.Any(), gomock.AnyOf(types.ModuleName, types.RaffleModuleName)).Return(nil).Times(2)

	burner.InitGenesis(ctx, k, genesisState)
	got := burner.ExportGenesis(ctx, k)
	require.NotNil(t, got)

	nullify.Fill(&genesisState)
	nullify.Fill(got)

	// this line is used by starport scaffolding # genesis/test/assert
}
