package app_test

import (
	"testing"

	upgradetypes "cosmossdk.io/x/upgrade/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
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
	// on mainnet the address holds the old ModuleAccount, which stays in auth state
	bApp.AccountKeeper.SetAccount(ctx, bApp.AccountKeeper.NewAccount(ctx, authtypes.NewEmptyModuleAccount("txfeecollector_burner")))

	left := sdk.NewCoins(
		sdk.NewInt64Coin("ubze", 1_234),
		sdk.NewInt64Coin("factory/bze1creator/token", 56),
		sdk.NewInt64Coin("ibc/6490A7EAB61059BFC1CDDEB05917DD70BDF3A611654162A1A47DB930D40D8AF4", 7),
		sdk.NewInt64Coin(tradebintypes.GetLpDenom("factory/bze1creator/token_ubze"), 89),
	)
	require.NoError(t, bApp.BankKeeper.MintCoins(ctx, minttypes.ModuleName, left))
	// pre-upgrade state: the address now stays blocked, so seed it with a plain send (no blocked-recipient check)
	require.NoError(t, bApp.BankKeeper.SendCoins(ctx, authtypes.NewModuleAddress(minttypes.ModuleName), old, left))
	burnerBefore := bApp.BankKeeper.GetAllBalances(ctx, burner)
	supplyBefore := supplyOf(bApp, ctx, left)

	_, err := v830.CreateUpgradeHandler(bApp.Configurator(), bApp.ModuleManager, bApp.GuardianKeeper, bApp.RatelimitKeeper, bApp.BankKeeper, bApp.TradebinKeeper, bApp.BankKeeper)(
		ctx, upgradetypes.Plan{Name: v830.UpgradeName}, bApp.ModuleManager.GetVersionMap(),
	)
	require.NoError(t, err)

	require.True(t, bApp.BankKeeper.GetAllBalances(ctx, old).IsZero(), "the old collector is empty")
	require.Equal(t, burnerBefore.Add(left...).String(), bApp.BankKeeper.GetAllBalances(ctx, burner).String(), "the burner holds every swept coin")
	require.Equal(t, supplyBefore.String(), supplyOf(bApp, ctx, left).String(), "nothing minted or burned")
}

// The app no longer registers txfeecollector_burner as a module account (no permission), but its retired address
// stays blocked so nothing sent there is stranded.
func TestAppWiring_NoBurnerFeeCollector(t *testing.T) {
	bApp, _ := newRateLimitApp(t)
	require.NotContains(t, app.GetMaccPerms(), "txfeecollector_burner")
	require.True(t, app.BlockedAddresses()["txfeecollector_burner"], "the retired address stays blocked")
	require.True(t, bApp.BankKeeper.BlockedAddr(authtypes.NewModuleAddress("txfeecollector_burner")), "the bank keeper blocks it")
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

// After the upgrade a plain bank send to the retired txfeecollector_burner address is rejected by the real bank msg
// server, and nothing moves.
func TestBankSend_RetiredBurnerFeeCollectorRejected(t *testing.T) {
	bApp, ctx := newRateLimitApp(t)
	old := authtypes.NewModuleAddress("txfeecollector_burner")
	sender := sdk.AccAddress("sender______________")
	coins := sdk.NewCoins(sdk.NewInt64Coin("ubze", 1_000))
	require.NoError(t, bApp.BankKeeper.MintCoins(ctx, minttypes.ModuleName, coins))
	require.NoError(t, bApp.BankKeeper.SendCoinsFromModuleToAccount(ctx, minttypes.ModuleName, sender, coins))

	baseKeeper, ok := bApp.BankKeeper.(bankkeeper.BaseKeeper)
	require.True(t, ok)
	_, err := bankkeeper.NewMsgServerImpl(baseKeeper).Send(ctx, banktypes.NewMsgSend(sender, old, coins))
	require.ErrorIs(t, err, sdkerrors.ErrUnauthorized)
	require.ErrorContains(t, err, "is not allowed to receive funds")

	require.True(t, bApp.BankKeeper.GetAllBalances(ctx, old).IsZero(), "nothing reached the retired address")
	require.Equal(t, coins.String(), bApp.BankKeeper.GetAllBalances(ctx, sender).String(), "the sender keeps the coins")
}
