package app_test

import (
	"testing"

	upgradetypes "cosmossdk.io/x/upgrade/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	"github.com/stretchr/testify/require"

	"github.com/bze-alphateam/bze/app"
	v830 "github.com/bze-alphateam/bze/app/upgrades/v830"
	burnertypes "github.com/bze-alphateam/bze/x/burner/types"
	tradebintypes "github.com/bze-alphateam/bze/x/tradebin/types"
)

// The v8.3.0 upgrade moves every coin left at the removed txfeecollector_burner address to the burner module
// with the real bank keeper: native, factory, IBC and LP-share denoms alike. The old address ends empty and the
// supply does not move.
func TestUpgrade_SweepsBurnerFeeCollectorToBurner(t *testing.T) {
	bApp, ctx := newRateLimitApp(t)
	old := authtypes.NewModuleAddress("txfeecollector_burner")
	burner := authtypes.NewModuleAddress(burnertypes.ModuleName)

	left := sdk.NewCoins(
		sdk.NewInt64Coin("ubze", 1_234),
		sdk.NewInt64Coin("factory/bze1creator/token", 56),
		sdk.NewInt64Coin("ibc/6490A7EAB61059BFC1CDDEB05917DD70BDF3A611654162A1A47DB930D40D8AF4", 7),
		sdk.NewInt64Coin(tradebintypes.GetLpDenom("factory/bze1creator/token_ubze"), 89),
	)
	require.NoError(t, bApp.BankKeeper.MintCoins(ctx, minttypes.ModuleName, left))
	require.NoError(t, bApp.BankKeeper.SendCoinsFromModuleToAccount(ctx, minttypes.ModuleName, old, left))
	burnerBefore := bApp.BankKeeper.GetAllBalances(ctx, burner)
	supplyBefore := supplyOf(bApp, ctx, left)

	_, err := v830.CreateUpgradeHandler(bApp.Configurator(), bApp.ModuleManager, bApp.GuardianKeeper, bApp.RatelimitKeeper, bApp.BankKeeper)(
		ctx, upgradetypes.Plan{Name: v830.UpgradeName}, bApp.ModuleManager.GetVersionMap(),
	)
	require.NoError(t, err)

	require.True(t, bApp.BankKeeper.GetAllBalances(ctx, old).IsZero(), "the old collector is empty")
	require.Equal(t, burnerBefore.Add(left...).String(), bApp.BankKeeper.GetAllBalances(ctx, burner).String(), "the burner holds every swept coin")
	require.Equal(t, supplyBefore.String(), supplyOf(bApp, ctx, left).String(), "nothing minted or burned")
}

// The app no longer registers txfeecollector_burner: no module account permission and no blocked address.
func TestAppWiring_NoBurnerFeeCollector(t *testing.T) {
	bApp, _ := newRateLimitApp(t)
	require.NotContains(t, app.GetMaccPerms(), "txfeecollector_burner")
	require.NotContains(t, app.BlockedAddresses(), "txfeecollector_burner")
	require.Nil(t, bApp.AccountKeeper.GetModuleAddress("txfeecollector_burner"))
	// the collectors that stay are still registered
	require.Contains(t, app.GetMaccPerms(), "txfeecollector")
	require.Contains(t, app.GetMaccPerms(), "txfeecollector_cp")
}

func supplyOf(bApp *app.App, ctx sdk.Context, coins sdk.Coins) sdk.Coins {
	supply := sdk.NewCoins()
	for _, c := range coins {
		supply = supply.Add(bApp.BankKeeper.GetSupply(ctx, c.Denom))
	}
	return supply
}
