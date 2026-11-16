package v5_test

import (
	"errors"
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	v5 "github.com/bze-alphateam/bze/x/burner/migrations/v5"
	"github.com/bze-alphateam/bze/x/burner/testutil"
	"github.com/bze-alphateam/bze/x/burner/types"
)

// paramsMatcher compares params by value; LegacyDec fields can't be compared with reflect.DeepEqual.
type paramsMatcher struct{ want types.Params }

func (m paramsMatcher) Matches(x any) bool {
	p, ok := x.(types.Params)
	return ok && p.Equal(m.want)
}

func (m paramsMatcher) String() string { return m.want.String() }

// Params stored before consensus version 5 decode with a nil ratio and a 0 cooldown; both get their defaults and the
// burning period is kept.
func TestMigrateParams_UnsetLimitsGetTheDefaults(t *testing.T) {
	for _, ratio := range []math.LegacyDec{{}, math.LegacyZeroDec()} {
		pk := testutil.NewMockV5ParamsKeeper(gomock.NewController(t))
		stored := types.Params{PeriodicBurningWeeks: 4, IbcMoveMaxReserveRatio: ratio}
		want := types.Params{
			PeriodicBurningWeeks:   4,
			IbcMoveMaxReserveRatio: math.LegacyNewDecWithPrec(3, 2),
			IbcMoveCooldownBlocks:  1000,
		}

		pk.EXPECT().GetParams(gomock.Any()).Return(stored).Times(1)
		pk.EXPECT().SetParams(gomock.Any(), paramsMatcher{want}).Return(nil).Times(1)

		require.NoError(t, v5.MigrateParams(sdk.Context{}, pk))
	}
}

// Limits that are already set are kept.
func TestMigrateParams_SetLimitsAreKept(t *testing.T) {
	pk := testutil.NewMockV5ParamsKeeper(gomock.NewController(t))
	stored := types.Params{
		PeriodicBurningWeeks:   2,
		IbcMoveMaxReserveRatio: math.LegacyNewDecWithPrec(1, 1),
		IbcMoveCooldownBlocks:  7,
	}

	pk.EXPECT().GetParams(gomock.Any()).Return(stored).Times(1)
	pk.EXPECT().SetParams(gomock.Any(), paramsMatcher{stored}).Return(nil).Times(1)

	require.NoError(t, v5.MigrateParams(sdk.Context{}, pk))
}

func TestMigrateParams_SetParamsError(t *testing.T) {
	pk := testutil.NewMockV5ParamsKeeper(gomock.NewController(t))
	pk.EXPECT().GetParams(gomock.Any()).Return(types.Params{PeriodicBurningWeeks: 1}).Times(1)
	pk.EXPECT().SetParams(gomock.Any(), gomock.Any()).Return(errors.New("boom")).Times(1)

	require.ErrorContains(t, v5.MigrateParams(sdk.Context{}, pk), "boom")
}
