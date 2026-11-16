package guardian_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/bze-alphateam/bze/testutil/keeper"
	guardian "github.com/bze-alphateam/bze/x/guardian/module"
	"github.com/bze-alphateam/bze/x/guardian/types"
)

func TestGenesis_ExportThenImportEquals(t *testing.T) {
	k, ctx := keepertest.GuardianKeeper(t, nil)

	genesis := types.GenesisState{Params: types.NewParams(sdk.AccAddress("guardian_address____").String())}
	guardian.InitGenesis(ctx, &k, genesis)

	exported := guardian.ExportGenesis(ctx, &k)
	require.Equal(t, genesis, *exported)

	other, otherCtx := keepertest.GuardianKeeper(t, nil)
	guardian.InitGenesis(otherCtx, &other, *exported)
	require.Equal(t, exported, guardian.ExportGenesis(otherCtx, &other))
}

func TestGenesis_DefaultRoundTrip(t *testing.T) {
	k, ctx := keepertest.GuardianKeeper(t, nil)

	guardian.InitGenesis(ctx, &k, *types.DefaultGenesis())
	require.Equal(t, types.DefaultGenesis(), guardian.ExportGenesis(ctx, &k))
}

func TestGenesis_InvalidGuardianPanics(t *testing.T) {
	k, ctx := keepertest.GuardianKeeper(t, nil)

	require.Panics(t, func() {
		guardian.InitGenesis(ctx, &k, types.GenesisState{Params: types.NewParams("nope")})
	})
}
