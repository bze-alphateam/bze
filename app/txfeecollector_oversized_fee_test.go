package app_test

import (
	"math/big"
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/stretchr/testify/require"

	burnertypes "github.com/bze-alphateam/bze/x/burner/types"
	txfeecollectortypes "github.com/bze-alphateam/bze/x/txfeecollector/types"
)

// feeConvOversizedDenom sorts before healthyFeeDenom ("ibc/HEALTHYTOKEN").
const feeConvOversizedDenom = "ibc/BIGFEE"

// A collected coin of ~2^240 with a deep pool is too large for the swap arithmetic. With the real tradebin keeper
// and one coin looked at per EndBlock, the first EndBlock looks at the oversized coin and keeps it, the cursor moves
// past it, and the second EndBlock converts the normal coin next to it. Nothing panics, the oversized coin stays in
// the collector and nothing of it is sent on or burned.
func TestTxfeecollector_OversizedCoinStaysAndCursorMovesOn(t *testing.T) {
	bApp, ctx := newZeroOutputApp(t)
	setMaxBalanceIterations(t, bApp, ctx, 1)
	setNativePool(t, bApp, ctx, feeConvOversizedDenom, math.NewInt(200_000_000_000), math.NewInt(200_000_000_000), math.LegacyNewDecWithPrec(1, 2))
	require.True(t, bApp.TradebinKeeper.HasDeepLiquidityWithNativeDenom(ctx, feeConvOversizedDenom))

	oversized := sdk.NewCoin(feeConvOversizedDenom, math.NewIntFromBigInt(new(big.Int).Lsh(big.NewInt(1), 240)))
	healthy := sdk.NewInt64Coin(healthyFeeDenom, 1_000_000)
	held := sdk.NewCoins(oversized, healthy)
	require.Equal(t, feeConvOversizedDenom, held[0].Denom, "the oversized coin sorts first")
	mintToModule(t, bApp, ctx, txfeecollectortypes.ModuleName, held)

	collector := authtypes.NewModuleAddress(txfeecollectortypes.ModuleName)
	feeCollector := authtypes.NewModuleAddress(authtypes.FeeCollectorName)
	native := bApp.TradebinKeeper.GetParams(ctx).NativeDenom
	nativeBefore := bApp.BankKeeper.GetBalance(ctx, feeCollector, native)

	// pass 1 only looks at the oversized coin and keeps it
	require.NotPanics(t, func() { txfeecollectorEndBlock(t, bApp, ctx) })
	require.Equal(t, held, bApp.BankKeeper.GetAllBalances(ctx, collector))

	// pass 2 starts after it and converts the normal coin
	require.NotPanics(t, func() { txfeecollectorEndBlock(t, bApp, ctx) })
	require.Equal(t, sdk.NewCoins(oversized), bApp.BankKeeper.GetAllBalances(ctx, collector))
	require.True(t, bApp.BankKeeper.GetBalance(ctx, feeCollector, native).Amount.GT(nativeBefore.Amount), "the normal coin was swapped for the validators")

	require.True(t, bApp.BankKeeper.GetBalance(ctx, feeCollector, feeConvOversizedDenom).IsZero(), "the oversized coin was not sent on")
	require.True(t, bApp.BankKeeper.GetAllBalances(ctx, authtypes.NewModuleAddress(burnertypes.ModuleName)).IsZero(), "nothing was burned")
}
