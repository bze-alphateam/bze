package app_test

import (
	"testing"

	"cosmossdk.io/math"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	ratelimittypes "github.com/cosmos/ibc-apps/modules/rate-limiting/v10/types"
	"github.com/stretchr/testify/require"

	"github.com/bze-alphateam/bze/app"
	v830 "github.com/bze-alphateam/bze/app/upgrades/v830"
	guardiantypes "github.com/bze-alphateam/bze/x/guardian/types"
)

var guardianSigner = sdk.AccAddress("guardian_address____").String()

func addLimit(t *testing.T, send, recv int64) (*app.App, sdk.Context) {
	t.Helper()

	bApp, ctx := newRateLimitApp(t)
	require.NoError(t, handle(bApp, ctx, &ratelimittypes.MsgAddRateLimit{
		Authority:         govAuthority(),
		Denom:             rlDenom,
		ChannelOrClientId: rlChannel,
		MaxPercentSend:    math.NewInt(send),
		MaxPercentRecv:    math.NewInt(recv),
		DurationHours:     24,
	}))
	require.NoError(t, bApp.GuardianKeeper.SetParams(ctx, guardiantypes.NewParams(guardianSigner)))

	return bApp, ctx
}

// The v8.3.0 handler gives x/guardian the configured guardian, and the
// address is a valid account address under the chain's prefix.
func TestGuardian_UpgradeHandlerSetsTheGuardian(t *testing.T) {
	bApp, ctx := newRateLimitApp(t)
	require.Empty(t, bApp.GuardianKeeper.GetParams(ctx).Guardian)

	vm := bApp.ModuleManager.GetVersionMap()
	delete(vm, guardiantypes.ModuleName)

	newVM, err := v830.CreateUpgradeHandler(bApp.Configurator(), bApp.ModuleManager, bApp.GuardianKeeper)(
		ctx, upgradetypes.Plan{Name: v830.UpgradeName}, vm,
	)
	require.NoError(t, err)
	require.Equal(t, uint64(1), newVM[guardiantypes.ModuleName])

	params := bApp.GuardianKeeper.GetParams(ctx)
	require.Equal(t, v830.GuardianAddress, params.Guardian)
	require.NoError(t, params.Validate())
}

// The guardian lowers a quota through the real message router and the real
// ratelimit keeper; the flow already counted in the window is kept.
func TestGuardian_TightenKeepsTheWindowFlow(t *testing.T) {
	a, ctx := addLimit(t, 10, 20)

	limit, found := a.RatelimitKeeper.GetRateLimit(ctx, rlDenom, rlChannel)
	require.True(t, found)
	limit.Flow.Outflow = math.NewInt(777)
	limit.Flow.Inflow = math.NewInt(55)
	a.RatelimitKeeper.SetRateLimit(ctx, limit)

	require.NoError(t, handle(a, ctx, &guardiantypes.MsgTightenRateLimit{
		Signer: guardianSigner, Denom: rlDenom, ChannelId: rlChannel,
		MaxPercentSend: math.NewInt(3), MaxPercentRecv: math.NewInt(0),
	}))

	got, found := a.RatelimitKeeper.GetRateLimit(ctx, rlDenom, rlChannel)
	require.True(t, found)
	require.Equal(t, math.NewInt(3), got.Quota.MaxPercentSend)
	require.True(t, got.Quota.MaxPercentRecv.IsZero())
	require.Equal(t, uint64(24), got.Quota.DurationHours)
	require.Equal(t, math.NewInt(777), got.Flow.Outflow)
	require.Equal(t, math.NewInt(55), got.Flow.Inflow)
	require.Equal(t, limit.Flow.ChannelValue, got.Flow.ChannelValue)
}

func TestGuardian_RaiseAndStrangerAreRefused(t *testing.T) {
	a, ctx := addLimit(t, 10, 20)

	err := handle(a, ctx, &guardiantypes.MsgTightenRateLimit{
		Signer: guardianSigner, Denom: rlDenom, ChannelId: rlChannel,
		MaxPercentSend: math.NewInt(11), MaxPercentRecv: math.NewInt(20),
	})
	require.ErrorIs(t, err, guardiantypes.ErrQuotaRaise)

	err = handle(a, ctx, &guardiantypes.MsgTightenRateLimit{
		Signer: sdk.AccAddress("stranger_address_____").String(), Denom: rlDenom, ChannelId: rlChannel,
		MaxPercentSend: math.NewInt(1), MaxPercentRecv: math.NewInt(1),
	})
	require.ErrorIs(t, err, guardiantypes.ErrUnauthorized)

	got, _ := a.RatelimitKeeper.GetRateLimit(ctx, rlDenom, rlChannel)
	require.Equal(t, math.NewInt(10), got.Quota.MaxPercentSend)
	require.Equal(t, math.NewInt(20), got.Quota.MaxPercentRecv)
}

// Governance signs too; only governance changes the params.
func TestGuardian_GovernanceTightensAndOnlyGovernanceUpdatesParams(t *testing.T) {
	a, ctx := addLimit(t, 10, 20)

	require.NoError(t, handle(a, ctx, &guardiantypes.MsgTightenRateLimit{
		Signer: govAuthority(), Denom: rlDenom, ChannelId: rlChannel,
		MaxPercentSend: math.NewInt(1), MaxPercentRecv: math.NewInt(1),
	}))

	err := handle(a, ctx, &guardiantypes.MsgUpdateParams{Authority: guardianSigner, Params: guardiantypes.DefaultParams()})
	require.ErrorIs(t, err, guardiantypes.ErrInvalidSigner)

	require.NoError(t, handle(a, ctx, &guardiantypes.MsgUpdateParams{Authority: govAuthority(), Params: guardiantypes.DefaultParams()}))
	require.Empty(t, a.GuardianKeeper.GetParams(ctx).Guardian)
}
