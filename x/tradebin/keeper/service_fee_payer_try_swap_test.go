package keeper_test

import (
	"fmt"

	"cosmossdk.io/math"
	burnermoduletypes "github.com/bze-alphateam/bze/x/burner/types"
	"github.com/bze-alphateam/bze/x/tradebin/keeper"
	"github.com/bze-alphateam/bze/x/tradebin/types"
	txfeecollectormoduletypes "github.com/bze-alphateam/bze/x/txfeecollector/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"go.uber.org/mock/gomock"
)

// helper to create a standard deep-liquidity pool used by most tests in this file
func trySwapDeepPool() types.LiquidityPool {
	return types.LiquidityPool{
		Id:           "stake_ubze",
		Base:         denomStake,
		Quote:        denomBze,
		ReserveBase:  math.NewInt(50_000_000_000),
		ReserveQuote: math.NewInt(100_000_000_000),
		Fee:          math.LegacyNewDecWithPrec(3, 3), // 0.3%
		FeeDest: &types.FeeDestination{
			Treasury:  math.LegacyNewDecWithPrec(5, 1),
			Burner:    math.LegacyNewDecWithPrec(3, 1),
			Providers: math.LegacyNewDecWithPrec(2, 1),
		},
	}
}

// ============================================================================
// CaptureAndTryToSwapUserFeesOrSendItAsIs tests
// ============================================================================

// --- Invalid input ---

func (suite *IntegrationTestSuite) TestTrySwap_InvalidFee() {
	addr1 := sdk.AccAddress("addr1_______________")

	// Empty fee
	_, err := suite.k.CaptureAndTryToSwapUserFeesOrSendItAsIs(suite.ctx, addr1, sdk.NewCoins(), types.ModuleName)
	suite.Require().NotNil(err)
	suite.Require().Contains(err.Error(), "can not capture user fees that are not all positive")

	// Negative fee
	invalidFee := sdk.Coins{sdk.Coin{Denom: denomBze, Amount: math.NewInt(-100)}}
	_, err = suite.k.CaptureAndTryToSwapUserFeesOrSendItAsIs(suite.ctx, addr1, invalidFee, types.ModuleName)
	suite.Require().NotNil(err)
}

// --- Fallback to native denom (same behavior as CaptureAndSwapUserFee) ---

func (suite *IntegrationTestSuite) TestTrySwap_NoPreferredDenom() {
	addr1 := sdk.AccAddress("addr1_______________")
	fee := sdk.NewCoins(sdk.NewCoin(denomBze, math.NewInt(1000000)))

	suite.bankMock.EXPECT().
		SendCoinsFromAccountToModule(gomock.Any(), addr1, types.ModuleName, fee).
		Times(1).
		Return(nil)

	captured, err := suite.k.CaptureAndTryToSwapUserFeesOrSendItAsIs(suite.ctx, addr1, fee, types.ModuleName)
	suite.Require().Nil(err)
	suite.Require().Equal(fee, captured)
}

func (suite *IntegrationTestSuite) TestTrySwap_PreferredDenomSameAsNative() {
	addr1 := sdk.AccAddress("addr1_______________")
	fee := sdk.NewCoins(sdk.NewCoin(denomBze, math.NewInt(1000000)))
	ctx := suite.ctx.WithValue(keeper.CtxFeeDenomKey, denomBze)

	suite.bankMock.EXPECT().
		SendCoinsFromAccountToModule(gomock.Any(), addr1, types.ModuleName, fee).
		Times(1).
		Return(nil)

	captured, err := suite.k.CaptureAndTryToSwapUserFeesOrSendItAsIs(ctx, addr1, fee, types.ModuleName)
	suite.Require().Nil(err)
	suite.Require().Equal(fee, captured)
}

func (suite *IntegrationTestSuite) TestTrySwap_NoNativeFeeInCoins() {
	addr1 := sdk.AccAddress("addr1_______________")
	fee := sdk.NewCoins(sdk.NewCoin(denomStake, math.NewInt(1000000)))
	ctx := suite.ctx.WithValue(keeper.CtxFeeDenomKey, "uatom")

	suite.bankMock.EXPECT().
		SendCoinsFromAccountToModule(gomock.Any(), addr1, types.ModuleName, fee).
		Times(1).
		Return(nil)

	captured, err := suite.k.CaptureAndTryToSwapUserFeesOrSendItAsIs(ctx, addr1, fee, types.ModuleName)
	suite.Require().Nil(err)
	suite.Require().Equal(fee, captured)
}

func (suite *IntegrationTestSuite) TestTrySwap_PoolNotFound() {
	addr1 := sdk.AccAddress("addr1_______________")
	fee := sdk.NewCoins(sdk.NewCoin(denomBze, math.NewInt(1000000)))
	ctx := suite.ctx.WithValue(keeper.CtxFeeDenomKey, "uatom")

	suite.bankMock.EXPECT().
		SendCoinsFromAccountToModule(gomock.Any(), addr1, types.ModuleName, fee).
		Times(1).
		Return(nil)

	captured, err := suite.k.CaptureAndTryToSwapUserFeesOrSendItAsIs(ctx, addr1, fee, types.ModuleName)
	suite.Require().Nil(err)
	suite.Require().Equal(fee, captured)
}

func (suite *IntegrationTestSuite) TestTrySwap_ZeroNativeReserves() {
	addr1 := sdk.AccAddress("addr1_______________")
	fee := sdk.NewCoins(sdk.NewCoin(denomBze, math.NewInt(100000000)))

	pool := trySwapDeepPool()
	pool.ReserveQuote = math.ZeroInt()
	suite.k.SetLiquidityPool(suite.ctx, pool)

	ctx := suite.ctx.WithValue(keeper.CtxFeeDenomKey, denomStake)

	suite.bankMock.EXPECT().
		SendCoinsFromAccountToModule(gomock.Any(), addr1, types.ModuleName, fee).
		Times(1).
		Return(nil)

	captured, err := suite.k.CaptureAndTryToSwapUserFeesOrSendItAsIs(ctx, addr1, fee, types.ModuleName)
	suite.Require().Nil(err)
	suite.Require().Equal(fee, captured)
}

func (suite *IntegrationTestSuite) TestTrySwap_InsufficientLiquidity() {
	addr1 := sdk.AccAddress("addr1_______________")
	fee := sdk.NewCoins(sdk.NewCoin(denomBze, math.NewInt(100000000)))

	pool := trySwapDeepPool()
	pool.ReserveQuote = math.NewInt(74_999_999_999) // below 75B threshold
	pool.ReserveBase = math.NewInt(37_000_000_000)
	suite.k.SetLiquidityPool(suite.ctx, pool)

	ctx := suite.ctx.WithValue(keeper.CtxFeeDenomKey, denomStake)

	suite.bankMock.EXPECT().
		SendCoinsFromAccountToModule(gomock.Any(), addr1, types.ModuleName, fee).
		Times(1).
		Return(nil)

	captured, err := suite.k.CaptureAndTryToSwapUserFeesOrSendItAsIs(ctx, addr1, fee, types.ModuleName)
	suite.Require().Nil(err)
	suite.Require().Equal(fee, captured)
}

func (suite *IntegrationTestSuite) TestTrySwap_SwapCalculationFails() {
	addr1 := sdk.AccAddress("addr1_______________")
	// Fee equals pool reserve — CalculateOptimalInputForOutput fails
	nativeFee := sdk.NewCoin(denomBze, math.NewInt(100_000_000_000))
	fee := sdk.NewCoins(nativeFee)

	pool := trySwapDeepPool()
	suite.k.SetLiquidityPool(suite.ctx, pool)

	ctx := suite.ctx.WithValue(keeper.CtxFeeDenomKey, denomStake)

	suite.bankMock.EXPECT().
		SendCoinsFromAccountToModule(gomock.Any(), addr1, types.ModuleName, fee).
		Times(1).
		Return(nil)

	captured, err := suite.k.CaptureAndTryToSwapUserFeesOrSendItAsIs(ctx, addr1, fee, types.ModuleName)
	suite.Require().Nil(err)
	suite.Require().Equal(fee, captured)
}

func (suite *IntegrationTestSuite) TestTrySwap_InsufficientBalance() {
	addr1 := sdk.AccAddress("addr1_______________")
	nativeFee := sdk.NewCoin(denomBze, math.NewInt(100000000))
	fee := sdk.NewCoins(nativeFee)

	pool := trySwapDeepPool()
	suite.k.SetLiquidityPool(suite.ctx, pool)

	ctx := suite.ctx.WithValue(keeper.CtxFeeDenomKey, denomStake)

	suite.bankMock.EXPECT().
		SpendableCoins(gomock.Any(), addr1).
		Times(1).
		Return(sdk.NewCoins(sdk.NewCoin(denomStake, math.NewInt(100))))

	suite.bankMock.EXPECT().
		SendCoinsFromAccountToModule(gomock.Any(), addr1, types.ModuleName, fee).
		Times(1).
		Return(nil)

	captured, err := suite.k.CaptureAndTryToSwapUserFeesOrSendItAsIs(ctx, addr1, fee, types.ModuleName)
	suite.Require().Nil(err)
	suite.Require().Equal(fee, captured)
}

func (suite *IntegrationTestSuite) TestTrySwap_CaptureFails() {
	addr1 := sdk.AccAddress("addr1_______________")
	fee := sdk.NewCoins(sdk.NewCoin(denomBze, math.NewInt(1000000)))

	suite.bankMock.EXPECT().
		SendCoinsFromAccountToModule(gomock.Any(), addr1, types.ModuleName, fee).
		Times(1).
		Return(fmt.Errorf("insufficient funds"))

	_, err := suite.k.CaptureAndTryToSwapUserFeesOrSendItAsIs(suite.ctx, addr1, fee, types.ModuleName)
	suite.Require().NotNil(err)
	suite.Require().Contains(err.Error(), "insufficient funds")
}

// --- Amount too small to swap — the key new behavior ---

// TestTrySwap_AmountTooSmall_CapturesNonNativeAsIs verifies that when the converted fee amount is too small
// to produce a non-zero pool swap fee, the function captures the non-native coins as-is (without swapping)
// into the toModule. The caller (captureTradingFees) is then responsible for routing them to the destination
// fee collector where txfeecollector's EndBlock will accumulate and batch-swap them.
func (suite *IntegrationTestSuite) TestTrySwap_AmountTooSmall_CapturesNonNativeAsIs() {
	addr1 := sdk.AccAddress("addr1_______________")
	// Very small native fee — when converted to stake, pool fee will round to zero
	nativeFee := sdk.NewCoin(denomBze, math.NewInt(1))
	fee := sdk.NewCoins(nativeFee)

	pool := trySwapDeepPool()
	suite.k.SetLiquidityPool(suite.ctx, pool)

	ctx := suite.ctx.WithValue(keeper.CtxFeeDenomKey, denomStake)

	requiredStake, err := suite.k.CalculateOptimalInputForOutput(pool, nativeFee)
	suite.Require().Nil(err)

	toCapture := sdk.NewCoins(requiredStake)

	// Mock balance check — user has enough
	suite.bankMock.EXPECT().
		SpendableCoins(gomock.Any(), addr1).
		Times(1).
		Return(sdk.NewCoins(sdk.NewCoin(denomStake, math.NewInt(50_000_000_000))))

	// Mock capture from user into toModule — non-native coins captured as-is
	suite.bankMock.EXPECT().
		SendCoinsFromAccountToModule(gomock.Any(), addr1, types.ModuleName, toCapture).
		Times(1).
		Return(nil)

	captured, err := suite.k.CaptureAndTryToSwapUserFeesOrSendItAsIs(ctx, addr1, fee, types.ModuleName)
	suite.Require().Nil(err)
	// Returns the non-native toCapture coins (not nil, not native)
	suite.Require().Equal(toCapture, captured)
	// Verify it's the preferred denom, not the native denom
	suite.Require().Equal(denomStake, captured[0].Denom)
}

// TestTrySwap_AmountTooSmall_CaptureError verifies that when the amount is too small to swap
// and the bank capture fails, the error is propagated.
func (suite *IntegrationTestSuite) TestTrySwap_AmountTooSmall_CaptureError() {
	addr1 := sdk.AccAddress("addr1_______________")
	nativeFee := sdk.NewCoin(denomBze, math.NewInt(1))
	fee := sdk.NewCoins(nativeFee)

	pool := trySwapDeepPool()
	suite.k.SetLiquidityPool(suite.ctx, pool)

	ctx := suite.ctx.WithValue(keeper.CtxFeeDenomKey, denomStake)

	requiredStake, err := suite.k.CalculateOptimalInputForOutput(pool, nativeFee)
	suite.Require().Nil(err)

	toCapture := sdk.NewCoins(requiredStake)

	suite.bankMock.EXPECT().
		SpendableCoins(gomock.Any(), addr1).
		Times(1).
		Return(sdk.NewCoins(sdk.NewCoin(denomStake, math.NewInt(50_000_000_000))))

	// Capture fails
	suite.bankMock.EXPECT().
		SendCoinsFromAccountToModule(gomock.Any(), addr1, types.ModuleName, toCapture).
		Times(1).
		Return(fmt.Errorf("insufficient funds"))

	_, err = suite.k.CaptureAndTryToSwapUserFeesOrSendItAsIs(ctx, addr1, fee, types.ModuleName)
	suite.Require().NotNil(err)
	suite.Require().Contains(err.Error(), "insufficient funds")
}

// TestTrySwap_AmountTooSmall_OriginalFunctionFails verifies that the original CaptureAndSwapUserFee
// FAILS with the same tiny fee (the swap error propagates), confirming the behavioral difference.
func (suite *IntegrationTestSuite) TestTrySwap_AmountTooSmall_OriginalFunctionFails() {
	addr1 := sdk.AccAddress("addr1_______________")
	nativeFee := sdk.NewCoin(denomBze, math.NewInt(1))
	fee := sdk.NewCoins(nativeFee)

	pool := trySwapDeepPool()
	suite.k.SetLiquidityPool(suite.ctx, pool)

	ctx := suite.ctx.WithValue(keeper.CtxFeeDenomKey, denomStake)

	requiredStake, err := suite.k.CalculateOptimalInputForOutput(pool, nativeFee)
	suite.Require().Nil(err)

	toCapture := sdk.NewCoins(requiredStake)

	suite.bankMock.EXPECT().
		SpendableCoins(gomock.Any(), addr1).
		Times(1).
		Return(sdk.NewCoins(sdk.NewCoin(denomStake, math.NewInt(50_000_000_000))))

	// Capture into tradebin succeeds
	suite.bankMock.EXPECT().
		SendCoinsFromAccountToModule(gomock.Any(), addr1, types.ModuleName, toCapture).
		Times(1).
		Return(nil)

	// The original function attempts the swap which fails — "amount is too low to be traded"
	_, err = suite.k.CaptureAndSwapUserFee(ctx, addr1, fee, types.ModuleName)
	suite.Require().NotNil(err)
	suite.Require().Contains(err.Error(), "amount is too low to be traded")
}

// --- Successful swap (amount is large enough) ---

func (suite *IntegrationTestSuite) TestTrySwap_SwapSuccess() {
	addr1 := sdk.AccAddress("addr1_______________")
	nativeFee := sdk.NewCoin(denomBze, math.NewInt(100000000))
	fee := sdk.NewCoins(nativeFee)

	pool := trySwapDeepPool()
	suite.k.SetLiquidityPool(suite.ctx, pool)

	ctx := suite.ctx.WithValue(keeper.CtxFeeDenomKey, denomStake)

	requiredStake, err := suite.k.CalculateOptimalInputForOutput(pool, nativeFee)
	suite.Require().Nil(err)

	toCapture := sdk.NewCoins(requiredStake)

	suite.bankMock.EXPECT().
		SpendableCoins(gomock.Any(), addr1).
		Times(1).
		Return(sdk.NewCoins(sdk.NewCoin(denomStake, math.NewInt(50_000_000_000))))

	suite.bankMock.EXPECT().
		SendCoinsFromAccountToModule(gomock.Any(), addr1, types.ModuleName, toCapture).
		Times(1).
		Return(nil)

	// collectSwapFee sends treasury portion from tradebin to CpFeeCollector
	suite.bankMock.EXPECT().
		SendCoinsFromModuleToModule(gomock.Any(), types.ModuleName, txfeecollectormoduletypes.CpFeeCollector, gomock.Any()).
		Times(1).
		Return(nil)

	// collectSwapFee sends burner portion from tradebin to burner module
	suite.bankMock.EXPECT().
		SendCoinsFromModuleToModule(gomock.Any(), types.ModuleName, burnermoduletypes.ModuleName, gomock.Any()).
		Times(1).
		Return(nil)

	// toModule == types.ModuleName so no cross-module transfer expected

	captured, err := suite.k.CaptureAndTryToSwapUserFeesOrSendItAsIs(ctx, addr1, fee, types.ModuleName)
	suite.Require().Nil(err)
	suite.Require().NotNil(captured)
	suite.Require().Len(captured, 1)
	suite.Require().Equal(denomBze, captured[0].Denom)
	suite.Require().True(captured[0].Amount.GT(math.ZeroInt()))
}

// TestTrySwap_SwapSuccess_CrossModule verifies that when toModule differs from tradebin,
// swapped coins are transferred from tradebin to the target module.
func (suite *IntegrationTestSuite) TestTrySwap_SwapSuccess_CrossModule() {
	addr1 := sdk.AccAddress("addr1_______________")
	nativeFee := sdk.NewCoin(denomBze, math.NewInt(100000000))
	fee := sdk.NewCoins(nativeFee)
	targetModule := "rewards"

	pool := trySwapDeepPool()
	suite.k.SetLiquidityPool(suite.ctx, pool)

	ctx := suite.ctx.WithValue(keeper.CtxFeeDenomKey, denomStake)

	requiredStake, err := suite.k.CalculateOptimalInputForOutput(pool, nativeFee)
	suite.Require().Nil(err)

	toCapture := sdk.NewCoins(requiredStake)

	suite.bankMock.EXPECT().
		SpendableCoins(gomock.Any(), addr1).
		Times(1).
		Return(sdk.NewCoins(sdk.NewCoin(denomStake, math.NewInt(50_000_000_000))))

	suite.bankMock.EXPECT().
		SendCoinsFromAccountToModule(gomock.Any(), addr1, types.ModuleName, toCapture).
		Times(1).
		Return(nil)

	// collectSwapFee sends treasury portion from tradebin to CpFeeCollector
	suite.bankMock.EXPECT().
		SendCoinsFromModuleToModule(gomock.Any(), types.ModuleName, txfeecollectormoduletypes.CpFeeCollector, gomock.Any()).
		Times(1).
		Return(nil)

	// collectSwapFee sends burner portion from tradebin to burner module
	suite.bankMock.EXPECT().
		SendCoinsFromModuleToModule(gomock.Any(), types.ModuleName, burnermoduletypes.ModuleName, gomock.Any()).
		Times(1).
		Return(nil)

	// Cross-module transfer: swapped coins from tradebin to the target module
	suite.bankMock.EXPECT().
		SendCoinsFromModuleToModule(gomock.Any(), types.ModuleName, targetModule, gomock.Any()).
		Times(1).
		Return(nil)

	captured, err := suite.k.CaptureAndTryToSwapUserFeesOrSendItAsIs(ctx, addr1, fee, targetModule)
	suite.Require().Nil(err)
	suite.Require().NotNil(captured)
	suite.Require().Len(captured, 1)
	suite.Require().Equal(denomBze, captured[0].Denom)
	suite.Require().True(captured[0].Amount.GT(math.ZeroInt()))
}

// --- Zero pool fee edge case ---

// TestTrySwap_ZeroPoolFee_SwapsNormally verifies that when pool.Fee is zero, the "too small" check
// is skipped (pool.Fee.IsPositive() is false) and the swap proceeds even for tiny amounts.
func (suite *IntegrationTestSuite) TestTrySwap_ZeroPoolFee_SwapsNormally() {
	addr1 := sdk.AccAddress("addr1_______________")
	nativeFee := sdk.NewCoin(denomBze, math.NewInt(1))
	fee := sdk.NewCoins(nativeFee)

	pool := trySwapDeepPool()
	pool.Fee = math.LegacyZeroDec()
	suite.k.SetLiquidityPool(suite.ctx, pool)

	ctx := suite.ctx.WithValue(keeper.CtxFeeDenomKey, denomStake)

	requiredStake, err := suite.k.CalculateOptimalInputForOutput(pool, nativeFee)
	suite.Require().Nil(err)

	toCapture := sdk.NewCoins(requiredStake)

	suite.bankMock.EXPECT().
		SpendableCoins(gomock.Any(), addr1).
		Times(1).
		Return(sdk.NewCoins(sdk.NewCoin(denomStake, math.NewInt(50_000_000_000))))

	suite.bankMock.EXPECT().
		SendCoinsFromAccountToModule(gomock.Any(), addr1, types.ModuleName, toCapture).
		Times(1).
		Return(nil)

	// Zero pool fee means collectSwapFee receives a zero fee coin, so no
	// SendCoinsFromModuleToModule calls are made (treasury and burner are both zero).
	// No cross-module transfer either since toModule == types.ModuleName.

	captured, err := suite.k.CaptureAndTryToSwapUserFeesOrSendItAsIs(ctx, addr1, fee, types.ModuleName)
	suite.Require().Nil(err)
	// With zero pool fee, even tiny amounts can be swapped
	suite.Require().NotNil(captured)
	suite.Require().Len(captured, 1)
	suite.Require().Equal(denomBze, captured[0].Denom)
}
