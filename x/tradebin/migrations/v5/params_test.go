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

		// no SetParams: an unexpected call fails the test
		pk.EXPECT().GetParams(gomock.Any()).Return(params).Times(1)

		require.NoError(t, v5.MigrateParams(mt.ctx, pk))
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
