package tradebin_test

import (
	"github.com/bze-alphateam/bze/x/tradebin/testutil"
	"go.uber.org/mock/gomock"
	"testing"

	keepertest "github.com/bze-alphateam/bze/testutil/keeper"
	"github.com/bze-alphateam/bze/testutil/nullify"
	tradebin "github.com/bze-alphateam/bze/x/tradebin/module"
	"github.com/bze-alphateam/bze/x/tradebin/types"
	v2types "github.com/bze-alphateam/bze/x/tradebin/v2types"
	"github.com/stretchr/testify/require"
)

func TestGenesis(t *testing.T) {
	genesisState := types.GenesisState{
		Params: v2types.DefaultParams(),

		// this line is used by starport scaffolding # genesis/test/state
	}

	ctrl := gomock.NewController(t)
	acc := testutil.NewMockAccountKeeper(ctrl)

	k, ctx := keepertest.TradebinKeeper(t, nil, acc)
	tradebin.InitGenesis(ctx, &k, genesisState)
	got := tradebin.ExportGenesis(ctx, &k)
	require.NotNil(t, got)

	nullify.Fill(&genesisState)
	nullify.Fill(got)

	// this line is used by starport scaffolding # genesis/test/assert
}
