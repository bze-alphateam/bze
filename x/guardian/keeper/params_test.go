package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/bze-alphateam/bze/testutil/keeper"
	"github.com/bze-alphateam/bze/x/guardian/keeper"
	"github.com/bze-alphateam/bze/x/guardian/types"
)

func newBareKeeper(t *testing.T) (keeper.Keeper, sdk.Context) {
	t.Helper()

	return keepertest.GuardianKeeper(t)
}

func TestParams_DefaultHasNoGuardian(t *testing.T) {
	k, ctx := newBareKeeper(t)

	require.Equal(t, types.DefaultParams(), k.GetParams(ctx))
	require.Empty(t, k.GetParams(ctx).Guardian)
}

func TestParams_SetAndGetRoundTrip(t *testing.T) {
	k, ctx := newBareKeeper(t)

	require.NoError(t, k.SetParams(ctx, types.NewParams(guardianAddr)))
	require.Equal(t, types.NewParams(guardianAddr), k.GetParams(ctx))
}

func TestUpdateParams_AuthoritySetsTheGuardian(t *testing.T) {
	e := newEnv(t)

	_, err := e.msgServer.UpdateParams(e.ctx, &types.MsgUpdateParams{Authority: authority, Params: types.NewParams(strangerAddr)})
	require.NoError(t, err)
	require.Equal(t, strangerAddr, e.k.GetParams(e.ctx).Guardian)

	_, err = e.msgServer.UpdateParams(e.ctx, &types.MsgUpdateParams{Authority: authority, Params: types.DefaultParams()})
	require.NoError(t, err)
	require.Empty(t, e.k.GetParams(e.ctx).Guardian)
}

// Only governance changes the params: the guardian cannot name its successor.
func TestUpdateParams_AnyoneElseIsRefused(t *testing.T) {
	e := newEnv(t)

	for _, signer := range []string{guardianAddr, strangerAddr} {
		_, err := e.msgServer.UpdateParams(e.ctx, &types.MsgUpdateParams{Authority: signer, Params: types.NewParams(strangerAddr)})
		require.ErrorIs(t, err, types.ErrInvalidSigner)
	}
	require.Equal(t, guardianAddr, e.k.GetParams(e.ctx).Guardian)
}

func TestUpdateParams_InvalidGuardianIsRefused(t *testing.T) {
	e := newEnv(t)

	_, err := e.msgServer.UpdateParams(e.ctx, &types.MsgUpdateParams{Authority: authority, Params: types.NewParams("not-an-address")})
	require.Error(t, err)
	require.Equal(t, guardianAddr, e.k.GetParams(e.ctx).Guardian)
}

func TestQueryParams(t *testing.T) {
	e := newEnv(t)

	res, err := e.k.Params(e.ctx, &types.QueryParamsRequest{})
	require.NoError(t, err)
	require.Equal(t, types.NewParams(guardianAddr), res.Params)

	_, err = e.k.Params(e.ctx, nil)
	require.Error(t, err)
}
