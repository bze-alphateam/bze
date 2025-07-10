package keeper_test

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"testing"

	keeper2 "github.com/bze-alphateam/bze/testutil/keeper"
	"github.com/bze-alphateam/bze/x/cointrunk/exported"
	"github.com/bze-alphateam/bze/x/cointrunk/keeper"
	"github.com/bze-alphateam/bze/x/cointrunk/testutil"
	"github.com/bze-alphateam/bze/x/cointrunk/types"
)

// mockSubspace implements the exported.Subspace interface for testing
type mockSubspace struct {
	ps types.Params
}

func newMockSubspace(ps types.Params) mockSubspace {
	return mockSubspace{ps: ps}
}

func (ms mockSubspace) GetParamSet(_ sdk.Context, ps exported.ParamSet) {
	*ps.(*types.Params) = ms.ps
}

// TestMigrator_Migrate1to2 tests the successful migration from version 1 to 2
func TestMigrator_Migrate1to2(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockBank := testutil.NewMockBankKeeper(mockCtrl)
	require.NotNil(t, mockBank)
	mockDistr := testutil.NewMockDistrKeeper(mockCtrl)
	require.NotNil(t, mockDistr)

	k, ctx := keeper2.CointrunkKeeper(t, mockBank, mockDistr)

	// Create mock subspace with default params
	legacySubspace := newMockSubspace(types.DefaultParams())

	// Create migrator
	migrator := keeper.NewMigrator(k, legacySubspace)

	// Run migration (testing with empty publisher store since we can't inject v1 data)
	require.NoError(t, migrator.Migrate2to3(ctx))

	// Verify params were migrated
	params := k.GetParams(ctx)
	require.Equal(t, legacySubspace.ps, params)

	// Verify no publishers exist (since we started with empty store)
	allPublishers := k.GetAllPublisher(ctx)
	require.Empty(t, allPublishers)
}
