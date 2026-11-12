package app_test

import (
	"context"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	"github.com/stretchr/testify/require"

	"github.com/bze-alphateam/bze/app"
	burnertypes "github.com/bze-alphateam/bze/x/burner/types"
	guardiantypes "github.com/bze-alphateam/bze/x/guardian/types"
	txfeecollectortypes "github.com/bze-alphateam/bze/x/txfeecollector/types"
)

// burnerEndBlock runs the burner module's EndBlock on ctx.
func burnerEndBlock(t *testing.T, bApp *app.App, ctx sdk.Context) {
	t.Helper()

	endBlocker, ok := bApp.ModuleManager.Modules[burnertypes.ModuleName].(interface {
		EndBlock(context.Context) error
	})
	require.True(t, ok)
	require.NoError(t, endBlocker.EndBlock(ctx))
}

// The guardian's pause reaches txfeecollector and the burner through the app wiring: while paused, the coins in the
// fee collector and in the burner stay where they are across EndBlocks; the guardian can't resume, governance can, and
// the next EndBlocks convert and burn them.
func TestGuardian_PauseAutomationFreezesFeeConversionAndPeriodicBurn(t *testing.T) {
	bApp, ctx := newRateLimitApp(t)
	require.NoError(t, bApp.GuardianKeeper.SetParams(ctx, guardiantypes.NewParams(guardianSigner)))
	setMaxBalanceIterations(t, bApp, ctx, 10) // no InitChain here: the default params are not set

	collector := authtypes.NewModuleAddress(txfeecollectortypes.ModuleName)
	burner := authtypes.NewModuleAddress(burnertypes.ModuleName)
	fee := sdk.NewCoins(sdk.NewInt64Coin(rlDenom, 1_000))
	toBurn := sdk.NewCoins(sdk.NewInt64Coin(rlDenom, 2_000))
	require.NoError(t, bApp.BankKeeper.MintCoins(ctx, minttypes.ModuleName, fee.Add(toBurn...)))
	require.NoError(t, bApp.BankKeeper.SendCoinsFromModuleToModule(ctx, minttypes.ModuleName, txfeecollectortypes.ModuleName, fee))
	require.NoError(t, bApp.BankKeeper.SendCoinsFromModuleToModule(ctx, minttypes.ModuleName, burnertypes.ModuleName, toBurn))
	bApp.BurnerKeeper.EnqueuePeriodicBurn(ctx)

	require.NoError(t, handle(bApp, ctx, &guardiantypes.MsgPauseAutomation{Signer: guardianSigner}))
	require.True(t, bApp.TxfeecollectorKeeper.IsAutomationPaused(ctx))
	require.True(t, bApp.BurnerKeeper.IsAutomationPaused(ctx))

	for i := 0; i < 3; i++ {
		txfeecollectorEndBlock(t, bApp, ctx)
		burnerEndBlock(t, bApp, ctx)
	}
	require.Equal(t, fee.String(), bApp.BankKeeper.GetAllBalances(ctx, collector).String(), "the fees wait in the collector")
	require.Equal(t, toBurn.String(), bApp.BankKeeper.GetAllBalances(ctx, burner).String(), "nothing is burned")

	require.ErrorIs(t, handle(bApp, ctx, &guardiantypes.MsgResumeAutomation{Authority: guardianSigner}), guardiantypes.ErrUnauthorized)
	require.NoError(t, handle(bApp, ctx, &guardiantypes.MsgResumeAutomation{Authority: govAuthority()}))
	require.False(t, bApp.TxfeecollectorKeeper.IsAutomationPaused(ctx))
	require.False(t, bApp.BurnerKeeper.IsAutomationPaused(ctx))

	supplyBefore := bApp.BankKeeper.GetSupply(ctx, rlDenom)
	txfeecollectorEndBlock(t, bApp, ctx)
	burnerEndBlock(t, bApp, ctx)
	require.True(t, bApp.BankKeeper.GetAllBalances(ctx, collector).IsZero(), "the fees are forwarded")
	require.True(t, bApp.BankKeeper.GetAllBalances(ctx, burner).IsZero(), "the burner's coins are burned")
	require.Equal(t, supplyBefore.Sub(toBurn[0]).String(), bApp.BankKeeper.GetSupply(ctx, rlDenom).String())
}
