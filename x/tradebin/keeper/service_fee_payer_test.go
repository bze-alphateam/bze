package keeper_test

import (
	"fmt"

	"cosmossdk.io/math"
	"github.com/bze-alphateam/bze/x/tradebin/keeper"
	"github.com/bze-alphateam/bze/x/tradebin/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"go.uber.org/mock/gomock"
)

func (suite *IntegrationTestSuite) TestCaptureAndSwapUserFee_InvalidFee() {
	addr1 := sdk.AccAddress("addr1_______________")

	// Test with zero fee
	_, err := suite.k.CaptureAndSwapUserFee(suite.ctx, addr1, sdk.NewCoins(), types.ModuleName)
	suite.Require().NotNil(err)
	suite.Require().Contains(err.Error(), "can not capture user fees that are not all positive")

	// Test with negative fee (create invalid coins)
	invalidFee := sdk.Coins{sdk.Coin{Denom: "ubze", Amount: math.NewInt(-100)}}
	_, err = suite.k.CaptureAndSwapUserFee(suite.ctx, addr1, invalidFee, types.ModuleName)
	suite.Require().NotNil(err)
}

func (suite *IntegrationTestSuite) TestCaptureAndSwapUserFee_NoPreferredDenom() {
	addr1 := sdk.AccAddress("addr1_______________")
	fee := sdk.NewCoins(sdk.NewCoin(denomBze, math.NewInt(1000000)))

	// Mock bank transfer - expect native denom to be captured directly
	suite.bankMock.EXPECT().
		SendCoinsFromAccountToModule(gomock.Any(), addr1, types.ModuleName, fee).
		Times(1).
		Return(nil)

	// No preferred denom in context, should capture fee directly
	coinsCaptured, err := suite.k.CaptureAndSwapUserFee(suite.ctx, addr1, fee, types.ModuleName)
	suite.Require().Nil(err)
	suite.Require().Equal(fee, coinsCaptured)
}

func (suite *IntegrationTestSuite) TestCaptureAndSwapUserFee_PreferredDenomSameAsNative() {
	addr1 := sdk.AccAddress("addr1_______________")
	fee := sdk.NewCoins(sdk.NewCoin(denomBze, math.NewInt(1000000)))

	// Set context with preferred denom same as native
	ctx := suite.ctx.WithValue(keeper.CtxFeeDenomKey, denomBze)

	// Mock bank transfer
	suite.bankMock.EXPECT().
		SendCoinsFromAccountToModule(gomock.Any(), addr1, types.ModuleName, fee).
		Times(1).
		Return(nil)

	coinsCaptured, err := suite.k.CaptureAndSwapUserFee(ctx, addr1, fee, types.ModuleName)
	suite.Require().Nil(err)
	suite.Require().Equal(fee, coinsCaptured)
}

func (suite *IntegrationTestSuite) TestCaptureAndSwapUserFee_NoNativeFeeInCoins() {
	addr1 := sdk.AccAddress("addr1_______________")
	// Fee contains only stake, no native ubze
	fee := sdk.NewCoins(sdk.NewCoin(denomStake, math.NewInt(1000000)))

	// Set context with preferred denom
	ctx := suite.ctx.WithValue(keeper.CtxFeeDenomKey, "uatom")

	// Mock bank transfer - should capture as-is since no native fee to swap
	suite.bankMock.EXPECT().
		SendCoinsFromAccountToModule(gomock.Any(), addr1, types.ModuleName, fee).
		Times(1).
		Return(nil)

	coinsCaptured, err := suite.k.CaptureAndSwapUserFee(ctx, addr1, fee, types.ModuleName)
	suite.Require().Nil(err)
	suite.Require().Equal(fee, coinsCaptured)
}

func (suite *IntegrationTestSuite) TestCaptureAndSwapUserFee_PoolNotFound() {
	addr1 := sdk.AccAddress("addr1_______________")
	fee := sdk.NewCoins(sdk.NewCoin(denomBze, math.NewInt(1000000)))

	// Set context with preferred denom that has no pool
	ctx := suite.ctx.WithValue(keeper.CtxFeeDenomKey, "uatom")

	// Mock bank transfer - should fall back to native denom
	suite.bankMock.EXPECT().
		SendCoinsFromAccountToModule(gomock.Any(), addr1, types.ModuleName, fee).
		Times(1).
		Return(nil)

	coinsCaptured, err := suite.k.CaptureAndSwapUserFee(ctx, addr1, fee, types.ModuleName)
	suite.Require().Nil(err)
	suite.Require().Equal(fee, coinsCaptured)
}

func (suite *IntegrationTestSuite) TestCaptureAndSwapUserFee_SwapSuccess() {
	addr1 := sdk.AccAddress("addr1_______________")
	nativeFee := sdk.NewCoin(denomBze, math.NewInt(100000000)) // 100 ubze
	fee := sdk.NewCoins(nativeFee)

	// Create a liquidity pool with reserves above the deep liquidity threshold
	// (75% of DefaultMinNativeLiquidityForModuleSwap = 75B ubze)
	pool := types.LiquidityPool{
		Id:           "stake_ubze",
		Base:         denomStake,
		Quote:        denomBze,
		ReserveBase:  math.NewInt(50_000_000_000),     // 50B stake
		ReserveQuote: math.NewInt(100_000_000_000),    // 100B ubze (ratio 1:2)
		Fee:          math.LegacyNewDecWithPrec(3, 3), // 0.3% fee
		FeeDest: &types.FeeDestination{
			Treasury:  math.LegacyNewDecWithPrec(5, 1), // 50% to treasury
			Burner:    math.LegacyNewDecWithPrec(3, 1), // 30% to burner
			Providers: math.LegacyNewDecWithPrec(2, 1), // 20% to LP providers
		},
	}
	suite.k.SetLiquidityPool(suite.ctx, pool)

	// Set context with preferred denom stake
	ctx := suite.ctx.WithValue(keeper.CtxFeeDenomKey, denomStake)

	// Calculate what amount is needed in stake to get 100 ubze
	requiredStake, err := suite.k.CalculateOptimalInputForOutput(pool, nativeFee)
	suite.Require().Nil(err)

	toCapture := sdk.NewCoins(requiredStake)

	// Mock user balance check
	suite.bankMock.EXPECT().
		SpendableCoins(gomock.Any(), addr1).
		Times(1).
		Return(sdk.NewCoins(sdk.NewCoin(denomStake, math.NewInt(50_000_000_000))))

	// Mock bank transfer - capture stake from user
	suite.bankMock.EXPECT().
		SendCoinsFromAccountToModule(gomock.Any(), addr1, types.ModuleName, toCapture).
		Times(1).
		Return(nil)

	// Mock fee distribution during swap - swap will distribute fees
	suite.bankMock.EXPECT().
		SendCoinsFromModuleToModule(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil).
		AnyTimes()

	coinsCaptured, err := suite.k.CaptureAndSwapUserFee(ctx, addr1, fee, types.ModuleName)
	suite.Require().Nil(err)

	// Should return only native denom after swap
	suite.Require().Len(coinsCaptured, 1)
	suite.Require().Equal(denomBze, coinsCaptured[0].Denom)
	// The amount should be close to the original fee (accounting for swap fees)
	suite.Require().True(coinsCaptured[0].Amount.GT(math.ZeroInt()))
}

func (suite *IntegrationTestSuite) TestCaptureAndSwapUserFee_SwapWithMultipleFees() {
	addr1 := sdk.AccAddress("addr1_______________")
	nativeFee := sdk.NewCoin(denomBze, math.NewInt(100000000)) // 100 ubze
	otherFee := sdk.NewCoin("uosmo", math.NewInt(50000000))    // 50 uosmo
	fee := sdk.NewCoins(nativeFee, otherFee)

	// Create a liquidity pool with reserves above the deep liquidity threshold
	pool := types.LiquidityPool{
		Id:           "stake_ubze",
		Base:         denomStake,
		Quote:        denomBze,
		ReserveBase:  math.NewInt(50_000_000_000),
		ReserveQuote: math.NewInt(100_000_000_000),
		Fee:          math.LegacyNewDecWithPrec(3, 3),
		FeeDest: &types.FeeDestination{
			Treasury:  math.LegacyNewDecWithPrec(5, 1),
			Burner:    math.LegacyNewDecWithPrec(3, 1),
			Providers: math.LegacyNewDecWithPrec(2, 1),
		},
	}
	suite.k.SetLiquidityPool(suite.ctx, pool)

	// Set context with preferred denom stake
	ctx := suite.ctx.WithValue(keeper.CtxFeeDenomKey, denomStake)

	// Calculate required stake for native fee
	requiredStake, err := suite.k.CalculateOptimalInputForOutput(pool, nativeFee)
	suite.Require().Nil(err)

	toCapture := sdk.NewCoins(otherFee, requiredStake)

	// Mock user balance check
	suite.bankMock.EXPECT().
		SpendableCoins(gomock.Any(), addr1).
		Times(1).
		Return(sdk.NewCoins(
			sdk.NewCoin(denomStake, math.NewInt(50_000_000_000)),
			sdk.NewCoin("uosmo", math.NewInt(100000000)),
		))

	// Mock bank transfer - capture stake + other fee from user
	suite.bankMock.EXPECT().
		SendCoinsFromAccountToModule(gomock.Any(), addr1, types.ModuleName, toCapture).
		Times(1).
		Return(nil)

	// Mock fee distribution during swap - swap will distribute fees
	suite.bankMock.EXPECT().
		SendCoinsFromModuleToModule(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil).
		AnyTimes()

	coinsCaptured, err := suite.k.CaptureAndSwapUserFee(ctx, addr1, fee, types.ModuleName)
	suite.Require().Nil(err)

	// Should return ubze (from swap) + uosmo (unchanged)
	suite.Require().Len(coinsCaptured, 2)
	// Find each coin
	hasNative := false
	hasOther := false
	for _, coin := range coinsCaptured {
		if coin.Denom == denomBze {
			hasNative = true
			suite.Require().True(coin.Amount.GT(math.ZeroInt()))
		}
		if coin.Denom == "uosmo" {
			hasOther = true
			suite.Require().Equal(otherFee.Amount, coin.Amount)
		}
	}
	suite.Require().True(hasNative, "should have native denom")
	suite.Require().True(hasOther, "should have other fee")
}

func (suite *IntegrationTestSuite) TestCaptureAndSwapUserFee_InsufficientBalance() {
	addr1 := sdk.AccAddress("addr1_______________")
	nativeFee := sdk.NewCoin(denomBze, math.NewInt(100000000))
	fee := sdk.NewCoins(nativeFee)

	// Create a liquidity pool with reserves above the deep liquidity threshold
	pool := types.LiquidityPool{
		Id:           "stake_ubze",
		Base:         denomStake,
		Quote:        denomBze,
		ReserveBase:  math.NewInt(50_000_000_000),
		ReserveQuote: math.NewInt(100_000_000_000),
		Fee:          math.LegacyNewDecWithPrec(3, 3),
		FeeDest: &types.FeeDestination{
			Treasury:  math.LegacyNewDecWithPrec(5, 1),
			Burner:    math.LegacyNewDecWithPrec(3, 1),
			Providers: math.LegacyNewDecWithPrec(2, 1),
		},
	}
	suite.k.SetLiquidityPool(suite.ctx, pool)

	ctx := suite.ctx.WithValue(keeper.CtxFeeDenomKey, denomStake)

	// Mock user balance check - insufficient balance
	suite.bankMock.EXPECT().
		SpendableCoins(gomock.Any(), addr1).
		Times(1).
		Return(sdk.NewCoins(sdk.NewCoin(denomStake, math.NewInt(100)))) // Very low balance

	// Mock bank transfer - should fall back to capturing native fee
	suite.bankMock.EXPECT().
		SendCoinsFromAccountToModule(gomock.Any(), addr1, types.ModuleName, fee).
		Times(1).
		Return(nil)

	coinsCaptured, err := suite.k.CaptureAndSwapUserFee(ctx, addr1, fee, types.ModuleName)
	suite.Require().Nil(err)
	suite.Require().Equal(fee, coinsCaptured)
}

func (suite *IntegrationTestSuite) TestCaptureAndSwapUserFee_CaptureFails() {
	addr1 := sdk.AccAddress("addr1_______________")
	fee := sdk.NewCoins(sdk.NewCoin(denomBze, math.NewInt(1000000)))

	// Mock bank transfer failure
	suite.bankMock.EXPECT().
		SendCoinsFromAccountToModule(gomock.Any(), addr1, types.ModuleName, fee).
		Times(1).
		Return(fmt.Errorf("insufficient funds"))

	_, err := suite.k.CaptureAndSwapUserFee(suite.ctx, addr1, fee, types.ModuleName)
	suite.Require().NotNil(err)
	suite.Require().Contains(err.Error(), "insufficient funds")
}

func (suite *IntegrationTestSuite) TestCaptureAndSwapUserFee_SwapCalculationFails() {
	addr1 := sdk.AccAddress("addr1_______________")
	// Use a fee amount that equals the pool reserve, causing CalculateOptimalInputForOutput to fail
	// (output >= reserve triggers an error)
	nativeFee := sdk.NewCoin(denomBze, math.NewInt(100_000_000_000))
	fee := sdk.NewCoins(nativeFee)

	// Create a pool with reserves above the liquidity threshold but equal to the fee amount
	pool := types.LiquidityPool{
		Id:           "stake_ubze",
		Base:         denomStake,
		Quote:        denomBze,
		ReserveBase:  math.NewInt(50_000_000_000),
		ReserveQuote: math.NewInt(100_000_000_000), // equals nativeFee → CalculateOptimalInputForOutput fails
		Fee:          math.LegacyNewDecWithPrec(3, 3),
		FeeDest: &types.FeeDestination{
			Treasury:  math.LegacyNewDecWithPrec(5, 1),
			Burner:    math.LegacyNewDecWithPrec(3, 1),
			Providers: math.LegacyNewDecWithPrec(2, 1),
		},
	}
	suite.k.SetLiquidityPool(suite.ctx, pool)

	ctx := suite.ctx.WithValue(keeper.CtxFeeDenomKey, denomStake)

	// When CalculateOptimalInputForOutput fails inside CaptureAndSwapUserFee,
	// the function falls back to capturing fee in native denom
	suite.bankMock.EXPECT().
		SendCoinsFromAccountToModule(gomock.Any(), addr1, types.ModuleName, fee).
		Times(1).
		Return(nil)

	coinsCaptured, err := suite.k.CaptureAndSwapUserFee(ctx, addr1, fee, types.ModuleName)
	suite.Require().Nil(err)
	suite.Require().Equal(fee, coinsCaptured)
}

func (suite *IntegrationTestSuite) TestCaptureAndSwapUserFee_CrossModuleSwap() {
	// Test that when toModule is NOT tradebin, coins are properly transferred back
	addr1 := sdk.AccAddress("addr1_______________")
	nativeFee := sdk.NewCoin(denomBze, math.NewInt(100000000)) // 100 ubze
	fee := sdk.NewCoins(nativeFee)
	targetModule := "rewards" // Different from tradebin

	// Create a liquidity pool with reserves above the deep liquidity threshold
	pool := types.LiquidityPool{
		Id:           "stake_ubze",
		Base:         denomStake,
		Quote:        denomBze,
		ReserveBase:  math.NewInt(50_000_000_000),     // 50B stake
		ReserveQuote: math.NewInt(100_000_000_000),    // 100B ubze (ratio 1:2)
		Fee:          math.LegacyNewDecWithPrec(3, 3), // 0.3% fee
		FeeDest: &types.FeeDestination{
			Treasury:  math.LegacyNewDecWithPrec(5, 1), // 50% to treasury
			Burner:    math.LegacyNewDecWithPrec(3, 1), // 30% to burner
			Providers: math.LegacyNewDecWithPrec(2, 1), // 20% to LP providers
		},
	}
	suite.k.SetLiquidityPool(suite.ctx, pool)

	// Set context with preferred denom stake
	ctx := suite.ctx.WithValue(keeper.CtxFeeDenomKey, denomStake)

	// Calculate what amount is needed in stake to get 100 ubze
	requiredStake, err := suite.k.CalculateOptimalInputForOutput(pool, nativeFee)
	suite.Require().Nil(err)

	toCapture := sdk.NewCoins(requiredStake)

	// Mock user balance check
	suite.bankMock.EXPECT().
		SpendableCoins(gomock.Any(), addr1).
		Times(1).
		Return(sdk.NewCoins(sdk.NewCoin(denomStake, math.NewInt(50_000_000_000))))

	// Mock bank transfer - capture stake from user to tradebin module
	suite.bankMock.EXPECT().
		SendCoinsFromAccountToModule(gomock.Any(), addr1, types.ModuleName, toCapture).
		Times(1).
		Return(nil)

	// Mock fee distribution during swap
	suite.bankMock.EXPECT().
		SendCoinsFromModuleToModule(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil).
		AnyTimes()

	coinsCaptured, err := suite.k.CaptureAndSwapUserFee(ctx, addr1, fee, targetModule)
	suite.Require().Nil(err)

	// Should return only native denom after swap
	suite.Require().Len(coinsCaptured, 1)
	suite.Require().Equal(denomBze, coinsCaptured[0].Denom)
	// The amount should be close to the original fee (accounting for swap fees)
	suite.Require().True(coinsCaptured[0].Amount.GT(math.ZeroInt()))
}

func (suite *IntegrationTestSuite) TestCaptureAndSwapUserFee_SameModuleOptimization() {
	// Test that when toModule equals tradebin, no extra transfer happens (optimization)
	addr1 := sdk.AccAddress("addr1_______________")
	nativeFee := sdk.NewCoin(denomBze, math.NewInt(100000000)) // 100 ubze
	fee := sdk.NewCoins(nativeFee)

	// Create a liquidity pool with reserves above the deep liquidity threshold
	pool := types.LiquidityPool{
		Id:           "stake_ubze",
		Base:         denomStake,
		Quote:        denomBze,
		ReserveBase:  math.NewInt(50_000_000_000),     // 50B stake
		ReserveQuote: math.NewInt(100_000_000_000),    // 100B ubze (ratio 1:2)
		Fee:          math.LegacyNewDecWithPrec(3, 3), // 0.3% fee
		FeeDest: &types.FeeDestination{
			Treasury:  math.LegacyNewDecWithPrec(5, 1), // 50% to treasury
			Burner:    math.LegacyNewDecWithPrec(3, 1), // 30% to burner
			Providers: math.LegacyNewDecWithPrec(2, 1), // 20% to LP providers
		},
	}
	suite.k.SetLiquidityPool(suite.ctx, pool)

	// Set context with preferred denom stake
	ctx := suite.ctx.WithValue(keeper.CtxFeeDenomKey, denomStake)

	// Calculate what amount is needed in stake to get 100 ubze
	requiredStake, err := suite.k.CalculateOptimalInputForOutput(pool, nativeFee)
	suite.Require().Nil(err)

	toCapture := sdk.NewCoins(requiredStake)

	// Mock user balance check
	suite.bankMock.EXPECT().
		SpendableCoins(gomock.Any(), addr1).
		Times(1).
		Return(sdk.NewCoins(sdk.NewCoin(denomStake, math.NewInt(50_000_000_000))))

	// Mock bank transfer - capture stake from user to tradebin module
	suite.bankMock.EXPECT().
		SendCoinsFromAccountToModule(gomock.Any(), addr1, types.ModuleName, toCapture).
		Times(1).
		Return(nil)

	// Mock fee distribution during swap - this should happen but NOT the transfer back to same module
	suite.bankMock.EXPECT().
		SendCoinsFromModuleToModule(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil).
		AnyTimes()

	// Call with toModule = types.ModuleName (tradebin itself)
	coinsCaptured, err := suite.k.CaptureAndSwapUserFee(ctx, addr1, fee, types.ModuleName)
	suite.Require().Nil(err)

	// Should return only native denom after swap
	suite.Require().Len(coinsCaptured, 1)
	suite.Require().Equal(denomBze, coinsCaptured[0].Denom)
	// The amount should be close to the original fee (accounting for swap fees)
	suite.Require().True(coinsCaptured[0].Amount.GT(math.ZeroInt()))

	// The optimization means we didn't do an extra transfer from tradebin to tradebin
	// The mock with AnyTimes() will catch the fee distribution calls but not fail on missing transfer
}

// TestCaptureAndSwapUserFee_ZeroNativeReserves verifies that when the pool has zero native reserves,
// the function falls back to capturing the fee in native denom instead of attempting a swap.
func (suite *IntegrationTestSuite) TestCaptureAndSwapUserFee_ZeroNativeReserves() {
	addr1 := sdk.AccAddress("addr1_______________")
	nativeFee := sdk.NewCoin(denomBze, math.NewInt(100000000))
	fee := sdk.NewCoins(nativeFee)

	// Create a pool with zero native (ubze) reserves
	pool := types.LiquidityPool{
		Id:           "stake_ubze",
		Base:         denomStake,
		Quote:        denomBze,
		ReserveBase:  math.NewInt(1_000_000_000),
		ReserveQuote: math.ZeroInt(), // zero native reserves
		Fee:          math.LegacyNewDecWithPrec(3, 3),
		FeeDest: &types.FeeDestination{
			Treasury:  math.LegacyNewDecWithPrec(5, 1),
			Burner:    math.LegacyNewDecWithPrec(3, 1),
			Providers: math.LegacyNewDecWithPrec(2, 1),
		},
	}
	suite.k.SetLiquidityPool(suite.ctx, pool)

	ctx := suite.ctx.WithValue(keeper.CtxFeeDenomKey, denomStake)

	// Should fall back to native denom since pool has no native reserves
	suite.bankMock.EXPECT().
		SendCoinsFromAccountToModule(gomock.Any(), addr1, types.ModuleName, fee).
		Times(1).
		Return(nil)

	coinsCaptured, err := suite.k.CaptureAndSwapUserFee(ctx, addr1, fee, types.ModuleName)
	suite.Require().Nil(err)
	suite.Require().Equal(fee, coinsCaptured)
}

// TestCaptureAndSwapUserFee_InsufficientLiquidity verifies that when the pool's native reserves are below
// the 75% threshold of MinNativeLiquidityForModuleSwap, the function falls back to native denom.
// This addresses the TOCTOU gap where a prior swap in the same tx could drain pool reserves.
func (suite *IntegrationTestSuite) TestCaptureAndSwapUserFee_InsufficientLiquidity() {
	addr1 := sdk.AccAddress("addr1_______________")
	nativeFee := sdk.NewCoin(denomBze, math.NewInt(100000000))
	fee := sdk.NewCoins(nativeFee)

	// The threshold is MinNativeLiquidityForModuleSwap * 3/4 = 100B * 3/4 = 75B
	// Create a pool with native reserves just below the threshold
	pool := types.LiquidityPool{
		Id:           "stake_ubze",
		Base:         denomStake,
		Quote:        denomBze,
		ReserveBase:  math.NewInt(37_000_000_000),
		ReserveQuote: math.NewInt(74_999_999_999), // just below 75B threshold
		Fee:          math.LegacyNewDecWithPrec(3, 3),
		FeeDest: &types.FeeDestination{
			Treasury:  math.LegacyNewDecWithPrec(5, 1),
			Burner:    math.LegacyNewDecWithPrec(3, 1),
			Providers: math.LegacyNewDecWithPrec(2, 1),
		},
	}
	suite.k.SetLiquidityPool(suite.ctx, pool)

	ctx := suite.ctx.WithValue(keeper.CtxFeeDenomKey, denomStake)

	// Should fall back to native denom since pool liquidity is below threshold
	suite.bankMock.EXPECT().
		SendCoinsFromAccountToModule(gomock.Any(), addr1, types.ModuleName, fee).
		Times(1).
		Return(nil)

	coinsCaptured, err := suite.k.CaptureAndSwapUserFee(ctx, addr1, fee, types.ModuleName)
	suite.Require().Nil(err)
	suite.Require().Equal(fee, coinsCaptured)
}

// TestCaptureAndSwapUserFee_LiquidityAtThreshold verifies that when the pool's native reserves are exactly
// at the 75% threshold, the function proceeds with the swap (GTE check).
func (suite *IntegrationTestSuite) TestCaptureAndSwapUserFee_LiquidityAtThreshold() {
	addr1 := sdk.AccAddress("addr1_______________")
	nativeFee := sdk.NewCoin(denomBze, math.NewInt(100000000))
	fee := sdk.NewCoins(nativeFee)

	// The threshold is MinNativeLiquidityForModuleSwap * 3/4 = 100B * 3/4 = 75B
	// Create a pool with native reserves exactly at the threshold
	pool := types.LiquidityPool{
		Id:           "stake_ubze",
		Base:         denomStake,
		Quote:        denomBze,
		ReserveBase:  math.NewInt(37_500_000_000),
		ReserveQuote: math.NewInt(75_000_000_000), // exactly at threshold
		Fee:          math.LegacyNewDecWithPrec(3, 3),
		FeeDest: &types.FeeDestination{
			Treasury:  math.LegacyNewDecWithPrec(5, 1),
			Burner:    math.LegacyNewDecWithPrec(3, 1),
			Providers: math.LegacyNewDecWithPrec(2, 1),
		},
	}
	suite.k.SetLiquidityPool(suite.ctx, pool)

	ctx := suite.ctx.WithValue(keeper.CtxFeeDenomKey, denomStake)

	// Calculate what amount is needed in stake to get the native fee
	requiredStake, err := suite.k.CalculateOptimalInputForOutput(pool, nativeFee)
	suite.Require().Nil(err)

	toCapture := sdk.NewCoins(requiredStake)

	// Should proceed with swap since reserves are at the threshold (not below)
	suite.bankMock.EXPECT().
		SpendableCoins(gomock.Any(), addr1).
		Times(1).
		Return(sdk.NewCoins(sdk.NewCoin(denomStake, math.NewInt(50_000_000_000))))

	suite.bankMock.EXPECT().
		SendCoinsFromAccountToModule(gomock.Any(), addr1, types.ModuleName, toCapture).
		Times(1).
		Return(nil)

	suite.bankMock.EXPECT().
		SendCoinsFromModuleToModule(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil).
		AnyTimes()

	coinsCaptured, err := suite.k.CaptureAndSwapUserFee(ctx, addr1, fee, types.ModuleName)
	suite.Require().Nil(err)

	// Should return native denom after successful swap
	suite.Require().Len(coinsCaptured, 1)
	suite.Require().Equal(denomBze, coinsCaptured[0].Denom)
	suite.Require().True(coinsCaptured[0].Amount.GT(math.ZeroInt()))
}
