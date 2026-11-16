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
	k, ctx := keepertest.BurnerKeeper(t, nil, acc, nil, nil, nil)

	burner.InitGenesis(ctx, k, genesisState)
	got := burner.ExportGenesis(ctx, k)
	require.NotNil(t, got)

	nullify.Fill(&genesisState)
	nullify.Fill(got)

	// this line is used by starport scaffolding # genesis/test/assert
}

// Every MsgMoveIbcLockedCoins lock survives an export and an import.
func TestGenesis_IbcMoveLocksRoundTrip(t *testing.T) {
	genesisState := types.GenesisState{
		Params: types.DefaultParams(),
		IbcMoveLocks: []types.IbcMoveLock{
			{Denom: "ibc/AAA", LastMoveHeight: 10},
			{Denom: "ibc/BBB", LastMoveHeight: 2000},
		},
	}
	require.NoError(t, genesisState.Validate())

	ctrl := gomock.NewController(t)
	k, ctx := keepertest.BurnerKeeper(t, nil, testutil.NewMockAccountKeeper(ctrl), nil, nil, nil)
	burner.InitGenesis(ctx, k, genesisState)

	lock, found := k.GetIbcMoveLock(ctx, "ibc/BBB")
	require.True(t, found)
	require.Equal(t, int64(2000), lock.LastMoveHeight)

	got := burner.ExportGenesis(ctx, k)
	require.Equal(t, genesisState.IbcMoveLocks, got.IbcMoveLocks)
	require.True(t, got.Params.Equal(genesisState.Params))

	k2, ctx2 := keepertest.BurnerKeeper(t, nil, testutil.NewMockAccountKeeper(ctrl), nil, nil, nil)
	burner.InitGenesis(ctx2, k2, *got)
	require.Equal(t, genesisState.IbcMoveLocks, burner.ExportGenesis(ctx2, k2).IbcMoveLocks)
}
