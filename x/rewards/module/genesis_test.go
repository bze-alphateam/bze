package rewards_test

import (
	"github.com/bze-alphateam/bze/x/rewards/testutil"
	"go.uber.org/mock/gomock"
	"testing"

	keepertest "github.com/bze-alphateam/bze/testutil/keeper"
	"github.com/bze-alphateam/bze/testutil/nullify"
	rewards "github.com/bze-alphateam/bze/x/rewards/module"
	"github.com/bze-alphateam/bze/x/rewards/types"
	"github.com/stretchr/testify/require"
)

func TestGenesis(t *testing.T) {
	genesisState := types.GenesisState{
		Params: types.DefaultParams(),

		// this line is used by starport scaffolding # genesis/test/state
	}
	ctrl := gomock.NewController(t)
	acc := testutil.NewMockAccountKeeper(ctrl)

	k, ctx := keepertest.RewardsKeeper(t, nil, nil, nil, nil, acc)
	rewards.InitGenesis(ctx, k, genesisState)
	got := rewards.ExportGenesis(ctx, k)
	require.NotNil(t, got)

	nullify.Fill(&genesisState)
	nullify.Fill(got)

	// this line is used by starport scaffolding # genesis/test/assert
}
