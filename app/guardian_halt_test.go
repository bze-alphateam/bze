package app_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	guardiantypes "github.com/bze-alphateam/bze/x/guardian/types"
	tradebintypes "github.com/bze-alphateam/bze/x/tradebin/types"
)

const guardianHaltDenom = "factory/bze1guardianhalt/utoken"

// The app wires the tradebin keeper into x/guardian: the guardian halts a
// denom through the real message router and tradebin reports it halted;
// tradebin's MsgUnhaltDenoms from governance lifts it.
func TestGuardian_HaltTradingDenomsReachesTradebin(t *testing.T) {
	bApp, ctx := newRateLimitApp(t)
	require.NoError(t, bApp.GuardianKeeper.SetParams(ctx, guardiantypes.NewParams(guardianSigner)))
	require.False(t, bApp.TradebinKeeper.IsDenomHalted(ctx, guardianHaltDenom))

	require.NoError(t, handle(bApp, ctx, &guardiantypes.MsgHaltTradingDenoms{
		Signer: guardianSigner,
		Denoms: []string{guardianHaltDenom},
	}))
	require.True(t, bApp.TradebinKeeper.IsDenomHalted(ctx, guardianHaltDenom))
	require.Equal(t, []string{guardianHaltDenom}, bApp.TradebinKeeper.GetAllHaltedDenoms(ctx))

	// the guardian cannot lift it: tradebin's unhalt is governance only
	require.Error(t, handle(bApp, ctx, &tradebintypes.MsgUnhaltDenoms{Authority: guardianSigner, Denoms: []string{guardianHaltDenom}}))
	require.True(t, bApp.TradebinKeeper.IsDenomHalted(ctx, guardianHaltDenom))

	require.NoError(t, handle(bApp, ctx, &tradebintypes.MsgUnhaltDenoms{Authority: govAuthority(), Denoms: []string{guardianHaltDenom}}))
	require.False(t, bApp.TradebinKeeper.IsDenomHalted(ctx, guardianHaltDenom))
}

// The native denom read from tradebin's params is refused through the app.
func TestGuardian_HaltTradingDenomsRefusesNative(t *testing.T) {
	bApp, ctx := newRateLimitApp(t)
	require.NoError(t, bApp.GuardianKeeper.SetParams(ctx, guardiantypes.NewParams(guardianSigner)))
	native := bApp.TradebinKeeper.GetParams(ctx).NativeDenom
	require.NotEmpty(t, native)

	err := handle(bApp, ctx, &guardiantypes.MsgHaltTradingDenoms{Signer: guardianSigner, Denoms: []string{native}})
	require.ErrorIs(t, err, guardiantypes.ErrInvalidDenom)
	require.False(t, bApp.TradebinKeeper.IsDenomHalted(ctx, native))
}
