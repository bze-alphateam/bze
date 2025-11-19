package epochs_test

import (
	"testing"

	keepertest "github.com/bze-alphateam/bze/testutil/keeper"
	"github.com/bze-alphateam/bze/testutil/nullify"
	epoch "github.com/bze-alphateam/bze/x/epochs/module"
	"github.com/bze-alphateam/bze/x/epochs/types"
	"github.com/stretchr/testify/require"
)

func TestGenesis(t *testing.T) {
	genesisState := types.GenesisState{}

	k, ctx := keepertest.EpochKeeper(t)
	epoch.InitGenesis(ctx, k, genesisState)
	got := epoch.ExportGenesis(ctx, k)
	require.NotNil(t, got)

	nullify.Fill(&genesisState)
	nullify.Fill(got)

	// this line is used by starport scaffolding # genesis/test/assert
}
