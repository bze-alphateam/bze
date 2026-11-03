package app_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/client/flags"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	"github.com/stretchr/testify/require"

	"github.com/bze-alphateam/bze/app"
	burnertypes "github.com/bze-alphateam/bze/x/burner/types"
	tradebintypes "github.com/bze-alphateam/bze/x/tradebin/types"
	v2types "github.com/bze-alphateam/bze/x/tradebin/v2types"
	txfeecollectortypes "github.com/bze-alphateam/bze/x/txfeecollector/types"
)

const (
	// zeroOutputDenom is worth 0.000001 ubze per unit in its pool: 10,000 of it pays a positive pool fee but
	// swaps to less than 1 ubze
	zeroOutputDenom  = "ibc/CHEAPTOKEN"
	healthyFeeDenom  = "ibc/HEALTHYTOKEN"
	zeroOutputAmount = 10_000
)

// newZeroOutputApp builds the real app with default bank, tradebin, txfeecollector and distribution state and two
// native pools: zeroOutputDenom (200,000 BZE / 2e17 units) and healthyFeeDenom (200,000 BZE / 2e11 units), both
// deep enough for module swaps, with a 1% fee that stays in the pool.
func newZeroOutputApp(t *testing.T) (*app.App, sdk.Context) {
	t.Helper()

	appOptions := make(simtestutil.AppOptionsMap, 0)
	appOptions[flags.FlagHome] = t.TempDir()
	bApp, err := app.New(log.NewNopLogger(), dbm.NewMemDB(), nil, true, appOptions)
	require.NoError(t, err)
	ctx := bApp.NewUncachedContext(false, cmtproto.Header{Time: time.Unix(1_700_000_000, 0)})

	// no InitChain here: set the state the conversions read
	require.NoError(t, bApp.BankKeeper.SetParams(ctx, banktypes.DefaultParams()))
	require.NoError(t, bApp.TradebinKeeper.SetParams(ctx, v2types.DefaultParams()))
	require.NoError(t, bApp.TxfeecollectorKeeper.SetParams(ctx, txfeecollectortypes.DefaultParams()))
	require.NoError(t, bApp.DistrKeeper.FeePool.Set(ctx, distrtypes.InitialFeePool()))

	fee := math.LegacyNewDecWithPrec(1, 2)
	setNativePool(t, bApp, ctx, zeroOutputDenom, math.NewInt(200_000_000_000), math.NewInt(200_000_000_000_000_000), fee)
	setNativePool(t, bApp, ctx, healthyFeeDenom, math.NewInt(200_000_000_000), math.NewInt(200_000_000_000), fee)

	return bApp, ctx
}

// setNativePool stores a pool between the native denom and denom, and mints its reserves to the tradebin module.
// The whole fee goes to the providers, so a swap sends nothing to the fee collectors.
func setNativePool(t *testing.T, bApp *app.App, ctx sdk.Context, denom string, nativeReserve, denomReserve math.Int, fee math.LegacyDec) tradebintypes.LiquidityPool {
	t.Helper()

	native := bApp.TradebinKeeper.GetParams(ctx).NativeDenom
	base, quote, id := bApp.TradebinKeeper.CreatePoolId(native, denom)
	pool := tradebintypes.LiquidityPool{
		Id:      id,
		Base:    base,
		Quote:   quote,
		LpDenom: "ulp_" + id,
		Fee:     fee,
		FeeDest: &tradebintypes.FeeDestination{
			Treasury:  math.LegacyZeroDec(),
			Burner:    math.LegacyZeroDec(),
			Providers: math.LegacyOneDec(),
		},
	}
	if base == native {
		pool.ReserveBase, pool.ReserveQuote = nativeReserve, denomReserve
	} else {
		pool.ReserveBase, pool.ReserveQuote = denomReserve, nativeReserve
	}
	bApp.TradebinKeeper.SetLiquidityPool(ctx, pool)

	reserves := sdk.NewCoins(sdk.NewCoin(native, nativeReserve), sdk.NewCoin(denom, denomReserve))
	mintToModule(t, bApp, ctx, tradebintypes.ModuleName, reserves)

	return pool
}

func mintToModule(t *testing.T, bApp *app.App, ctx sdk.Context, module string, coins sdk.Coins) {
	t.Helper()
	require.NoError(t, bApp.BankKeeper.MintCoins(ctx, minttypes.ModuleName, coins))
	require.NoError(t, bApp.BankKeeper.SendCoinsFromModuleToModule(ctx, minttypes.ModuleName, module, coins))
}

// A coin that pays a positive pool fee but swaps to 0 ubze is not swappable, so the community pool conversion leaves
// it out of the module swap batch and the healthy coin next to it is still converted in the same EndBlock. Before,
// the zero-output coin joined the batch, the batch failed and EndBlock discarded the whole conversion.
func TestTxfeecollector_ZeroOutputCoinDoesNotBlockConversion(t *testing.T) {
	bApp, ctx := newZeroOutputApp(t)
	healthy := sdk.NewInt64Coin(healthyFeeDenom, 1_000_000)
	dust := sdk.NewInt64Coin(zeroOutputDenom, zeroOutputAmount)
	mintToModule(t, bApp, ctx, txfeecollectortypes.CpFeeCollector, sdk.NewCoins(healthy, dust))

	native := bApp.TradebinKeeper.GetParams(ctx).NativeDenom
	poolBefore, err := bApp.DistrKeeper.FeePool.Get(ctx)
	require.NoError(t, err)

	endBlocker, ok := bApp.ModuleManager.Modules[txfeecollectortypes.ModuleName].(interface {
		EndBlock(context.Context) error
	})
	require.True(t, ok)
	require.NoError(t, endBlocker.EndBlock(ctx))

	cpCollector := authtypes.NewModuleAddress(txfeecollectortypes.CpFeeCollector)
	require.True(t, bApp.BankKeeper.GetBalance(ctx, cpCollector, healthyFeeDenom).IsZero(), "the healthy coin was swapped")

	poolAfter, err := bApp.DistrKeeper.FeePool.Get(ctx)
	require.NoError(t, err)
	gained := poolAfter.CommunityPool.AmountOf(native).Sub(poolBefore.CommunityPool.AmountOf(native))
	require.True(t, gained.IsPositive(), "the community pool received the swapped native coins")
}

// A zero-output IBC coin the burner is asked to burn is not exchangeable, so it is locked in the black hole instead
// of being sent to a liquidity add whose swap fails and leaves it in the burner.
func TestBurner_ZeroOutputIbcCoinIsLocked(t *testing.T) {
	bApp, ctx := newZeroOutputApp(t)
	dust := sdk.NewCoins(sdk.NewInt64Coin(zeroOutputDenom, zeroOutputAmount))
	mintToModule(t, bApp, ctx, burnertypes.ModuleName, dust)

	_, err := bApp.BurnerKeeper.BurnAnyCoins(ctx, burnertypes.ModuleName, dust)
	require.NoError(t, err)

	blackHole := authtypes.NewModuleAddress(burnertypes.BlackHoleModuleName)
	require.Equal(t, dust, bApp.BankKeeper.GetAllBalances(ctx, blackHole))
	require.True(t, bApp.BankKeeper.GetAllBalances(ctx, authtypes.NewModuleAddress(burnertypes.ModuleName)).IsZero())
}

// Moving a locked balance that swaps to 0 ubze is refused up front with the liquidity error, and the coins stay locked.
func TestBurner_MoveIbcLockedCoins_ZeroOutputBalanceIsRefused(t *testing.T) {
	bApp, ctx := newZeroOutputApp(t)
	dust := sdk.NewCoins(sdk.NewInt64Coin(zeroOutputDenom, zeroOutputAmount))
	mintToModule(t, bApp, ctx, burnertypes.BlackHoleModuleName, dust)

	msg := burnertypes.NewMsgMoveIbcLockedCoins(sdk.AccAddress("locked_coins_mover__").String(), zeroOutputDenom)
	_, err := bApp.MsgServiceRouter().Handler(msg)(ctx, msg)
	require.ErrorContains(t, err, "cannot move the locked coins due to liquidity availability")

	blackHole := authtypes.NewModuleAddress(burnertypes.BlackHoleModuleName)
	require.Equal(t, dust, bApp.BankKeeper.GetAllBalances(ctx, blackHole))
}

// canSwapAgreementPool is a native pool for the agreement property; ratio is the denom units per native unit.
type canSwapAgreementPool struct {
	denom string
	ratio int64
	fee   math.LegacyDec
}

// CanSwapForNativeDenom answers yes exactly when ModuleSwapForNativeDenom of that single coin succeeds with at least
// one native unit, for pools on both sides of the native denom, different fees and prices, and amounts around the
// smallest one that swaps to 1 ubze.
func TestTradebin_CanSwapForNativeDenomAgreesWithModuleSwap(t *testing.T) {
	bApp, ctx := newZeroOutputApp(t)
	nativeReserve := math.NewInt(200_000_000_000)

	var pools []canSwapAgreementPool
	for _, denom := range []string{"ibc/AGREEMENT", "uzagreement"} { // sorts before, resp. after, ubze
		for _, ratio := range []int64{1_000_000, 1_000, 1} {
			for _, fee := range []math.LegacyDec{math.LegacyZeroDec(), math.LegacyNewDecWithPrec(3, 3), math.LegacyNewDecWithPrec(1, 2)} {
				pools = append(pools, canSwapAgreementPool{denom: fmt.Sprintf("%s%d%s", denom, ratio, fee.String()[2:5]), ratio: ratio, fee: fee})
			}
		}
	}

	for _, p := range pools {
		denomReserve := nativeReserve.MulRaw(p.ratio)
		pool := setNativePool(t, bApp, ctx, p.denom, nativeReserve, denomReserve, p.fee)
		native := bApp.TradebinKeeper.GetParams(ctx).NativeDenom
		require.Equal(t, p.denom < native, pool.Base == p.denom, "pool orientation")

		// smallest real input for an output of 1: input_reserve / (output_reserve - 1), then gross it up by the fee
		realInput := denomReserve.Quo(nativeReserve.SubRaw(1)).AddRaw(1)
		boundary := math.LegacyNewDecFromInt(realInput).Quo(math.LegacyOneDec().Sub(p.fee)).TruncateInt()
		amounts := []math.Int{math.NewInt(1), math.NewInt(2), boundary.MulRaw(1_000)}
		for d := int64(-3); d <= 3; d++ {
			if a := boundary.AddRaw(d); a.IsPositive() {
				amounts = append(amounts, a)
			}
		}

		var yes, no int
		for _, amount := range amounts {
			coin := sdk.NewCoin(p.denom, amount)
			cached, _ := ctx.CacheContext()
			canSwap := bApp.TradebinKeeper.CanSwapForNativeDenom(cached, coin)

			mintToModule(t, bApp, cached, txfeecollectortypes.CpFeeCollector, sdk.NewCoins(coin))
			out, err := bApp.TradebinKeeper.ModuleSwapForNativeDenom(cached, txfeecollectortypes.CpFeeCollector, sdk.NewCoins(coin))
			swapped := err == nil && out.Amount.IsPositive()
			require.Equal(t, swapped, canSwap, "pool %s, coin %s: swap error %v", pool.Id, coin, err)

			if canSwap {
				yes++
			} else {
				no++
			}
		}
		require.Positive(t, yes, "pool %s: some amount swaps", pool.Id)
		require.Positive(t, no, "pool %s: some amount is too small", pool.Id)
	}
}
