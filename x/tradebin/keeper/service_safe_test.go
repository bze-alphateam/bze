package keeper_test

import (
	"errors"

	"cosmossdk.io/math"
	"github.com/bze-alphateam/bze/x/tradebin/keeper"
	"github.com/bze-alphateam/bze/x/tradebin/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"go.uber.org/mock/gomock"
)

const safeHeight = 10

func safeFeeDest() *types.FeeDestination {
	return &types.FeeDestination{
		Treasury:  math.LegacyNewDecWithPrec(5, 1),
		Burner:    math.LegacyNewDecWithPrec(5, 1),
		Providers: math.LegacyZeroDec(),
	}
}

func (suite *IntegrationTestSuite) safeCtx() sdk.Context {
	return suite.ctx.WithBlockHeight(safeHeight)
}

// safeSeedPool saves the pool between the native denom and other. The first save of the block seeds the average
// with the price of this pool.
func (suite *IntegrationTestSuite) safeSeedPool(ctx sdk.Context, other string, nativeReserve, otherReserve int64) types.LiquidityPool {
	base, quote, id := suite.k.CreatePoolId(denomBze, other)
	pool := types.LiquidityPool{
		Id:      id,
		Base:    base,
		Quote:   quote,
		Fee:     math.LegacyNewDecWithPrec(3, 3),
		FeeDest: safeFeeDest(),
	}
	if base == denomBze {
		pool.ReserveBase, pool.ReserveQuote = math.NewInt(nativeReserve), math.NewInt(otherReserve)
	} else {
		pool.ReserveBase, pool.ReserveQuote = math.NewInt(otherReserve), math.NewInt(nativeReserve)
	}
	suite.k.SetLiquidityPool(ctx, pool)

	return pool
}

// safeMoveQuote changes the quote reserve in the same block, so the average keeps the seeded price.
func (suite *IntegrationTestSuite) safeMoveQuote(ctx sdk.Context, pool types.LiquidityPool, quote math.Int) types.LiquidityPool {
	pool.ReserveQuote = quote
	suite.k.SetLiquidityPool(ctx, pool)

	return pool
}

func (suite *IntegrationTestSuite) safeDropAverage(poolId string) {
	suite.ctx.KVStore(suite.storeKey).Delete(append(types.PoolPriceAveragePrefix(), types.PoolPriceAverageKey(poolId)...))
	_, found := suite.k.GetPoolPriceAverage(suite.safeCtx(), poolId)
	suite.Require().False(found)
}

func (suite *IntegrationTestSuite) safeSetDrifts(fee, module string) {
	params := suite.k.GetParams(suite.ctx)
	params.FeeMaxPriceDrift = math.LegacyMustNewDecFromStr(fee)
	params.ModuleMaxPriceDrift = math.LegacyMustNewDecFromStr(module)
	suite.Require().NoError(suite.k.SetParams(suite.ctx, params))
}

// ---------- fee ----------

var safePayer = sdk.AccAddress("addr1_______________")

func safeFee() sdk.Coins {
	return sdk.NewCoins(sdk.NewCoin(denomBze, math.NewInt(100_000_000)))
}

func (suite *IntegrationTestSuite) safeFeeCtx(ctx sdk.Context) sdk.Context {
	return ctx.WithValue(keeper.CtxFeeDenomKey, denomStake).WithValue(keeper.CtxFeePayerKey, safePayer)
}

// expectFeeInNative expects the fee to be taken directly in the native denom, with no swap.
func (suite *IntegrationTestSuite) expectFeeInNative(err error) {
	suite.bankMock.EXPECT().
		SendCoinsFromAccountToModule(gomock.Any(), safePayer, types.ModuleName, safeFee()).
		Times(1).
		Return(err)
}

// expectFeeSwapped expects the fee to be paid in the preferred denom and swapped.
func (suite *IntegrationTestSuite) expectFeeSwapped(pool types.LiquidityPool) {
	required, err := suite.k.CalculateOptimalInputForOutput(pool, safeFee()[0])
	suite.Require().NoError(err)

	suite.bankMock.EXPECT().SpendableCoins(gomock.Any(), safePayer).Times(1).
		Return(sdk.NewCoins(sdk.NewCoin(denomStake, math.NewInt(50_000_000_000))))
	suite.bankMock.EXPECT().
		SendCoinsFromAccountToModule(gomock.Any(), safePayer, types.ModuleName, sdk.NewCoins(required)).
		Times(1).Return(nil)
	suite.bankMock.EXPECT().SendCoinsFromModuleToModule(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil).AnyTimes()
}

func (suite *IntegrationTestSuite) TestSafeCaptureAndSwapUserFee_NoDriftSwaps() {
	ctx := suite.safeCtx()
	pool := suite.safeSeedPool(ctx, denomStake, 100_000_000_000, 50_000_000_000)
	suite.expectFeeSwapped(pool)

	coins, err := suite.k.SafeCaptureAndSwapUserFee(suite.safeFeeCtx(ctx), safePayer, safeFee(), types.ModuleName)
	suite.Require().NoError(err)
	suite.Require().Len(coins, 1)
	suite.Require().Equal(denomBze, coins[0].Denom)
}

func (suite *IntegrationTestSuite) TestSafeCaptureAndSwapUserFee_AtLimitIsAllowed() {
	ctx := suite.safeCtx()
	// the seeded price is 2 (quote / base); a +10% move is exactly the default limit
	pool := suite.safeSeedPool(ctx, denomStake, 100_000_000_000, 50_000_000_000)
	pool = suite.safeMoveQuote(ctx, pool, math.NewInt(110_000_000_000))
	suite.expectFeeSwapped(pool)

	_, err := suite.k.SafeCaptureAndSwapUserFee(suite.safeFeeCtx(ctx), safePayer, safeFee(), types.ModuleName)
	suite.Require().NoError(err)
}

func (suite *IntegrationTestSuite) TestSafeCaptureAndSwapUserFee_JustOverLimitFallsBackToNative() {
	ctx := suite.safeCtx()
	pool := suite.safeSeedPool(ctx, denomStake, 100_000_000_000, 50_000_000_000)
	suite.safeMoveQuote(ctx, pool, math.NewInt(110_000_000_001))
	// no SpendableCoins and no swap: only the native fee is taken
	suite.expectFeeInNative(nil)

	coins, err := suite.k.SafeCaptureAndSwapUserFee(suite.safeFeeCtx(ctx), safePayer, safeFee(), types.ModuleName)
	suite.Require().NoError(err)
	suite.Require().Equal(safeFee(), coins)
}

func (suite *IntegrationTestSuite) TestSafeCaptureAndSwapUserFee_DriftDownIsChecked() {
	ctx := suite.safeCtx()
	pool := suite.safeSeedPool(ctx, denomStake, 100_000_000_000, 50_000_000_000)
	suite.safeMoveQuote(ctx, pool, math.NewInt(89_000_000_000))
	suite.expectFeeInNative(nil)

	_, err := suite.k.SafeCaptureAndSwapUserFee(suite.safeFeeCtx(ctx), safePayer, safeFee(), types.ModuleName)
	suite.Require().NoError(err)
}

func (suite *IntegrationTestSuite) TestSafeCaptureAndSwapUserFee_DriftedPayerWithoutNativeFails() {
	ctx := suite.safeCtx()
	pool := suite.safeSeedPool(ctx, denomStake, 100_000_000_000, 50_000_000_000)
	suite.safeMoveQuote(ctx, pool, math.NewInt(120_000_000_000))
	boom := errors.New("insufficient funds")
	suite.expectFeeInNative(boom)

	_, err := suite.k.SafeCaptureAndSwapUserFee(suite.safeFeeCtx(ctx), safePayer, safeFee(), types.ModuleName)
	suite.Require().ErrorIs(err, boom)
}

func (suite *IntegrationTestSuite) TestSafeCaptureAndSwapUserFee_MissingAverageFallsBackToNative() {
	ctx := suite.safeCtx()
	pool := suite.safeSeedPool(ctx, denomStake, 100_000_000_000, 50_000_000_000)
	suite.safeDropAverage(pool.Id)
	suite.expectFeeInNative(nil)

	_, err := suite.k.SafeCaptureAndSwapUserFee(suite.safeFeeCtx(ctx), safePayer, safeFee(), types.ModuleName)
	suite.Require().NoError(err)
}

func (suite *IntegrationTestSuite) TestSafeCaptureAndSwapUserFee_NoPriceFallsBackToNative() {
	ctx := suite.safeCtx()
	pool := suite.safeSeedPool(ctx, denomStake, 100_000_000_000, 50_000_000_000)
	suite.safeMoveQuote(ctx, pool, math.ZeroInt())
	suite.expectFeeInNative(nil)

	_, err := suite.k.SafeCaptureAndSwapUserFee(suite.safeFeeCtx(ctx), safePayer, safeFee(), types.ModuleName)
	suite.Require().NoError(err)
}

func (suite *IntegrationTestSuite) TestSafeCaptureAndSwapUserFee_NoPreferredDenom() {
	suite.expectFeeInNative(nil)

	coins, err := suite.k.SafeCaptureAndSwapUserFee(suite.safeCtx(), safePayer, safeFee(), types.ModuleName)
	suite.Require().NoError(err)
	suite.Require().Equal(safeFee(), coins)
}

func (suite *IntegrationTestSuite) TestSafeCaptureAndSwapUserFee_PreferredDenomIsNative() {
	suite.expectFeeInNative(nil)
	ctx := suite.safeCtx().WithValue(keeper.CtxFeeDenomKey, denomBze).WithValue(keeper.CtxFeePayerKey, safePayer)

	_, err := suite.k.SafeCaptureAndSwapUserFee(ctx, safePayer, safeFee(), types.ModuleName)
	suite.Require().NoError(err)
}

func (suite *IntegrationTestSuite) TestSafeCaptureAndSwapUserFee_NoPool() {
	suite.expectFeeInNative(nil)

	_, err := suite.k.SafeCaptureAndSwapUserFee(suite.safeFeeCtx(suite.safeCtx()), safePayer, safeFee(), types.ModuleName)
	suite.Require().NoError(err)
}

func (suite *IntegrationTestSuite) TestSafeCaptureAndSwapUserFee_UsesFeeLimit() {
	ctx := suite.safeCtx()
	pool := suite.safeSeedPool(ctx, denomStake, 100_000_000_000, 50_000_000_000)
	pool = suite.safeMoveQuote(ctx, pool, math.NewInt(120_000_000_000)) // +20%

	// the module limit is tiny, the fee limit is wide: the fee is still swapped
	suite.safeSetDrifts("0.25", "0.001")
	suite.expectFeeSwapped(pool)
	_, err := suite.k.SafeCaptureAndSwapUserFee(suite.safeFeeCtx(ctx), safePayer, safeFee(), types.ModuleName)
	suite.Require().NoError(err)
}

func (suite *IntegrationTestSuite) TestSafeCaptureAndSwapUserFee_IgnoresModuleLimit() {
	ctx := suite.safeCtx()
	pool := suite.safeSeedPool(ctx, denomStake, 100_000_000_000, 50_000_000_000)
	suite.safeMoveQuote(ctx, pool, math.NewInt(120_000_000_000)) // +20%

	// the fee limit is tiny, the module limit is wide: the fee falls back to native
	suite.safeSetDrifts("0.001", "0.9")
	suite.expectFeeInNative(nil)
	_, err := suite.k.SafeCaptureAndSwapUserFee(suite.safeFeeCtx(ctx), safePayer, safeFee(), types.ModuleName)
	suite.Require().NoError(err)
}

// ---------- module functions ----------

const (
	safeModule = "test_module"
	safeOther1 = "uother"
	safeOther2 = "uthird"
)

func (suite *IntegrationTestSuite) expectModuleAccounts(withTradebin bool) {
	addr := sdk.AccAddress("moduleacc")
	suite.accountMock.EXPECT().GetModuleAccount(gomock.Any(), safeModule).Times(1).Return(&authtypes.ModuleAccount{
		BaseAccount: &authtypes.BaseAccount{Address: addr.String()},
		Name:        safeModule,
	})
	if withTradebin {
		tbAddr := sdk.AccAddress("tradebinmodule")
		suite.accountMock.EXPECT().GetModuleAccount(gomock.Any(), types.ModuleName).Times(1).Return(&authtypes.ModuleAccount{
			BaseAccount: &authtypes.BaseAccount{Address: tbAddr.String()},
			Name:        types.ModuleName,
		})
	}
}

func (suite *IntegrationTestSuite) expectModuleSwapCalls(coins sdk.Coins) {
	suite.expectModuleAccounts(true)
	suite.bankMock.EXPECT().SendCoinsFromModuleToModule(gomock.Any(), types.ModuleName, "burner", gomock.Any()).Return(nil).Times(1)
	suite.bankMock.EXPECT().SendCoinsFromModuleToModule(gomock.Any(), safeModule, types.ModuleName, coins).Return(nil).Times(1)
	suite.bankMock.EXPECT().SendCoinsFromModuleToModule(gomock.Any(), types.ModuleName, safeModule, gomock.Any()).Return(nil).Times(1)
	suite.bankMock.EXPECT().SendCoinsFromModuleToModule(gomock.Any(), types.ModuleName, gomock.Any(), gomock.Any()).Return(nil).Times(1)
}

func (suite *IntegrationTestSuite) TestSafeModuleSwapForNativeDenom_NoDriftSwaps() {
	ctx := suite.safeCtx()
	suite.safeSeedPool(ctx, safeOther1, 200_000_000_000, 1_000_000_000)
	coins := sdk.NewCoins(sdk.NewInt64Coin(safeOther1, 1000))
	suite.expectModuleSwapCalls(coins)

	result, err := suite.k.SafeModuleSwapForNativeDenom(ctx, safeModule, coins)
	suite.Require().NoError(err)
	suite.Require().Equal(denomBze, result.Denom)
	suite.Require().True(result.Amount.IsPositive())
}

func (suite *IntegrationTestSuite) TestSafeModuleSwapForNativeDenom_AtLimitIsAllowed() {
	ctx := suite.safeCtx()
	// price 1 -> 1.03 is exactly the default 3% limit
	pool := suite.safeSeedPool(ctx, safeOther1, 200_000_000_000, 200_000_000_000)
	suite.safeMoveQuote(ctx, pool, math.NewInt(206_000_000_000))
	coins := sdk.NewCoins(sdk.NewInt64Coin(safeOther1, 1000))
	suite.expectModuleSwapCalls(coins)

	_, err := suite.k.SafeModuleSwapForNativeDenom(ctx, safeModule, coins)
	suite.Require().NoError(err)
}

func (suite *IntegrationTestSuite) TestSafeModuleSwapForNativeDenom_JustOverLimitIsRefused() {
	ctx := suite.safeCtx()
	pool := suite.safeSeedPool(ctx, safeOther1, 200_000_000_000, 200_000_000_000)
	moved := suite.safeMoveQuote(ctx, pool, math.NewInt(206_000_000_001))
	coins := sdk.NewCoins(sdk.NewInt64Coin(safeOther1, 1000))

	// gomock fails the test on any unexpected bank or account call
	result, err := suite.k.SafeModuleSwapForNativeDenom(ctx, safeModule, coins)
	suite.Require().ErrorIs(err, types.ErrPoolPriceDrift)
	suite.Require().Equal(sdk.Coin{}, result)

	stored, found := suite.k.GetLiquidityPool(ctx, pool.Id)
	suite.Require().True(found)
	suite.Require().Equal(moved.ReserveBase, stored.ReserveBase)
	suite.Require().Equal(moved.ReserveQuote, stored.ReserveQuote)
}

func (suite *IntegrationTestSuite) TestSafeModuleSwapForNativeDenom_OneDriftedPoolOfSeveralIsRefused() {
	ctx := suite.safeCtx()
	suite.safeSeedPool(ctx, safeOther1, 200_000_000_000, 200_000_000_000)
	second := suite.safeSeedPool(ctx, safeOther2, 200_000_000_000, 200_000_000_000)
	suite.safeMoveQuote(ctx, second, math.NewInt(230_000_000_000))
	coins := sdk.NewCoins(sdk.NewInt64Coin(safeOther1, 1000), sdk.NewInt64Coin(safeOther2, 1000))

	_, err := suite.k.SafeModuleSwapForNativeDenom(ctx, safeModule, coins)
	suite.Require().ErrorIs(err, types.ErrPoolPriceDrift)
}

func (suite *IntegrationTestSuite) TestSafeModuleSwapForNativeDenom_MissingAverageIsRefused() {
	ctx := suite.safeCtx()
	pool := suite.safeSeedPool(ctx, safeOther1, 200_000_000_000, 200_000_000_000)
	suite.safeDropAverage(pool.Id)

	_, err := suite.k.SafeModuleSwapForNativeDenom(ctx, safeModule, sdk.NewCoins(sdk.NewInt64Coin(safeOther1, 1000)))
	suite.Require().ErrorIs(err, types.ErrPoolPriceDrift)
}

func (suite *IntegrationTestSuite) TestSafeModuleSwapForNativeDenom_UsesModuleLimit() {
	ctx := suite.safeCtx()
	pool := suite.safeSeedPool(ctx, safeOther1, 200_000_000_000, 200_000_000_000)
	suite.safeMoveQuote(ctx, pool, math.NewInt(220_000_000_000)) // +10%
	coins := sdk.NewCoins(sdk.NewInt64Coin(safeOther1, 1000))

	// a wide fee limit does not help the module swap
	suite.safeSetDrifts("0.9", "0.03")
	_, err := suite.k.SafeModuleSwapForNativeDenom(ctx, safeModule, coins)
	suite.Require().ErrorIs(err, types.ErrPoolPriceDrift)

	// a wide module limit lets it through
	suite.safeSetDrifts("0.001", "0.2")
	suite.expectModuleSwapCalls(coins)
	_, err = suite.k.SafeModuleSwapForNativeDenom(ctx, safeModule, coins)
	suite.Require().NoError(err)
}

func (suite *IntegrationTestSuite) TestSafeModuleSwapForNativeDenom_NoPoolIsLeftToTheOriginal() {
	coins := sdk.NewCoins(sdk.NewInt64Coin(safeOther1, 1000))
	suite.expectModuleAccounts(false)
	suite.bankMock.EXPECT().SendCoinsFromModuleToModule(gomock.Any(), safeModule, types.ModuleName, coins).Return(nil).Times(1)

	_, err := suite.k.SafeModuleSwapForNativeDenom(suite.safeCtx(), safeModule, coins)
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "cannot find liquidity pool")
}

func (suite *IntegrationTestSuite) expectAddLiquidityCalls(coins sdk.Coins, lpDenom string) {
	suite.expectModuleAccountsTradebinOnly()
	suite.bankMock.EXPECT().GetSupply(gomock.Any(), lpDenom).Return(sdk.NewCoin(lpDenom, math.NewInt(1_000_000_000))).Times(1)
	suite.bankMock.EXPECT().SendCoinsFromModuleToModule(gomock.Any(), safeModule, types.ModuleName, coins).Return(nil).Times(1)
	suite.bankMock.EXPECT().MintCoins(gomock.Any(), types.ModuleName, gomock.Any()).Return(nil).Times(1)
	suite.bankMock.EXPECT().SendCoinsFromModuleToModule(gomock.Any(), types.ModuleName, safeModule, gomock.Any()).Return(nil).Times(1)
}

func (suite *IntegrationTestSuite) expectModuleAccountsTradebinOnly() {
	tbAddr := sdk.AccAddress("tradebinmodule")
	suite.accountMock.EXPECT().GetModuleAccount(gomock.Any(), types.ModuleName).Times(1).Return(&authtypes.ModuleAccount{
		BaseAccount: &authtypes.BaseAccount{Address: tbAddr.String()},
		Name:        types.ModuleName,
	})
}

func (suite *IntegrationTestSuite) safeSeedLiquidityPool(ctx sdk.Context) types.LiquidityPool {
	pool := suite.safeSeedPool(ctx, safeOther1, 1_000_000, 2_000_000)
	pool.LpDenom = "ulp_ubze_uother"
	pool.FeeDest = &types.FeeDestination{Treasury: math.LegacyZeroDec(), Burner: math.LegacyZeroDec(), Providers: math.LegacyOneDec()}
	suite.k.SetLiquidityPool(ctx, pool)

	return pool
}

func (suite *IntegrationTestSuite) TestSafeModuleAddLiquidityWithNativeDenom_NoDriftAdds() {
	ctx := suite.safeCtx()
	pool := suite.safeSeedLiquidityPool(ctx)
	coins := sdk.NewCoins(sdk.NewInt64Coin(safeOther1, 500_000))
	suite.expectAddLiquidityCalls(coins, pool.LpDenom)

	added, _, err := suite.k.SafeModuleAddLiquidityWithNativeDenom(ctx, safeModule, coins)
	suite.Require().NoError(err)
	suite.Require().NotEmpty(added)
}

func (suite *IntegrationTestSuite) TestSafeModuleAddLiquidityWithNativeDenom_AtLimitIsAllowed() {
	ctx := suite.safeCtx()
	pool := suite.safeSeedLiquidityPool(ctx)
	// the seeded price is 2 (quote / base); a +3% move is exactly the limit
	pool = suite.safeMoveQuote(ctx, pool, math.NewInt(2_060_000))
	coins := sdk.NewCoins(sdk.NewInt64Coin(safeOther1, 500_000))
	suite.expectAddLiquidityCalls(coins, pool.LpDenom)

	_, _, err := suite.k.SafeModuleAddLiquidityWithNativeDenom(ctx, safeModule, coins)
	suite.Require().NoError(err)
}

func (suite *IntegrationTestSuite) TestSafeModuleAddLiquidityWithNativeDenom_JustOverLimitIsRefused() {
	ctx := suite.safeCtx()
	pool := suite.safeSeedLiquidityPool(ctx)
	moved := suite.safeMoveQuote(ctx, pool, math.NewInt(2_060_001))
	coins := sdk.NewCoins(sdk.NewInt64Coin(safeOther1, 500_000))

	added, refunded, err := suite.k.SafeModuleAddLiquidityWithNativeDenom(ctx, safeModule, coins)
	suite.Require().ErrorIs(err, types.ErrPoolPriceDrift)
	suite.Require().Nil(added)
	suite.Require().Nil(refunded)

	stored, found := suite.k.GetLiquidityPool(ctx, pool.Id)
	suite.Require().True(found)
	suite.Require().Equal(moved.ReserveQuote, stored.ReserveQuote)
}

func (suite *IntegrationTestSuite) TestSafeModuleAddLiquidityWithNativeDenom_OneDriftedPoolOfSeveralIsRefused() {
	ctx := suite.safeCtx()
	suite.safeSeedPool(ctx, safeOther1, 200_000_000_000, 200_000_000_000)
	second := suite.safeSeedPool(ctx, safeOther2, 200_000_000_000, 200_000_000_000)
	suite.safeMoveQuote(ctx, second, math.NewInt(230_000_000_000))
	coins := sdk.NewCoins(sdk.NewInt64Coin(safeOther1, 1000), sdk.NewInt64Coin(safeOther2, 1000))

	_, _, err := suite.k.SafeModuleAddLiquidityWithNativeDenom(ctx, safeModule, coins)
	suite.Require().ErrorIs(err, types.ErrPoolPriceDrift)
}

func (suite *IntegrationTestSuite) TestSafeModuleAddLiquidityWithNativeDenom_MissingAverageIsRefused() {
	ctx := suite.safeCtx()
	pool := suite.safeSeedLiquidityPool(ctx)
	suite.safeDropAverage(pool.Id)

	_, _, err := suite.k.SafeModuleAddLiquidityWithNativeDenom(ctx, safeModule, sdk.NewCoins(sdk.NewInt64Coin(safeOther1, 500_000)))
	suite.Require().ErrorIs(err, types.ErrPoolPriceDrift)
}

func (suite *IntegrationTestSuite) TestSafeModuleAddLiquidityWithNativeDenom_UsesModuleLimit() {
	ctx := suite.safeCtx()
	pool := suite.safeSeedLiquidityPool(ctx)
	pool = suite.safeMoveQuote(ctx, pool, math.NewInt(2_200_000)) // +10%
	coins := sdk.NewCoins(sdk.NewInt64Coin(safeOther1, 500_000))

	suite.safeSetDrifts("0.9", "0.03")
	_, _, err := suite.k.SafeModuleAddLiquidityWithNativeDenom(ctx, safeModule, coins)
	suite.Require().ErrorIs(err, types.ErrPoolPriceDrift)

	suite.safeSetDrifts("0.001", "0.2")
	suite.expectAddLiquidityCalls(coins, pool.LpDenom)
	_, _, err = suite.k.SafeModuleAddLiquidityWithNativeDenom(ctx, safeModule, coins)
	suite.Require().NoError(err)
}
