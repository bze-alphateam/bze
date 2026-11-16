package v5_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	v5 "github.com/bze-alphateam/bze/x/tradebin/migrations/v5"
	tbtestutil "github.com/bze-alphateam/bze/x/tradebin/testutil"
	v2types "github.com/bze-alphateam/bze/x/tradebin/v2types"
)

func TestMigrateParams_ZeroMaxFillsPerOrderGetsTheDefault(t *testing.T) {
	mt := setup(t)
	pk := tbtestutil.NewMockV5ParamsKeeper(gomock.NewController(t))
	stored := v2types.DefaultParams()
	stored.MaxFillsPerOrder = 0
	want := stored
	want.MaxFillsPerOrder = 500

	pk.EXPECT().GetParams(gomock.Any()).Return(stored).Times(1)
	pk.EXPECT().SetParams(gomock.Any(), want).Return(nil).Times(1)

	require.NoError(t, v5.MigrateParams(mt.ctx, pk))
}

func TestMigrateParams_NonZeroMaxFillsPerOrderIsKept(t *testing.T) {
	for _, stored := range []uint64{1, 7, 500} {
		mt := setup(t)
		pk := tbtestutil.NewMockV5ParamsKeeper(gomock.NewController(t))
		params := v2types.DefaultParams()
		params.MaxFillsPerOrder = stored

		pk.EXPECT().GetParams(gomock.Any()).Return(params).Times(1)
		pk.EXPECT().SetParams(gomock.Any(), params).Return(nil).Times(1)

		require.NoError(t, v5.MigrateParams(mt.ctx, pk))
	}
}

// Field 7 was the queue depth window (default 100) before version 5; it becomes the owners window and
// is set to 500 whatever was stored.
func TestMigrateParams_PriceLevelOwnersWindowBecomesTheDefault(t *testing.T) {
	for _, stored := range []uint64{0, 1, 100, 500, 2000} {
		mt := setup(t)
		pk := tbtestutil.NewMockV5ParamsKeeper(gomock.NewController(t))
		params := v2types.DefaultParams()
		params.PriceLevelOwnersWindow = stored
		want := params
		want.PriceLevelOwnersWindow = 500

		pk.EXPECT().GetParams(gomock.Any()).Return(params).Times(1)
		pk.EXPECT().SetParams(gomock.Any(), want).Return(nil).Times(1)

		require.NoError(t, v5.MigrateParams(mt.ctx, pk), "stored %d", stored)
	}
}

// Field 8 keeps a non-zero stored value as the extra gas per owner; a stored 0 gets the default.
func TestMigrateParams_PriceLevelExtraGas(t *testing.T) {
	for _, tc := range []struct{ stored, want uint64 }{{0, 25000}, {1, 1}, {25000, 25000}, {40000, 40000}} {
		mt := setup(t)
		pk := tbtestutil.NewMockV5ParamsKeeper(gomock.NewController(t))
		params := v2types.DefaultParams()
		params.PriceLevelExtraGas = tc.stored
		want := params
		want.PriceLevelExtraGas = tc.want

		pk.EXPECT().GetParams(gomock.Any()).Return(params).Times(1)
		pk.EXPECT().SetParams(gomock.Any(), want).Return(nil).Times(1)

		require.NoError(t, v5.MigrateParams(mt.ctx, pk), "stored %d", tc.stored)
	}
}

func TestMigrateParams_SetParamsError(t *testing.T) {
	mt := setup(t)
	pk := tbtestutil.NewMockV5ParamsKeeper(gomock.NewController(t))
	stored := v2types.DefaultParams()
	stored.MaxFillsPerOrder = 0

	pk.EXPECT().GetParams(gomock.Any()).Return(stored).Times(1)
	pk.EXPECT().SetParams(gomock.Any(), gomock.Any()).Return(errors.New("boom")).Times(1)

	require.Error(t, v5.MigrateParams(mt.ctx, pk))
}
