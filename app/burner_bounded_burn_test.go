package app_test

import (
	"fmt"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	"github.com/stretchr/testify/require"

	"github.com/bze-alphateam/bze/app"
	burnertypes "github.com/bze-alphateam/bze/x/burner/types"
)

// The periodic burn reads the burner account in bounded pages with the real bank keeper: with more denoms than one
// batch, each EndBlock call burns at most MaxDenomsBurnPerBlock of them, the queue stays pending until the account is
// empty, and the burned supply is gone.
func TestBurner_PeriodicBurnReadsBoundedPages(t *testing.T) {
	bApp, ctx := newZeroOutputApp(t)
	burner := authtypes.NewModuleAddress(burnertypes.ModuleName)
	creator := authtypes.NewModuleAddress("creator").String()

	factoryCount := 2*burnertypes.MaxDenomsBurnPerBlock + 5
	funds := sdk.NewCoins(sdk.NewInt64Coin("ubze", 1_000))
	for i := 0; i < factoryCount; i++ {
		funds = funds.Add(sdk.NewInt64Coin(fmt.Sprintf("factory/%s/t%03d", creator, i), int64(i+1)))
	}
	mintToModule(t, bApp, ctx, burnertypes.ModuleName, funds)
	supplyBefore := supplyOf(bApp, ctx, funds)
	require.Len(t, bApp.BankKeeper.GetAllBalances(ctx, burner), factoryCount+1)

	bApp.BurnerKeeper.EnqueuePeriodicBurn(ctx)

	// the factory denoms sort before ubze: two full batches, then the last 5 factory denoms with ubze
	for call, wantLeft := range []int{factoryCount + 1 - 100, factoryCount + 1 - 200, 0} {
		require.NoError(t, bApp.BurnerKeeper.ProcessPeriodicBurnQueue(ctx), "call %d", call)
		require.Len(t, bApp.BankKeeper.GetAllBalances(ctx, burner), wantLeft, "balances left after call %d", call)

		_, pending := bApp.BurnerKeeper.GetPeriodicBurnQueue(ctx)
		require.Equal(t, wantLeft > 0, pending, "queue pending after call %d", call)
	}

	require.True(t, bApp.BankKeeper.GetAllBalances(ctx, burner).IsZero(), "the burner account ends empty")
	require.Equal(t, supplyBefore.Sub(funds...).String(), supplyOf(bApp, ctx, funds).String(), "every coin was burned")
}

// The burner module is wired with the bounded page reader over the real bank store.
func TestBurnerBankKeeper_GetBalancesPage(t *testing.T) {
	bApp, ctx := newZeroOutputApp(t)
	baseKeeper, ok := bApp.BankKeeper.(bankkeeper.BaseKeeper)
	require.True(t, ok)
	bank := app.ProvideBurnerBankKeeper(baseKeeper)

	addr := authtypes.NewModuleAddress(burnertypes.ModuleName)
	coins := sdk.NewCoins(sdk.NewInt64Coin("ibc/A", 1), sdk.NewInt64Coin("ibc/B", 2), sdk.NewInt64Coin("ubze", 3))
	mintToModule(t, bApp, ctx, burnertypes.ModuleName, coins)

	page, last := bank.GetBalancesPage(ctx, addr, "", 2)
	require.Equal(t, coins[:2], page)
	require.Equal(t, "ibc/B", last)

	page, last = bank.GetBalancesPage(ctx, addr, "", 10)
	require.Equal(t, coins, page)
	require.Equal(t, "ubze", last)
}
