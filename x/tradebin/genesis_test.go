package tradebin_test

import (
	"github.com/bze-alphateam/bze/x/tradebin/testutil"
	"testing"

	keepertest "github.com/bze-alphateam/bze/testutil/keeper"
	"github.com/bze-alphateam/bze/testutil/nullify"
	"github.com/bze-alphateam/bze/x/tradebin"
	"github.com/bze-alphateam/bze/x/tradebin/types"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestGenesis(t *testing.T) {
	genesisState := types.GenesisState{
		Params: types.DefaultParams(),

		MarketList: []types.Market{
			{
				Base:  "0",
				Quote: "0",
			},
			{
				Base:  "1",
				Quote: "1",
			},
		},
		// this line is used by starport scaffolding # genesis/test/state
	}

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockBank := testutil.NewMockBankKeeper(mockCtrl)
	require.NotNil(t, mockBank)

	mockDistr := testutil.NewMockDistrKeeper(mockCtrl)
	require.NotNil(t, mockBank)

	mockAccount := testutil.NewMockAccountKeeper(mockCtrl)
	require.NotNil(t, mockAccount)

	k, ctx := keepertest.TradebinKeeper(t, mockBank, mockDistr, mockAccount)
	tradebin.InitGenesis(ctx, *k, genesisState)
	got := tradebin.ExportGenesis(ctx, *k)
	require.NotNil(t, got)

	nullify.Fill(&genesisState)
	nullify.Fill(got)

	require.ElementsMatch(t, genesisState.MarketList, got.MarketList)
	// this line is used by starport scaffolding # genesis/test/assert
}
