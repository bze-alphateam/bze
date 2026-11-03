package app_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	"github.com/stretchr/testify/require"

	"github.com/bze-alphateam/bze/app"
	txfeecollectortypes "github.com/bze-alphateam/bze/x/txfeecollector/types"
)

// The app's bank keeper for txfeecollector reads at most limit balances, strictly after the given denom, in denom
// order, from the real bank store.
func TestTxfeecollectorBankKeeper_GetBalancesPage(t *testing.T) {
	bApp, ctx := newZeroOutputApp(t)
	baseKeeper, ok := bApp.BankKeeper.(bankkeeper.BaseKeeper)
	require.True(t, ok)
	bank := app.ProvideTxfeecollectorBankKeeper(baseKeeper)

	module := txfeecollectortypes.CpFeeCollector
	addr := authtypes.NewModuleAddress(module)
	coins := sdk.NewCoins(
		sdk.NewInt64Coin("ibc/A", 1), sdk.NewInt64Coin("ibc/B", 2), sdk.NewInt64Coin("ibc/D", 4),
		sdk.NewInt64Coin("ibc/E", 5), sdk.NewInt64Coin("ubze", 6),
	)

	empty, last := bank.GetBalancesPage(ctx, addr, "", 10)
	require.Empty(t, empty, "empty account")
	require.Empty(t, last)

	mintToModule(t, bApp, ctx, module, coins)

	page, last := bank.GetBalancesPage(ctx, addr, "", 2)
	require.Equal(t, coins[:2], page, "first page")
	require.Equal(t, "ibc/B", last)

	page, last = bank.GetBalancesPage(ctx, addr, last, 2)
	require.Equal(t, coins[2:4], page, "starts strictly after the cursor")
	require.Equal(t, "ibc/E", last)

	page, last = bank.GetBalancesPage(ctx, addr, "ibc/C", 2)
	require.Equal(t, coins[2:4], page, "a cursor denom with no balance")
	require.Equal(t, "ibc/E", last)

	page, last = bank.GetBalancesPage(ctx, addr, "ibc/E", 10)
	require.Equal(t, coins[4:], page, "a short last page")
	require.Equal(t, "ubze", last)

	page, last = bank.GetBalancesPage(ctx, addr, "ubze", 10)
	require.Empty(t, page, "past the last denom")
	require.Empty(t, last)

	page, _ = bank.GetBalancesPage(ctx, addr, "", 0)
	require.Empty(t, page, "no limit, no balances")
}
