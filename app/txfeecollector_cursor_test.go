package app_test

import (
	"context"
	"fmt"
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/stretchr/testify/require"

	"github.com/bze-alphateam/bze/app"
	burnertypes "github.com/bze-alphateam/bze/x/burner/types"
	txfeecollectortypes "github.com/bze-alphateam/bze/x/txfeecollector/types"
)

// txfeecollectorEndBlock runs the txfeecollector module's EndBlock on ctx.
func txfeecollectorEndBlock(t *testing.T, bApp *app.App, ctx sdk.Context) {
	t.Helper()

	endBlocker, ok := bApp.ModuleManager.Modules[txfeecollectortypes.ModuleName].(interface {
		EndBlock(context.Context) error
	})
	require.True(t, ok)
	require.NoError(t, endBlocker.EndBlock(ctx))
}

func setMaxBalanceIterations(t *testing.T, bApp *app.App, ctx sdk.Context, n uint64) {
	t.Helper()

	params := bApp.TxfeecollectorKeeper.GetParams(ctx)
	params.MaxBalanceIterations = n
	require.NoError(t, bApp.TxfeecollectorKeeper.SetParams(ctx, params))
}

// More kept dust denoms than MaxBalanceIterations, all with deep pools and sorted before a healthy coin, can't starve
// it: each EndBlock pass starts after the denom the previous one stopped at, so within dust/MaxBalanceIterations + 1
// EndBlocks the healthy coin is converted. The dust is never burned: it waits in the collector. Before, every pass
// read the collector from its first denom and the dust filled the whole bound, so the healthy coin was never looked at.
func TestTxfeecollector_KeptDustDoesNotStarveLaterDenoms(t *testing.T) {
	for _, collector := range []string{txfeecollectortypes.ModuleName, txfeecollectortypes.CpFeeCollector} {
		t.Run(collector, func(t *testing.T) {
			bApp, ctx := newZeroOutputApp(t)
			maxIterations := int(bApp.TxfeecollectorKeeper.GetParams(ctx).MaxBalanceIterations)
			fee := math.LegacyNewDecWithPrec(1, 2)

			// same price as zeroOutputDenom, so zeroOutputAmount units can't be swapped; "ibc/DUST..." sorts before
			// healthyFeeDenom ("ibc/HEALTHYTOKEN")
			var dust sdk.Coins
			for i := 0; i < maxIterations+1; i++ {
				denom := fmt.Sprintf("ibc/DUST%03d", i)
				setNativePool(t, bApp, ctx, denom, math.NewInt(200_000_000_000), math.NewInt(200_000_000_000_000_000), fee)
				coin := sdk.NewInt64Coin(denom, zeroOutputAmount)
				require.False(t, bApp.TradebinKeeper.CanSwapForNativeDenom(ctx, coin))
				require.True(t, bApp.TradebinKeeper.HasDeepLiquidityWithNativeDenom(ctx, denom))
				dust = dust.Add(coin)
			}
			healthy := sdk.NewInt64Coin(healthyFeeDenom, 1_000_000)
			held := dust.Add(healthy)
			require.Equal(t, healthyFeeDenom, held[len(held)-1].Denom, "the healthy coin sorts after all the dust")
			mintToModule(t, bApp, ctx, collector, held)

			addr := authtypes.NewModuleAddress(collector)
			for i := 0; i < len(dust)/maxIterations+1; i++ {
				txfeecollectorEndBlock(t, bApp, ctx)
			}

			require.Equal(t, dust, bApp.BankKeeper.GetAllBalances(ctx, addr), "the healthy coin was converted, the dust waits")
			burner := authtypes.NewModuleAddress(burnertypes.ModuleName)
			require.True(t, bApp.BankKeeper.GetAllBalances(ctx, burner).IsZero(), "nothing was burned")
		})
	}
}

// The cursor moves past the denoms a pass looked at, survives the cursor denom's balance being spent, and wraps to the
// first denom once a pass reaches the end of the balances.
func TestTxfeecollector_CursorSkipsSpentDenomAndWraps(t *testing.T) {
	bApp, ctx := newZeroOutputApp(t)
	setMaxBalanceIterations(t, bApp, ctx, 2)
	fee := math.LegacyNewDecWithPrec(1, 2)
	for _, denom := range []string{"ibc/AAA", "ibc/BBB"} {
		setNativePool(t, bApp, ctx, denom, math.NewInt(200_000_000_000), math.NewInt(200_000_000_000_000_000), fee)
	}
	setNativePool(t, bApp, ctx, "ibc/000", math.NewInt(200_000_000_000), math.NewInt(200_000_000_000), fee)
	dustA := sdk.NewInt64Coin("ibc/AAA", zeroOutputAmount)
	dustB := sdk.NewInt64Coin("ibc/BBB", zeroOutputAmount)
	healthy := sdk.NewInt64Coin(healthyFeeDenom, 1_000_000)
	collector := authtypes.NewModuleAddress(txfeecollectortypes.ModuleName)
	mintToModule(t, bApp, ctx, txfeecollectortypes.ModuleName, sdk.NewCoins(dustA, dustB, healthy))

	// pass 1 looks at AAA and BBB (kept) and stops there
	txfeecollectorEndBlock(t, bApp, ctx)
	require.Equal(t, sdk.NewCoins(dustA, dustB, healthy), bApp.BankKeeper.GetAllBalances(ctx, collector))

	// BBB, the cursor denom, leaves the collector: pass 2 starts after it anyway and converts the healthy coin
	require.NoError(t, bApp.BankKeeper.SendCoinsFromModuleToModule(ctx, txfeecollectortypes.ModuleName, burnertypes.ModuleName, sdk.NewCoins(dustB)))
	txfeecollectorEndBlock(t, bApp, ctx)
	require.Equal(t, sdk.NewCoins(dustA), bApp.BankKeeper.GetAllBalances(ctx, collector))

	// pass 2 reached the end, so pass 3 starts from the first denom and converts a coin sorting before AAA
	early := sdk.NewInt64Coin("ibc/000", 1_000_000)
	mintToModule(t, bApp, ctx, txfeecollectortypes.ModuleName, sdk.NewCoins(early))
	txfeecollectorEndBlock(t, bApp, ctx)
	require.Equal(t, sdk.NewCoins(dustA), bApp.BankKeeper.GetAllBalances(ctx, collector))
}

// A gas fee in a token worth more than 1 ubze per unit can be too small to swap although it is worth well over 1 ubze
// (the pool fee rounds to 0): it waits in the validators' collector, never burned, and is swapped for the validators
// once enough of it builds up.
func TestTxfeecollector_ExpensiveTokenFeeWaitsThenSwaps(t *testing.T) {
	bApp, ctx := newZeroOutputApp(t)
	const expensive = "ibc/EXPENSIVE"
	// 1 unit = 1000 ubze, 1% pool fee: under 100 units the fee rounds to 0
	setNativePool(t, bApp, ctx, expensive, math.NewInt(200_000_000_000), math.NewInt(200_000_000), math.LegacyNewDecWithPrec(1, 2))
	small := sdk.NewInt64Coin(expensive, 10)
	require.False(t, bApp.TradebinKeeper.CanSwapForNativeDenom(ctx, small))
	collector := authtypes.NewModuleAddress(txfeecollectortypes.ModuleName)
	feeCollector := authtypes.NewModuleAddress(authtypes.FeeCollectorName)
	native := bApp.TradebinKeeper.GetParams(ctx).NativeDenom

	mintToModule(t, bApp, ctx, txfeecollectortypes.ModuleName, sdk.NewCoins(small))
	txfeecollectorEndBlock(t, bApp, ctx)
	require.Equal(t, sdk.NewCoins(small), bApp.BankKeeper.GetAllBalances(ctx, collector), "the fee waits")
	require.True(t, bApp.BankKeeper.GetAllBalances(ctx, authtypes.NewModuleAddress(burnertypes.ModuleName)).IsZero(), "nothing burned")

	mintToModule(t, bApp, ctx, txfeecollectortypes.ModuleName, sdk.NewCoins(sdk.NewInt64Coin(expensive, 100)))
	before := bApp.BankKeeper.GetBalance(ctx, feeCollector, native)
	txfeecollectorEndBlock(t, bApp, ctx)
	require.True(t, bApp.BankKeeper.GetAllBalances(ctx, collector).IsZero(), "the whole 110 was swapped")
	require.True(t, bApp.BankKeeper.GetBalance(ctx, feeCollector, native).Amount.GT(before.Amount), "the validators got ubze")
}
