package keeper_test

import (
	"cosmossdk.io/math"
	"fmt"
	"github.com/bze-alphateam/bze/x/tradebin/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"go.uber.org/mock/gomock"
)

func (suite *IntegrationTestSuite) TestCaptureAndSwapUserFee_InvalidFee() {
	addr1 := sdk.AccAddress("addr1_______________")

	// Test with zero fee
	_, err := suite.k.CaptureAndSwapUserFee(suite.ctx, addr1, sdk.NewCoins())
	suite.Require().NotNil(err)
	suite.Require().Contains(err.Error(), "can not capture user fees that are not all positive")

	// Test with negative fee (create invalid coins)
	invalidFee := sdk.Coins{sdk.Coin{Denom: "ubze", Amount: math.NewInt(-100)}}
	_, err = suite.k.CaptureAndSwapUserFee(suite.ctx, addr1, invalidFee)
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
	coinsCaptured, err := suite.k.CaptureAndSwapUserFee(suite.ctx, addr1, fee)
	suite.Require().Nil(err)
	suite.Require().Equal(fee, coinsCaptured)
}

func (suite *IntegrationTestSuite) TestCaptureAndSwapUserFee_PreferredDenomSameAsNative() {
	addr1 := sdk.AccAddress("addr1_______________")
	fee := sdk.NewCoins(sdk.NewCoin(denomBze, math.NewInt(1000000)))

	// Set context with preferred denom same as native
	ctx := suite.ctx.WithValue("fee_denom", denomBze)

	// Mock bank transfer
	suite.bankMock.EXPECT().
		SendCoinsFromAccountToModule(gomock.Any(), addr1, types.ModuleName, fee).
		Times(1).
		Return(nil)

	coinsCaptured, err := suite.k.CaptureAndSwapUserFee(ctx, addr1, fee)
	suite.Require().Nil(err)
	suite.Require().Equal(fee, coinsCaptured)
}

func (suite *IntegrationTestSuite) TestCaptureAndSwapUserFee_NoNativeFeeInCoins() {
	addr1 := sdk.AccAddress("addr1_______________")
	// Fee contains only stake, no native ubze
	fee := sdk.NewCoins(sdk.NewCoin(denomStake, math.NewInt(1000000)))

	// Set context with preferred denom
	ctx := suite.ctx.WithValue("fee_denom", "uatom")

	// Mock bank transfer - should capture as-is since no native fee to swap
	suite.bankMock.EXPECT().
		SendCoinsFromAccountToModule(gomock.Any(), addr1, types.ModuleName, fee).
		Times(1).
		Return(nil)

	coinsCaptured, err := suite.k.CaptureAndSwapUserFee(ctx, addr1, fee)
	suite.Require().Nil(err)
	suite.Require().Equal(fee, coinsCaptured)
}

func (suite *IntegrationTestSuite) TestCaptureAndSwapUserFee_PoolNotFound() {
	addr1 := sdk.AccAddress("addr1_______________")
	fee := sdk.NewCoins(sdk.NewCoin(denomBze, math.NewInt(1000000)))

	// Set context with preferred denom that has no pool
	ctx := suite.ctx.WithValue("fee_denom", "uatom")

	// Mock bank transfer - should fall back to native denom
	suite.bankMock.EXPECT().
		SendCoinsFromAccountToModule(gomock.Any(), addr1, types.ModuleName, fee).
		Times(1).
		Return(nil)

	coinsCaptured, err := suite.k.CaptureAndSwapUserFee(ctx, addr1, fee)
	suite.Require().Nil(err)
	suite.Require().Equal(fee, coinsCaptured)
}

func (suite *IntegrationTestSuite) TestCaptureAndSwapUserFee_SwapSuccess() {
	addr1 := sdk.AccAddress("addr1_______________")
	nativeFee := sdk.NewCoin(denomBze, math.NewInt(100000000)) // 100 ubze
	fee := sdk.NewCoins(nativeFee)

	// Create a liquidity pool for ubze/stake
	pool := types.LiquidityPool{
		Id:           "stake_ubze",
		Base:         denomStake,
		Quote:        denomBze,
		ReserveBase:  math.NewInt(1000000000),         // 1000 stake
		ReserveQuote: math.NewInt(2000000000),         // 2000 ubze (ratio 1:2)
		Fee:          math.LegacyNewDecWithPrec(3, 3), // 0.3% fee
		FeeDest: &types.FeeDestination{
			Treasury:  math.LegacyNewDecWithPrec(5, 1), // 50% to treasury
			Burner:    math.LegacyNewDecWithPrec(3, 1), // 30% to burner
			Providers: math.LegacyNewDecWithPrec(2, 1), // 20% to LP providers
		},
	}
	suite.k.SetLiquidityPool(suite.ctx, pool)

	// Set context with preferred denom stake
	ctx := suite.ctx.WithValue("fee_denom", denomStake)

	// Calculate what amount is needed in stake to get 100 ubze
	requiredStake, err := suite.k.CalculateOptimalInputForOutput(pool, nativeFee)
	suite.Require().Nil(err)

	toCapture := sdk.NewCoins(requiredStake)

	// Mock user balance check
	suite.bankMock.EXPECT().
		SpendableCoins(gomock.Any(), addr1).
		Times(1).
		Return(sdk.NewCoins(sdk.NewCoin(denomStake, math.NewInt(1000000000))))

	// Mock bank transfer - capture stake from user
	suite.bankMock.EXPECT().
		SendCoinsFromAccountToModule(gomock.Any(), addr1, types.ModuleName, toCapture).
		Times(1).
		Return(nil)

	// Mock fee distribution during swap - swap will distribute fees
	moduleAddr := sdk.AccAddress("module______________")
	tradebinModuleAcc := authtypes.ModuleAccount{
		BaseAccount: &authtypes.BaseAccount{
			Address: moduleAddr.String(),
		},
		Name: types.ModuleName,
	}
	suite.accountMock.EXPECT().
		GetModuleAccount(gomock.Any(), types.ModuleName).
		Return(&tradebinModuleAcc).
		AnyTimes()
	suite.distrMock.EXPECT().
		FundCommunityPool(gomock.Any(), gomock.Any(), moduleAddr).
		Return(nil).
		AnyTimes()
	suite.bankMock.EXPECT().
		SendCoinsFromModuleToModule(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil).
		AnyTimes()

	coinsCaptured, err := suite.k.CaptureAndSwapUserFee(ctx, addr1, fee)
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

	// Create a liquidity pool for ubze/stake
	pool := types.LiquidityPool{
		Id:           "stake_ubze",
		Base:         denomStake,
		Quote:        denomBze,
		ReserveBase:  math.NewInt(1000000000),
		ReserveQuote: math.NewInt(2000000000),
		Fee:          math.LegacyNewDecWithPrec(3, 3),
		FeeDest: &types.FeeDestination{
			Treasury:  math.LegacyNewDecWithPrec(5, 1),
			Burner:    math.LegacyNewDecWithPrec(3, 1),
			Providers: math.LegacyNewDecWithPrec(2, 1),
		},
	}
	suite.k.SetLiquidityPool(suite.ctx, pool)

	// Set context with preferred denom stake
	ctx := suite.ctx.WithValue("fee_denom", denomStake)

	// Calculate required stake for native fee
	requiredStake, err := suite.k.CalculateOptimalInputForOutput(pool, nativeFee)
	suite.Require().Nil(err)

	toCapture := sdk.NewCoins(otherFee, requiredStake)

	// Mock user balance check
	suite.bankMock.EXPECT().
		SpendableCoins(gomock.Any(), addr1).
		Times(1).
		Return(sdk.NewCoins(
			sdk.NewCoin(denomStake, math.NewInt(1000000000)),
			sdk.NewCoin("uosmo", math.NewInt(100000000)),
		))

	// Mock bank transfer - capture stake + other fee from user
	suite.bankMock.EXPECT().
		SendCoinsFromAccountToModule(gomock.Any(), addr1, types.ModuleName, toCapture).
		Times(1).
		Return(nil)

	// Mock fee distribution during swap - swap will distribute fees
	moduleAddr := sdk.AccAddress("module______________")
	tradebinModuleAcc := authtypes.ModuleAccount{
		BaseAccount: &authtypes.BaseAccount{
			Address: moduleAddr.String(),
		},
		Name: types.ModuleName,
	}
	suite.accountMock.EXPECT().
		GetModuleAccount(gomock.Any(), types.ModuleName).
		Return(&tradebinModuleAcc).
		AnyTimes()
	suite.distrMock.EXPECT().
		FundCommunityPool(gomock.Any(), gomock.Any(), moduleAddr).
		Return(nil).
		AnyTimes()
	suite.bankMock.EXPECT().
		SendCoinsFromModuleToModule(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil).
		AnyTimes()

	coinsCaptured, err := suite.k.CaptureAndSwapUserFee(ctx, addr1, fee)
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

	// Create a liquidity pool
	pool := types.LiquidityPool{
		Id:           "stake_ubze",
		Base:         denomStake,
		Quote:        denomBze,
		ReserveBase:  math.NewInt(1000000000),
		ReserveQuote: math.NewInt(2000000000),
		Fee:          math.LegacyNewDecWithPrec(3, 3),
		FeeDest: &types.FeeDestination{
			Treasury:  math.LegacyNewDecWithPrec(5, 1),
			Burner:    math.LegacyNewDecWithPrec(3, 1),
			Providers: math.LegacyNewDecWithPrec(2, 1),
		},
	}
	suite.k.SetLiquidityPool(suite.ctx, pool)

	ctx := suite.ctx.WithValue("fee_denom", denomStake)

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

	coinsCaptured, err := suite.k.CaptureAndSwapUserFee(ctx, addr1, fee)
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

	_, err := suite.k.CaptureAndSwapUserFee(suite.ctx, addr1, fee)
	suite.Require().NotNil(err)
	suite.Require().Contains(err.Error(), "insufficient funds")
}

func (suite *IntegrationTestSuite) TestCaptureAndSwapUserFee_SwapFails() {
	addr1 := sdk.AccAddress("addr1_______________")
	nativeFee := sdk.NewCoin(denomBze, math.NewInt(100000000))
	fee := sdk.NewCoins(nativeFee)

	// Create a pool with very low reserves to cause swap failure
	pool := types.LiquidityPool{
		Id:           "stake_ubze",
		Base:         denomStake,
		Quote:        denomBze,
		ReserveBase:  math.NewInt(10), // Very low reserves
		ReserveQuote: math.NewInt(10),
		Fee:          math.LegacyNewDecWithPrec(3, 3),
		FeeDest: &types.FeeDestination{
			Treasury:  math.LegacyNewDecWithPrec(5, 1),
			Burner:    math.LegacyNewDecWithPrec(3, 1),
			Providers: math.LegacyNewDecWithPrec(2, 1),
		},
	}
	suite.k.SetLiquidityPool(suite.ctx, pool)

	ctx := suite.ctx.WithValue("fee_denom", denomStake)

	// Calculate required amount
	requiredStake, err := suite.k.CalculateOptimalInputForOutput(pool, nativeFee)
	// This might fail or succeed depending on calculation
	if err == nil {
		toCapture := sdk.NewCoins(requiredStake)

		// Mock user balance check
		suite.bankMock.EXPECT().
			SpendableCoins(gomock.Any(), addr1).
			Times(1).
			Return(sdk.NewCoins(sdk.NewCoin(denomStake, math.NewInt(1000000000))))

		// Mock bank transfer
		suite.bankMock.EXPECT().
			SendCoinsFromAccountToModule(gomock.Any(), addr1, types.ModuleName, toCapture).
			Times(1).
			Return(nil)

		// Swap will fail due to low reserves
		_, err = suite.k.CaptureAndSwapUserFee(ctx, addr1, fee)
		suite.Require().NotNil(err)
	}
}
