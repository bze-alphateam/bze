package keeper_test

import (
	"errors"
	"fmt"

	"cosmossdk.io/math"
	"github.com/bze-alphateam/bze/x/tradebin/types"
	v2types "github.com/bze-alphateam/bze/x/tradebin/v2types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"go.uber.org/mock/gomock"
)

func denomTestParams(nativeDenom string) v2types.Params {
	p := v2types.DefaultParams()
	p.NativeDenom = nativeDenom
	return p
}

func denomTestParamsWithLiquidity(nativeDenom string, minLiquidity math.Int) v2types.Params {
	p := v2types.DefaultParams()
	p.NativeDenom = nativeDenom
	p.MinNativeLiquidityForModuleSwap = minLiquidity
	return p
}

func (suite *IntegrationTestSuite) TestServiceDenom_IsNativeDenom_True() {
	nativeDenom := "ubze"
	params := denomTestParams(nativeDenom)
	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	result := suite.k.IsNativeDenom(suite.ctx, nativeDenom)
	suite.Require().True(result)
}

func (suite *IntegrationTestSuite) TestServiceDenom_IsNativeDenom_False() {
	nativeDenom := "ubze"
	params := denomTestParams(nativeDenom)
	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	result := suite.k.IsNativeDenom(suite.ctx, "uother")
	suite.Require().False(result)
}

func (suite *IntegrationTestSuite) TestServiceDenom_CanSwapForNativeDenom_SameDenom() {
	nativeDenom := "ubze"
	params := denomTestParams(nativeDenom)
	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	// Should return false when trying to swap native denom for itself
	result := suite.k.CanSwapForNativeDenom(suite.ctx, sdk.NewInt64Coin(nativeDenom, 1000))
	suite.Require().False(result)
}

func (suite *IntegrationTestSuite) TestServiceDenom_CanSwapForNativeDenom_PoolNotExists() {
	nativeDenom := "ubze"
	otherDenom := "uother"
	params := denomTestParams(nativeDenom)
	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	// Pool doesn't exist - should return false
	result := suite.k.CanSwapForNativeDenom(suite.ctx, sdk.NewInt64Coin(otherDenom, 1000))
	suite.Require().False(result)
}

func (suite *IntegrationTestSuite) TestServiceDenom_CanSwapForNativeDenom_InsufficientLiquidity() {
	nativeDenom := "ubze"
	otherDenom := "uother"
	params := denomTestParamsWithLiquidity(nativeDenom, math.NewInt(2_000_000_000))
	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	// Create pool with insufficient native liquidity
	pool := types.LiquidityPool{
		Id:           "ubze_uother",
		Base:         nativeDenom,
		Quote:        otherDenom,
		ReserveBase:  math.NewInt(1_000_000_000), // Less than minNativeAmountForSwap (50B)
		ReserveQuote: math.NewInt(1_000_000_000),
	}
	suite.k.SetLiquidityPool(suite.ctx, pool)

	result := suite.k.CanSwapForNativeDenom(suite.ctx, sdk.NewInt64Coin(otherDenom, 1000))
	suite.Require().False(result)
}

func (suite *IntegrationTestSuite) TestServiceDenom_CanSwapForNativeDenom_SufficientLiquidity() {
	nativeDenom := "ubze"
	otherDenom := "uother"
	params := denomTestParams(nativeDenom)
	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	// Create pool with sufficient native liquidity
	pool := types.LiquidityPool{
		Id:           "ubze_uother",
		Base:         nativeDenom,
		Quote:        otherDenom,
		ReserveBase:  math.NewInt(100_000_000_000), // Greater than minNativeAmountForSwap (50B)
		ReserveQuote: math.NewInt(1_000_000_000),
	}
	suite.k.SetLiquidityPool(suite.ctx, pool)

	result := suite.k.CanSwapForNativeDenom(suite.ctx, sdk.NewInt64Coin(otherDenom, 1000))
	suite.Require().True(result)
}

func (suite *IntegrationTestSuite) TestServiceDenom_ModuleSwapForNativeDenom_EmptyNativeDenom() {
	// Set empty native denom
	params := denomTestParams("")
	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	coins := sdk.NewCoins(sdk.NewInt64Coin("uother", 1000))

	result, err := suite.k.ModuleSwapForNativeDenom(suite.ctx, "test_module", coins)
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "native denom not set")
	suite.Require().Equal(sdk.Coin{}, result, "should return zero coin on error")
}

func (suite *IntegrationTestSuite) TestServiceDenom_ModuleSwapForNativeDenom_SwapNativeToNative() {
	nativeDenom := "ubze"
	params := denomTestParams(nativeDenom)
	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	// Mock module account
	addr := sdk.AccAddress("moduleacc")
	moduleAcc := authtypes.ModuleAccount{
		BaseAccount: &authtypes.BaseAccount{
			Address: addr.String(),
		},
		Name: "test_module",
	}
	suite.accountMock.EXPECT().GetModuleAccount(gomock.Any(), "test_module").Return(&moduleAcc).Times(1)

	coins := sdk.NewCoins(sdk.NewInt64Coin(nativeDenom, 1000))

	// Mock the initial coin transfer from module to tradebin
	suite.bankMock.EXPECT().
		SendCoinsFromModuleToModule(gomock.Any(), "test_module", types.ModuleName, coins).
		Times(1).
		Return(nil)

	result, err := suite.k.ModuleSwapForNativeDenom(suite.ctx, "test_module", coins)
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "cannot swap native coin to native coin")
	suite.Require().Equal(sdk.Coin{}, result, "should return zero coin on error")
}

func (suite *IntegrationTestSuite) TestServiceDenom_ModuleSwapForNativeDenom_PoolNotExists() {
	nativeDenom := "ubze"
	otherDenom := "uother"
	params := denomTestParams(nativeDenom)
	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	// Mock module account
	addr := sdk.AccAddress("moduleacc")
	moduleAcc := authtypes.ModuleAccount{
		BaseAccount: &authtypes.BaseAccount{
			Address: addr.String(),
		},
		Name: "test_module",
	}
	suite.accountMock.EXPECT().GetModuleAccount(gomock.Any(), "test_module").Return(&moduleAcc).Times(1)

	coins := sdk.NewCoins(sdk.NewInt64Coin(otherDenom, 1000))

	// Mock the initial coin transfer from module to tradebin
	suite.bankMock.EXPECT().
		SendCoinsFromModuleToModule(gomock.Any(), "test_module", types.ModuleName, coins).
		Times(1).
		Return(nil)

	result, err := suite.k.ModuleSwapForNativeDenom(suite.ctx, "test_module", coins)
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "cannot find liquidity pool")
	suite.Require().Equal(sdk.Coin{}, result, "should return zero coin on error")
}

func (suite *IntegrationTestSuite) TestServiceDenom_ModuleSwapForNativeDenom_InsufficientLiquidity() {
	nativeDenom := "ubze"
	otherDenom := "uother"
	params := denomTestParamsWithLiquidity(nativeDenom, math.NewInt(2_000_000_000))
	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	// Mock module account
	addr := sdk.AccAddress("moduleacc")
	moduleAcc := authtypes.ModuleAccount{
		BaseAccount: &authtypes.BaseAccount{
			Address: addr.String(),
		},
		Name: "test_module",
	}
	suite.accountMock.EXPECT().GetModuleAccount(gomock.Any(), "test_module").Return(&moduleAcc).Times(1)

	// Create pool with insufficient liquidity
	pool := types.LiquidityPool{
		Id:           "ubze_uother",
		Base:         nativeDenom,
		Quote:        otherDenom,
		ReserveBase:  math.NewInt(1_000_000_000), // Less than minNativeAmountForSwap
		ReserveQuote: math.NewInt(1_000_000_000),
	}
	suite.k.SetLiquidityPool(suite.ctx, pool)

	coins := sdk.NewCoins(sdk.NewInt64Coin(otherDenom, 1000))

	// Mock the initial coin transfer from module to tradebin
	suite.bankMock.EXPECT().
		SendCoinsFromModuleToModule(gomock.Any(), "test_module", types.ModuleName, coins).
		Times(1).
		Return(nil)

	result, err := suite.k.ModuleSwapForNativeDenom(suite.ctx, "test_module", coins)
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "not enough liquidity available")
	suite.Require().Equal(sdk.Coin{}, result, "should return zero coin on error")
}

func (suite *IntegrationTestSuite) TestServiceDenom_ModuleSwapForNativeDenom_SwapTokensError() {
	nativeDenom := "ubze"
	otherDenom := "uother"
	params := denomTestParams(nativeDenom)
	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	// Mock module account for the calling module
	addr := sdk.AccAddress("moduleacc")
	moduleAcc := authtypes.ModuleAccount{
		BaseAccount: &authtypes.BaseAccount{
			Address: addr.String(),
		},
		Name: "test_module",
	}
	suite.accountMock.EXPECT().GetModuleAccount(gomock.Any(), "test_module").Return(&moduleAcc).Times(1)

	// Mock tradebin module account (used internally)
	tradebinAddr := sdk.AccAddress("tradebinmodule")
	tradebinModuleAcc := authtypes.ModuleAccount{
		BaseAccount: &authtypes.BaseAccount{
			Address: tradebinAddr.String(),
		},
		Name: types.ModuleName,
	}
	suite.accountMock.EXPECT().GetModuleAccount(gomock.Any(), types.ModuleName).Return(&tradebinModuleAcc).Times(1)

	// Create pool with sufficient liquidity but will cause swap error
	pool := types.LiquidityPool{
		Id:           "ubze_uother",
		Base:         nativeDenom,
		Quote:        otherDenom,
		ReserveBase:  math.NewInt(100_000_000_000),
		ReserveQuote: math.ZeroInt(), // This will cause swap error
	}
	suite.k.SetLiquidityPool(suite.ctx, pool)

	coins := sdk.NewCoins(sdk.NewInt64Coin(otherDenom, 1000))

	// Mock first send coins call to succeed
	suite.bankMock.EXPECT().SendCoinsFromModuleToModule(gomock.Any(), types.ModuleName, "burner", coins).Return(nil).Times(1)
	suite.bankMock.EXPECT().SendCoinsFromModuleToModule(gomock.Any(), "test_module", types.ModuleName, coins).Return(nil).Times(1)
	suite.bankMock.EXPECT().SendCoinsFromModuleToModule(gomock.Any(), types.ModuleName, "test_module", gomock.Any()).Return(fmt.Errorf("test err")).Times(1)

	result, err := suite.k.ModuleSwapForNativeDenom(suite.ctx, "test_module", coins)
	suite.Require().Error(err)
	suite.Require().Equal(sdk.Coin{}, result, "should return zero coin on error")
}

func (suite *IntegrationTestSuite) TestServiceDenom_ModuleSwapForNativeDenom_SendCoinsFromModuleError() {
	nativeDenom := "ubze"
	otherDenom := "uother"
	params := denomTestParams(nativeDenom)
	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	// Mock module account for the calling module
	addr := sdk.AccAddress("moduleacc")
	moduleAcc := authtypes.ModuleAccount{
		BaseAccount: &authtypes.BaseAccount{
			Address: addr.String(),
		},
		Name: "test_module",
	}
	suite.accountMock.EXPECT().GetModuleAccount(gomock.Any(), "test_module").Return(&moduleAcc).Times(1)
	suite.accountMock.EXPECT().GetModuleAccount(gomock.Any(), types.ModuleName).Return(&moduleAcc).Times(1)

	// Create pool with sufficient liquidity
	pool := types.LiquidityPool{
		Id:           "ubze_uother",
		Base:         nativeDenom,
		Quote:        otherDenom,
		ReserveBase:  math.NewInt(100_000_000_000),
		ReserveQuote: math.NewInt(1_000_000_000),
		Fee:          math.LegacyNewDecWithPrec(3, 3), // 0.3%
		FeeDest: &types.FeeDestination{
			Treasury:  math.LegacyNewDecWithPrec(1, 0), // 50%
			Burner:    math.LegacyZeroDec(),
			Providers: math.LegacyZeroDec(), // 0%
		},
	}
	suite.k.SetLiquidityPool(suite.ctx, pool)

	coins := sdk.NewCoins(sdk.NewInt64Coin(otherDenom, 1000))
	sendError := errors.New("send coins error")

	// Mock first send coins call to fail
	suite.bankMock.EXPECT().SendCoinsFromModuleToModule(gomock.Any(), "test_module", types.ModuleName, coins).Return(sendError).Times(1)
	suite.bankMock.EXPECT().SendCoinsFromModuleToModule(gomock.Any(), types.ModuleName, "test_module", coins).Return(nil).Times(1)
	suite.bankMock.EXPECT().SendCoinsFromModuleToModule(gomock.Any(), types.ModuleName, gomock.Any(), gomock.Any()).Return(nil).Times(1)

	result, err := suite.k.ModuleSwapForNativeDenom(suite.ctx, "test_module", coins)
	suite.Require().Error(err)
	suite.Require().Equal(sendError, err)
	suite.Require().Equal(sdk.Coin{}, result, "should return zero coin on error")
}

func (suite *IntegrationTestSuite) TestServiceDenom_ModuleSwapForNativeDenom_SendSwapResultError() {
	nativeDenom := "ubze"
	otherDenom := "uother"
	params := denomTestParams(nativeDenom)
	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	// Mock module account for the calling module
	addr := sdk.AccAddress("moduleacc")
	moduleAcc := authtypes.ModuleAccount{
		BaseAccount: &authtypes.BaseAccount{
			Address: addr.String(),
		},
		Name: "test_module",
	}
	suite.accountMock.EXPECT().GetModuleAccount(gomock.Any(), "test_module").Return(&moduleAcc).Times(1)

	// Mock tradebin module account (used internally)
	tradebinAddr := sdk.AccAddress("tradebinmodule")
	tradebinModuleAcc := authtypes.ModuleAccount{
		BaseAccount: &authtypes.BaseAccount{
			Address: tradebinAddr.String(),
		},
		Name: types.ModuleName,
	}
	suite.accountMock.EXPECT().GetModuleAccount(gomock.Any(), types.ModuleName).Return(&tradebinModuleAcc).Times(1)

	// Create pool with sufficient liquidity (must be > MinNativeLiquidityForModuleSwap)
	pool := types.LiquidityPool{
		Id:           "ubze_uother",
		Base:         nativeDenom,
		Quote:        otherDenom,
		ReserveBase:  math.NewInt(200_000_000_000),
		ReserveQuote: math.NewInt(1_000_000_000),
		Fee:          math.LegacyNewDecWithPrec(3, 3), // 0.3%
		FeeDest: &types.FeeDestination{
			Treasury:  math.LegacyNewDecWithPrec(5, 1), // 50%
			Burner:    math.LegacyNewDecWithPrec(5, 1), // 50%
			Providers: math.LegacyZeroDec(),            // 0%
		},
	}
	suite.k.SetLiquidityPool(suite.ctx, pool)

	coins := sdk.NewCoins(sdk.NewInt64Coin(otherDenom, 1000))
	sendError := errors.New("send swap result error")

	// Mock first send coins call to succeed
	suite.bankMock.EXPECT().SendCoinsFromModuleToModule(gomock.Any(), "test_module", types.ModuleName, coins).Return(nil).Times(1)

	// Mock second send coins call to fail
	suite.bankMock.EXPECT().SendCoinsFromModuleToModule(gomock.Any(), types.ModuleName, "burner", gomock.Any()).Return(nil).Times(1)
	suite.bankMock.EXPECT().SendCoinsFromModuleToModule(gomock.Any(), types.ModuleName, "test_module", gomock.Any()).Return(sendError).Times(1)
	suite.bankMock.EXPECT().SendCoinsFromModuleToModule(gomock.Any(), types.ModuleName, gomock.Any(), gomock.Any()).Return(nil).Times(1)

	result, err := suite.k.ModuleSwapForNativeDenom(suite.ctx, "test_module", coins)
	suite.Require().Error(err)
	suite.Require().Equal(sendError, err)
	suite.Require().Equal(sdk.Coin{}, result, "should return zero coin on error, not partial swap result")
}

func (suite *IntegrationTestSuite) TestServiceDenom_ModuleSwapForNativeDenom_Success() {
	nativeDenom := "ubze"
	otherDenom := "uother"
	params := denomTestParams(nativeDenom)
	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	// Mock module account for the calling module
	addr := sdk.AccAddress("moduleacc")
	moduleAcc := authtypes.ModuleAccount{
		BaseAccount: &authtypes.BaseAccount{
			Address: addr.String(),
		},
		Name: "test_module",
	}
	suite.accountMock.EXPECT().GetModuleAccount(gomock.Any(), "test_module").Return(&moduleAcc).Times(1)

	// Mock tradebin module account (used internally)
	tradebinAddr := sdk.AccAddress("tradebinmodule")
	tradebinModuleAcc := authtypes.ModuleAccount{
		BaseAccount: &authtypes.BaseAccount{
			Address: tradebinAddr.String(),
		},
		Name: types.ModuleName,
	}
	suite.accountMock.EXPECT().GetModuleAccount(gomock.Any(), types.ModuleName).Return(&tradebinModuleAcc).Times(1)

	// Create pool with sufficient liquidity (must be > MinNativeLiquidityForModuleSwap)
	pool := types.LiquidityPool{
		Id:           "ubze_uother",
		Base:         nativeDenom,
		Quote:        otherDenom,
		ReserveBase:  math.NewInt(200_000_000_000),
		ReserveQuote: math.NewInt(1_000_000_000),
		Fee:          math.LegacyNewDecWithPrec(3, 3), // 0.3%
		FeeDest: &types.FeeDestination{
			Treasury:  math.LegacyNewDecWithPrec(5, 1), // 50%
			Burner:    math.LegacyNewDecWithPrec(5, 1), // 50%
			Providers: math.LegacyZeroDec(),            // 0%
		},
	}
	suite.k.SetLiquidityPool(suite.ctx, pool)

	coins := sdk.NewCoins(sdk.NewInt64Coin(otherDenom, 1000))

	// Mock both send coins calls to succeed
	suite.bankMock.EXPECT().SendCoinsFromModuleToModule(gomock.Any(), types.ModuleName, "burner", gomock.Any()).Return(nil).Times(1)
	suite.bankMock.EXPECT().SendCoinsFromModuleToModule(gomock.Any(), "test_module", types.ModuleName, coins).Return(nil).Times(1)
	suite.bankMock.EXPECT().SendCoinsFromModuleToModule(gomock.Any(), types.ModuleName, "test_module", gomock.Any()).Return(nil).Times(1)
	suite.bankMock.EXPECT().SendCoinsFromModuleToModule(gomock.Any(), types.ModuleName, gomock.Any(), gomock.Any()).Return(nil).Times(1)

	var result sdk.Coin
	result, err = suite.k.ModuleSwapForNativeDenom(suite.ctx, "test_module", coins)
	suite.Require().NoError(err)
	suite.Require().Equal(nativeDenom, result.Denom)
	suite.Require().True(result.Amount.IsPositive())
}

// ============ ModuleAddLiquidityWithNativeDenom Tests ============

func (suite *IntegrationTestSuite) TestServiceDenom_ModuleAddLiquidityWithNativeDenom_EmptyNativeDenom() {
	// Set empty native denom
	params := denomTestParams("")
	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	coins := sdk.NewCoins(sdk.NewInt64Coin("uother", 1000))

	_, _, err = suite.k.ModuleAddLiquidityWithNativeDenom(suite.ctx, "test_module", coins)
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "native denom not set")
}

func (suite *IntegrationTestSuite) TestServiceDenom_ModuleAddLiquidityWithNativeDenom_CoinIsNativeDenom() {
	nativeDenom := "ubze"
	params := denomTestParams(nativeDenom)
	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	coins := sdk.NewCoins(sdk.NewInt64Coin(nativeDenom, 1000))

	// Mock initial coin capture
	suite.bankMock.EXPECT().SendCoinsFromModuleToModule(gomock.Any(), "test_module", types.ModuleName, coins).Return(nil).Times(1)

	// Mock refund (should refund the native coin)
	suite.bankMock.EXPECT().SendCoinsFromModuleToModule(gomock.Any(), types.ModuleName, "test_module", coins).Return(nil).Times(1)

	addedCoins, refundedCoins, err := suite.k.ModuleAddLiquidityWithNativeDenom(suite.ctx, "test_module", coins)
	suite.Require().NoError(err)
	suite.Require().Empty(addedCoins, "Should not add any coins")
	suite.Require().Equal(coins, refundedCoins, "Should refund the native denom coin")
}

func (suite *IntegrationTestSuite) TestServiceDenom_ModuleAddLiquidityWithNativeDenom_PoolNotExists() {
	nativeDenom := "ubze"
	otherDenom := "uother"
	params := denomTestParams(nativeDenom)
	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	coins := sdk.NewCoins(sdk.NewInt64Coin(otherDenom, 1000))

	// Mock initial coin capture
	suite.bankMock.EXPECT().SendCoinsFromModuleToModule(gomock.Any(), "test_module", types.ModuleName, coins).Return(nil).Times(1)

	// Mock refund (should refund the coin when pool doesn't exist)
	suite.bankMock.EXPECT().SendCoinsFromModuleToModule(gomock.Any(), types.ModuleName, "test_module", coins).Return(nil).Times(1)

	addedCoins, refundedCoins, err := suite.k.ModuleAddLiquidityWithNativeDenom(suite.ctx, "test_module", coins)
	suite.Require().NoError(err)
	suite.Require().Empty(addedCoins, "Should not add any coins")
	suite.Require().Equal(coins, refundedCoins, "Should refund the coin when pool doesn't exist")
}

func (suite *IntegrationTestSuite) TestServiceDenom_ModuleAddLiquidityWithNativeDenom_PoolIsEmpty() {
	nativeDenom := "ubze"
	otherDenom := "uother"
	params := denomTestParams(nativeDenom)
	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	// Create empty pool
	pool := types.LiquidityPool{
		Id:           "ubze_uother",
		Base:         nativeDenom,
		Quote:        otherDenom,
		ReserveBase:  math.ZeroInt(),
		ReserveQuote: math.ZeroInt(),
	}
	suite.k.SetLiquidityPool(suite.ctx, pool)

	coins := sdk.NewCoins(sdk.NewInt64Coin(otherDenom, 1000))

	// Mock initial coin capture
	suite.bankMock.EXPECT().SendCoinsFromModuleToModule(gomock.Any(), "test_module", types.ModuleName, coins).Return(nil).Times(1)

	// Mock refund (should refund the coin when pool is empty)
	suite.bankMock.EXPECT().SendCoinsFromModuleToModule(gomock.Any(), types.ModuleName, "test_module", coins).Return(nil).Times(1)

	addedCoins, refundedCoins, err := suite.k.ModuleAddLiquidityWithNativeDenom(suite.ctx, "test_module", coins)
	suite.Require().NoError(err)
	suite.Require().Empty(addedCoins, "Should not add any coins")
	suite.Require().Equal(coins, refundedCoins, "Should refund the coin when pool is empty")
}

func (suite *IntegrationTestSuite) TestServiceDenom_ModuleAddLiquidityWithNativeDenom_SendCoinsError() {
	nativeDenom := "ubze"
	otherDenom := "uother"
	params := denomTestParams(nativeDenom)
	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	coins := sdk.NewCoins(sdk.NewInt64Coin(otherDenom, 1000))
	sendError := fmt.Errorf("send coins error")

	// Mock send coins to fail
	suite.bankMock.EXPECT().SendCoinsFromModuleToModule(gomock.Any(), "test_module", types.ModuleName, coins).Return(sendError).Times(1)

	_, _, err = suite.k.ModuleAddLiquidityWithNativeDenom(suite.ctx, "test_module", coins)
	suite.Require().Error(err)
	suite.Require().Equal(sendError, err)
}

func (suite *IntegrationTestSuite) TestServiceDenom_ModuleAddLiquidityWithNativeDenom_Success_SingleCoin() {
	nativeDenom := "ubze"
	otherDenom := "uother"
	params := denomTestParams(nativeDenom)
	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	// Create pool with 1:2 ratio (1 ubze = 2 uother)
	// Pool: 1,000,000 ubze, 2,000,000 uother
	initialBaseReserve := math.NewInt(1_000_000)
	initialQuoteReserve := math.NewInt(2_000_000)
	pool := types.LiquidityPool{
		Id:           "ubze_uother",
		Base:         nativeDenom,
		Quote:        otherDenom,
		LpDenom:      "ulp_ubze_uother",
		ReserveBase:  initialBaseReserve,
		ReserveQuote: initialQuoteReserve,
		Fee:          math.LegacyNewDecWithPrec(3, 3), // 0.3% fee
		FeeDest: &types.FeeDestination{
			Treasury:  math.LegacyZeroDec(),
			Burner:    math.LegacyZeroDec(),
			Providers: math.LegacyOneDec(), // 100% to providers
		},
	}
	suite.k.SetLiquidityPool(suite.ctx, pool)

	// User sends 500,000 uother
	inputAmount := math.NewInt(500_000)
	coins := sdk.NewCoins(sdk.NewCoin(otherDenom, inputAmount))

	// Mock tradebin module account (for swaps and LP minting)
	tradebinAddr := sdk.AccAddress("tradebinmodule")
	tradebinModuleAcc := authtypes.ModuleAccount{
		BaseAccount: &authtypes.BaseAccount{
			Address: tradebinAddr.String(),
		},
		Name: types.ModuleName,
	}
	suite.accountMock.EXPECT().GetModuleAccount(gomock.Any(), types.ModuleName).Return(&tradebinModuleAcc).Times(1)

	// Mock LP supply
	lpSupply := sdk.NewCoin(pool.LpDenom, math.NewInt(1_000_000_000))
	suite.bankMock.EXPECT().GetSupply(gomock.Any(), pool.LpDenom).Return(lpSupply).Times(1)

	// Mock initial coin capture
	suite.bankMock.EXPECT().SendCoinsFromModuleToModule(gomock.Any(), "test_module", types.ModuleName, coins).Return(nil).Times(1)

	// Mock LP token minting
	suite.bankMock.EXPECT().MintCoins(gomock.Any(), types.ModuleName, gomock.Any()).Return(nil).Times(1)

	// Mock single refund send at the end (includes LP tokens + native leftovers)
	suite.bankMock.EXPECT().SendCoinsFromModuleToModule(gomock.Any(), types.ModuleName, "test_module", gomock.Any()).Return(nil).Times(1)

	addedCoins, refundedCoins, err := suite.k.ModuleAddLiquidityWithNativeDenom(suite.ctx, "test_module", coins)
	suite.Require().NoError(err)

	// Verify added coins contains the input denom
	suite.Require().NotEmpty(addedCoins, "Should have added coins")
	suite.T().Logf("Added coins: %s", addedCoins)
	suite.T().Logf("Refunded coins: %s", refundedCoins)

	// Verify pool was updated
	updatedPool, found := suite.k.GetLiquidityPool(suite.ctx, pool.Id)
	suite.Require().True(found)

	// Verify quote reserve increased
	suite.Require().True(updatedPool.ReserveQuote.GT(initialQuoteReserve), "Quote reserve should increase")

	// Base reserve may decrease slightly due to swap mechanics and leftover returns
	// This is acceptable as long as the leftover is returned to caller

	// Calculate changes for logging
	baseChange := updatedPool.ReserveBase.Sub(initialBaseReserve)
	quoteChange := updatedPool.ReserveQuote.Sub(initialQuoteReserve)

	suite.T().Logf("Initial pool: Base=%s, Quote=%s", initialBaseReserve, initialQuoteReserve)
	suite.T().Logf("Updated pool: Base=%s, Quote=%s", updatedPool.ReserveBase, updatedPool.ReserveQuote)
	suite.T().Logf("Changes: Base=%s, Quote=%s", baseChange, quoteChange)
}

func (suite *IntegrationTestSuite) TestServiceDenom_ModuleAddLiquidityWithNativeDenom_MathValidation() {
	nativeDenom := "ubze"
	otherDenom := "utoken"
	params := denomTestParams(nativeDenom)
	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	// Create pool with simple 1:1 ratio for easier math validation
	// Pool: 10,000,000 ubze, 10,000,000 utoken
	initialBaseReserve := math.NewInt(10_000_000)
	initialQuoteReserve := math.NewInt(10_000_000)
	pool := types.LiquidityPool{
		Id:           "ubze_utoken",
		Base:         nativeDenom,
		Quote:        otherDenom,
		LpDenom:      "ulp_ubze_utoken",
		ReserveBase:  initialBaseReserve,
		ReserveQuote: initialQuoteReserve,
		Fee:          math.LegacyNewDecWithPrec(3, 3), // 0.3% fee
		FeeDest: &types.FeeDestination{
			Treasury:  math.LegacyZeroDec(),
			Burner:    math.LegacyZeroDec(),
			Providers: math.LegacyOneDec(), // 100% to providers
		},
	}
	suite.k.SetLiquidityPool(suite.ctx, pool)

	// User sends 1,000,000 utoken
	inputAmount := math.NewInt(1_000_000)
	coins := sdk.NewCoins(sdk.NewCoin(otherDenom, inputAmount))

	// Mock tradebin module account
	tradebinAddr := sdk.AccAddress("tradebinmodule")
	tradebinModuleAcc := authtypes.ModuleAccount{
		BaseAccount: &authtypes.BaseAccount{
			Address: tradebinAddr.String(),
		},
		Name: types.ModuleName,
	}
	suite.accountMock.EXPECT().GetModuleAccount(gomock.Any(), types.ModuleName).Return(&tradebinModuleAcc).Times(1)

	// Mock LP supply
	lpSupply := sdk.NewCoin(pool.LpDenom, math.NewInt(100_000_000))
	suite.bankMock.EXPECT().GetSupply(gomock.Any(), pool.LpDenom).Return(lpSupply).Times(1)

	// Mock initial coin capture
	suite.bankMock.EXPECT().SendCoinsFromModuleToModule(gomock.Any(), "test_module", types.ModuleName, coins).Return(nil).Times(1)

	// Mock LP token minting
	var mintedLPTokens sdk.Coins
	suite.bankMock.EXPECT().MintCoins(gomock.Any(), types.ModuleName, gomock.Any()).DoAndReturn(
		func(_ sdk.Context, _ string, mintedCoins sdk.Coins) error {
			suite.T().Logf("Minted LP tokens: %s", mintedCoins)
			mintedLPTokens = mintedCoins
			return nil
		},
	).Times(1)

	// Mock single refund send at the end (includes LP tokens + native leftovers)
	var receivedRefunds sdk.Coins
	suite.bankMock.EXPECT().SendCoinsFromModuleToModule(gomock.Any(), types.ModuleName, "test_module", gomock.Any()).DoAndReturn(
		func(_ sdk.Context, _ string, _ string, sentCoins sdk.Coins) error {
			suite.T().Logf("Refunded coins sent to module: %s", sentCoins)
			receivedRefunds = sentCoins
			return nil
		},
	).Times(1)

	addedCoins, refundedCoins, err := suite.k.ModuleAddLiquidityWithNativeDenom(suite.ctx, "test_module", coins)
	suite.Require().NoError(err)

	suite.T().Logf("Added coins: %s", addedCoins)
	suite.T().Logf("Refunded coins: %s", refundedCoins)

	// Verify pool was updated
	updatedPool, found := suite.k.GetLiquidityPool(suite.ctx, pool.Id)
	suite.Require().True(found)

	// Math validation
	suite.T().Logf("=== Math Validation ===")
	suite.T().Logf("Input: %s %s", inputAmount, otherDenom)
	suite.T().Logf("Minted LP tokens: %s", mintedLPTokens)
	suite.T().Logf("Received refunds (LP + native leftovers): %s", receivedRefunds)

	// Verify refunded coins include LP tokens and native leftovers
	suite.Require().NotEmpty(refundedCoins, "Should have refunded coins (LP tokens + native leftovers)")
	suite.Require().Equal(receivedRefunds, refundedCoins, "Refunded coins should match what was sent")

	// Find LP tokens in refunded coins
	var hasLPTokens bool
	var hasNativeLeftover bool
	for _, coin := range refundedCoins {
		if coin.Denom == pool.LpDenom {
			hasLPTokens = true
			suite.Require().True(coin.Amount.IsPositive(), "LP token amount should be positive")
		}
		if coin.Denom == nativeDenom {
			hasNativeLeftover = true
			suite.T().Logf("Native leftover amount: %s (%.2f%% of input value)",
				coin.Amount,
				float64(coin.Amount.Int64())/float64(inputAmount.Int64())*100)
		}
	}
	suite.Require().True(hasLPTokens, "Refunded coins should include LP tokens")
	// Native leftover is optional depending on the math

	// Verify quote reserve increased
	suite.Require().True(updatedPool.ReserveQuote.GT(initialQuoteReserve),
		"Quote reserve should increase")

	// Base reserve may decrease slightly if native refunds are returned
	baseChange := updatedPool.ReserveBase.Sub(initialBaseReserve)
	if hasNativeLeftover && baseChange.IsNegative() {
		suite.T().Logf("Base decreased by %s (native leftover was refunded)", baseChange.Abs())
		// The native leftover should be reasonable (< 1% of input)
		for _, coin := range refundedCoins {
			if coin.Denom == nativeDenom {
				maxReasonableRefund := inputAmount.Quo(math.NewInt(100))
				suite.Require().True(coin.Amount.LTE(maxReasonableRefund),
					"Native leftover should be small (< 1%% of input)")
			}
		}
	}
}

func (suite *IntegrationTestSuite) TestServiceDenom_ModuleAddLiquidityWithNativeDenom_MultipleCoins() {
	nativeDenom := "ubze"
	token1 := "utoken1"
	token2 := "utoken2"
	params := denomTestParams(nativeDenom)
	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	// Create pools
	pool1 := types.LiquidityPool{
		Id:           "ubze_utoken1",
		Base:         nativeDenom,
		Quote:        token1,
		LpDenom:      "ulp_ubze_utoken1",
		ReserveBase:  math.NewInt(10_000_000),
		ReserveQuote: math.NewInt(20_000_000),
		Fee:          math.LegacyNewDecWithPrec(3, 3),
		FeeDest: &types.FeeDestination{
			Treasury:  math.LegacyZeroDec(),
			Burner:    math.LegacyZeroDec(),
			Providers: math.LegacyOneDec(),
		},
	}
	pool2 := types.LiquidityPool{
		Id:           "ubze_utoken2",
		Base:         nativeDenom,
		Quote:        token2,
		LpDenom:      "ulp_ubze_utoken2",
		ReserveBase:  math.NewInt(5_000_000),
		ReserveQuote: math.NewInt(10_000_000),
		Fee:          math.LegacyNewDecWithPrec(3, 3),
		FeeDest: &types.FeeDestination{
			Treasury:  math.LegacyZeroDec(),
			Burner:    math.LegacyZeroDec(),
			Providers: math.LegacyOneDec(),
		},
	}
	suite.k.SetLiquidityPool(suite.ctx, pool1)
	suite.k.SetLiquidityPool(suite.ctx, pool2)

	coins := sdk.NewCoins(
		sdk.NewCoin(token1, math.NewInt(100_000)),
		sdk.NewCoin(token2, math.NewInt(50_000)),
	)

	// Mock tradebin module account
	tradebinAddr := sdk.AccAddress("tradebinmodule")
	tradebinModuleAcc := authtypes.ModuleAccount{
		BaseAccount: &authtypes.BaseAccount{
			Address: tradebinAddr.String(),
		},
		Name: types.ModuleName,
	}
	suite.accountMock.EXPECT().GetModuleAccount(gomock.Any(), types.ModuleName).Return(&tradebinModuleAcc).Times(2)

	// Mock LP supplies
	suite.bankMock.EXPECT().GetSupply(gomock.Any(), pool1.LpDenom).Return(sdk.NewCoin(pool1.LpDenom, math.NewInt(10_000_000))).Times(1)
	suite.bankMock.EXPECT().GetSupply(gomock.Any(), pool2.LpDenom).Return(sdk.NewCoin(pool2.LpDenom, math.NewInt(5_000_000))).Times(1)

	// Mock initial coin capture
	suite.bankMock.EXPECT().SendCoinsFromModuleToModule(gomock.Any(), "test_module", types.ModuleName, coins).Return(nil).Times(1)

	// Mock LP token minting (2 times, one for each coin)
	suite.bankMock.EXPECT().MintCoins(gomock.Any(), types.ModuleName, gomock.Any()).Return(nil).Times(2)

	// Mock single refund send at the end (includes both LP tokens + any native leftovers)
	suite.bankMock.EXPECT().SendCoinsFromModuleToModule(gomock.Any(), types.ModuleName, "test_module", gomock.Any()).Return(nil).Times(1)

	addedCoins, refundedCoins, err := suite.k.ModuleAddLiquidityWithNativeDenom(suite.ctx, "test_module", coins)
	suite.Require().NoError(err)

	// Verify both coins were added
	suite.Require().NotEmpty(addedCoins, "Should have added coins")
	suite.Require().Equal(2, len(addedCoins), "Should have added both coins")
	suite.T().Logf("Added coins: %s", addedCoins)
	suite.T().Logf("Refunded coins: %s", refundedCoins)

	// Verify both pools were updated
	updatedPool1, found := suite.k.GetLiquidityPool(suite.ctx, pool1.Id)
	suite.Require().True(found)
	suite.Require().True(updatedPool1.ReserveBase.GTE(pool1.ReserveBase), "Pool1 base should not decrease")
	suite.Require().True(updatedPool1.ReserveQuote.GT(pool1.ReserveQuote), "Pool1 quote should increase")

	updatedPool2, found := suite.k.GetLiquidityPool(suite.ctx, pool2.Id)
	suite.Require().True(found)
	suite.Require().True(updatedPool2.ReserveBase.GTE(pool2.ReserveBase), "Pool2 base should not decrease")
	suite.Require().True(updatedPool2.ReserveQuote.GT(pool2.ReserveQuote), "Pool2 quote should increase")
}

func (suite *IntegrationTestSuite) TestServiceDenom_ModuleAddLiquidityWithNativeDenom_ResilientBehavior() {
	nativeDenom := "ubze"
	token1 := "utoken1" // Has pool - should succeed
	token2 := "utoken2" // No pool - should refund
	params := denomTestParams(nativeDenom)
	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	// Create pool only for token1
	pool1 := types.LiquidityPool{
		Id:           "ubze_utoken1",
		Base:         nativeDenom,
		Quote:        token1,
		LpDenom:      "ulp_ubze_utoken1",
		ReserveBase:  math.NewInt(10_000_000),
		ReserveQuote: math.NewInt(20_000_000),
		Fee:          math.LegacyNewDecWithPrec(3, 3),
		FeeDest: &types.FeeDestination{
			Treasury:  math.LegacyZeroDec(),
			Burner:    math.LegacyZeroDec(),
			Providers: math.LegacyOneDec(),
		},
	}
	suite.k.SetLiquidityPool(suite.ctx, pool1)

	// Send 3 coins: one will succeed, two will be refunded
	coins := sdk.NewCoins(
		sdk.NewCoin(token1, math.NewInt(100_000)),
		sdk.NewCoin(token2, math.NewInt(50_000)),
		sdk.NewCoin(nativeDenom, math.NewInt(25_000)),
	)

	// Mock tradebin module account (only once for the successful coin)
	tradebinAddr := sdk.AccAddress("tradebinmodule")
	tradebinModuleAcc := authtypes.ModuleAccount{
		BaseAccount: &authtypes.BaseAccount{
			Address: tradebinAddr.String(),
		},
		Name: types.ModuleName,
	}
	suite.accountMock.EXPECT().GetModuleAccount(gomock.Any(), types.ModuleName).Return(&tradebinModuleAcc).Times(1)

	// Mock LP supply for successful coin
	suite.bankMock.EXPECT().GetSupply(gomock.Any(), pool1.LpDenom).Return(sdk.NewCoin(pool1.LpDenom, math.NewInt(10_000_000))).Times(1)

	// Mock initial coin capture
	suite.bankMock.EXPECT().SendCoinsFromModuleToModule(gomock.Any(), "test_module", types.ModuleName, coins).Return(nil).Times(1)

	// Mock LP token minting (only 1 time for successful coin)
	suite.bankMock.EXPECT().MintCoins(gomock.Any(), types.ModuleName, gomock.Any()).Return(nil).Times(1)

	// Mock single refund send at the end (includes LP tokens + refunded coins + native leftover)
	suite.bankMock.EXPECT().SendCoinsFromModuleToModule(gomock.Any(), types.ModuleName, "test_module", gomock.Any()).Return(nil).Times(1)

	addedCoins, refundedCoins, err := suite.k.ModuleAddLiquidityWithNativeDenom(suite.ctx, "test_module", coins)
	suite.Require().NoError(err, "Should not error even when some coins fail")

	// Verify only token1 was added
	suite.Require().Equal(1, len(addedCoins), "Should have added only 1 coin")
	suite.Require().Equal(token1, addedCoins[0].Denom, "Should have added token1")

	// Verify token2 and native were refunded (may include native leftover from token1 processing)
	suite.Require().NotEmpty(refundedCoins, "Should have refunded coins")

	// Check that token2 is in refunds
	hasToken2 := false
	for _, coin := range refundedCoins {
		if coin.Denom == token2 {
			hasToken2 = true
			suite.Require().Equal(math.NewInt(50_000), coin.Amount, "Token2 should be fully refunded")
		}
	}
	suite.Require().True(hasToken2, "Token2 should be in refunds")

	suite.T().Logf("Added coins: %s", addedCoins)
	suite.T().Logf("Refunded coins: %s", refundedCoins)

	// Verify pool1 was updated
	updatedPool1, found := suite.k.GetLiquidityPool(suite.ctx, pool1.Id)
	suite.Require().True(found)
	suite.Require().True(updatedPool1.ReserveQuote.GT(pool1.ReserveQuote), "Pool1 quote should increase")
}

func (suite *IntegrationTestSuite) TestServiceDenom_ModuleSwapForNativeDenom_MultipleCoins() {
	nativeDenom := "ubze"
	otherDenom1 := "uother1"
	otherDenom2 := "uother2"
	params := denomTestParams(nativeDenom)
	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	// Mock module account for the calling module
	addr := sdk.AccAddress("moduleacc")
	moduleAcc := authtypes.ModuleAccount{
		BaseAccount: &authtypes.BaseAccount{
			Address: addr.String(),
		},
		Name: "test_module",
	}
	suite.accountMock.EXPECT().GetModuleAccount(gomock.Any(), "test_module").Return(&moduleAcc).Times(1)

	// Mock tradebin module account (used internally)
	tradebinAddr := sdk.AccAddress("tradebinmodule")
	tradebinModuleAcc := authtypes.ModuleAccount{
		BaseAccount: &authtypes.BaseAccount{
			Address: tradebinAddr.String(),
		},
		Name: types.ModuleName,
	}
	suite.accountMock.EXPECT().GetModuleAccount(gomock.Any(), types.ModuleName).Return(&tradebinModuleAcc).Times(2)

	// Create pools for both denoms (reserves must be > MinNativeLiquidityForModuleSwap)
	pool1 := types.LiquidityPool{
		Id:           "ubze_uother1",
		Base:         nativeDenom,
		Quote:        otherDenom1,
		ReserveBase:  math.NewInt(200_000_000_000),
		ReserveQuote: math.NewInt(1_000_000_000),
		Fee:          math.LegacyNewDecWithPrec(3, 3),
		FeeDest: &types.FeeDestination{
			Treasury:  math.LegacyNewDecWithPrec(5, 1),
			Burner:    math.LegacyNewDecWithPrec(5, 1),
			Providers: math.LegacyZeroDec(),
		},
	}
	pool2 := types.LiquidityPool{
		Id:           "ubze_uother2",
		Base:         nativeDenom,
		Quote:        otherDenom2,
		ReserveBase:  math.NewInt(200_000_000_000),
		ReserveQuote: math.NewInt(1_000_000_000),
		Fee:          math.LegacyNewDecWithPrec(3, 3),
		FeeDest: &types.FeeDestination{
			Treasury:  math.LegacyNewDecWithPrec(5, 1),
			Burner:    math.LegacyNewDecWithPrec(5, 1),
			Providers: math.LegacyZeroDec(),
		},
	}
	suite.k.SetLiquidityPool(suite.ctx, pool1)
	suite.k.SetLiquidityPool(suite.ctx, pool2)

	coins := sdk.NewCoins(
		sdk.NewInt64Coin(otherDenom1, 1000),
		sdk.NewInt64Coin(otherDenom2, 500),
	)

	// Mock both send coins calls to succeed
	suite.bankMock.EXPECT().SendCoinsFromModuleToModule(gomock.Any(), "test_module", types.ModuleName, coins).Return(nil).Times(1)
	suite.bankMock.EXPECT().SendCoinsFromModuleToModule(gomock.Any(), types.ModuleName, "test_module", gomock.Any()).Return(nil).Times(1)
	suite.bankMock.EXPECT().SendCoinsFromModuleToModule(gomock.Any(), types.ModuleName, "burner", gomock.Any()).Return(nil).Times(1)
	suite.bankMock.EXPECT().SendCoinsFromModuleToModule(gomock.Any(), types.ModuleName, gomock.Any(), gomock.Any()).Return(nil).Times(1)

	var result sdk.Coin
	result, err = suite.k.ModuleSwapForNativeDenom(suite.ctx, "test_module", coins)
	suite.Require().NoError(err)
	suite.Require().Equal(nativeDenom, result.Denom)
	suite.Require().True(result.Amount.IsPositive())
}

// TestServiceDenom_ModuleSwapForNativeDenom_PartialSwapReturnsZeroCoinOnError verifies that when multiple coins
// are being swapped and the first swap succeeds but a subsequent one fails, the function returns sdk.Coin{} (not
// the partial result from the first swap). This is important because the function uses CacheContext and never
// calls flush() on error, so the first swap's state changes are discarded. Returning the partial result would be
// misleading since those swaps were never committed.
func (suite *IntegrationTestSuite) TestServiceDenom_ModuleSwapForNativeDenom_PartialSwapReturnsZeroCoinOnError() {
	nativeDenom := "ubze"
	otherDenom1 := "uother1" // Has pool - swap will succeed
	otherDenom2 := "uother2" // No pool - will cause error
	params := denomTestParams(nativeDenom)
	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	// Mock module account for the calling module
	addr := sdk.AccAddress("moduleacc")
	moduleAcc := authtypes.ModuleAccount{
		BaseAccount: &authtypes.BaseAccount{
			Address: addr.String(),
		},
		Name: "test_module",
	}
	suite.accountMock.EXPECT().GetModuleAccount(gomock.Any(), "test_module").Return(&moduleAcc).Times(1)

	// Mock tradebin module account (used internally by swapTokens)
	tradebinAddr := sdk.AccAddress("tradebinmodule")
	tradebinModuleAcc := authtypes.ModuleAccount{
		BaseAccount: &authtypes.BaseAccount{
			Address: tradebinAddr.String(),
		},
		Name: types.ModuleName,
	}
	suite.accountMock.EXPECT().GetModuleAccount(gomock.Any(), types.ModuleName).Return(&tradebinModuleAcc).Times(1)

	// Create pool ONLY for otherDenom1 (otherDenom2 has no pool)
	pool1 := types.LiquidityPool{
		Id:           "ubze_uother1",
		Base:         nativeDenom,
		Quote:        otherDenom1,
		ReserveBase:  math.NewInt(200_000_000_000),
		ReserveQuote: math.NewInt(1_000_000_000),
		Fee:          math.LegacyNewDecWithPrec(3, 3),
		FeeDest: &types.FeeDestination{
			Treasury:  math.LegacyNewDecWithPrec(5, 1),
			Burner:    math.LegacyNewDecWithPrec(5, 1),
			Providers: math.LegacyZeroDec(),
		},
	}
	suite.k.SetLiquidityPool(suite.ctx, pool1)

	coins := sdk.NewCoins(
		sdk.NewInt64Coin(otherDenom1, 1000), // Will swap successfully
		sdk.NewInt64Coin(otherDenom2, 500),  // Will fail - no pool
	)

	// Mock the initial coin transfer
	suite.bankMock.EXPECT().SendCoinsFromModuleToModule(gomock.Any(), "test_module", types.ModuleName, coins).Return(nil).Times(1)
	// Mock fee distribution during first coin's swap
	suite.bankMock.EXPECT().SendCoinsFromModuleToModule(gomock.Any(), types.ModuleName, "burner", gomock.Any()).Return(nil).Times(1)
	suite.bankMock.EXPECT().SendCoinsFromModuleToModule(gomock.Any(), types.ModuleName, gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	result, err := suite.k.ModuleSwapForNativeDenom(suite.ctx, "test_module", coins)
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "cannot find liquidity pool")
	// The critical assertion: even though the first coin swapped successfully (adding to swapResult),
	// the function should return sdk.Coin{} on error, not the partial swap result
	suite.Require().Equal(sdk.Coin{}, result, "should return zero coin on error, not partial swap result from first successful swap")
}

func (suite *IntegrationTestSuite) TestServiceDenom_HasDeepLiquidityWithNativeDenom_SameDenom() {
	nativeDenom := "ubze"
	params := denomTestParamsWithLiquidity(nativeDenom, math.NewInt(2_000_000_000))
	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	// Should return false when checking native denom against itself
	result := suite.k.HasDeepLiquidityWithNativeDenom(suite.ctx, nativeDenom)
	suite.Require().False(result)
}

func (suite *IntegrationTestSuite) TestServiceDenom_HasDeepLiquidityWithNativeDenom_PoolNotExists() {
	nativeDenom := "ubze"
	otherDenom := "uother"
	params := denomTestParamsWithLiquidity(nativeDenom, math.NewInt(2_000_000_000))
	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	// Pool doesn't exist - should return false
	result := suite.k.HasDeepLiquidityWithNativeDenom(suite.ctx, otherDenom)
	suite.Require().False(result)
}

func (suite *IntegrationTestSuite) TestServiceDenom_HasDeepLiquidityWithNativeDenom_ZeroNativeReserves() {
	nativeDenom := "ubze"
	otherDenom := "uother"
	params := denomTestParamsWithLiquidity(nativeDenom, math.NewInt(2_000_000_000))
	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	// Create pool with zero native reserves
	pool := types.LiquidityPool{
		Id:           "ubze_uother",
		Base:         nativeDenom,
		Quote:        otherDenom,
		ReserveBase:  math.NewInt(0), // Zero native reserves
		ReserveQuote: math.NewInt(1_000_000_000),
	}
	suite.k.SetLiquidityPool(suite.ctx, pool)

	result := suite.k.HasDeepLiquidityWithNativeDenom(suite.ctx, otherDenom)
	suite.Require().False(result)
}

func (suite *IntegrationTestSuite) TestServiceDenom_HasDeepLiquidityWithNativeDenom_InsufficientLiquidity() {
	nativeDenom := "ubze"
	otherDenom := "uother"
	params := denomTestParamsWithLiquidity(nativeDenom, math.NewInt(5_000_000_000))
	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	// Create pool with native liquidity below the minimum threshold
	pool := types.LiquidityPool{
		Id:           "ubze_uother",
		Base:         nativeDenom,
		Quote:        otherDenom,
		ReserveBase:  math.NewInt(2_000_000_000), // Less than MinNativeLiquidityForModuleSwap
		ReserveQuote: math.NewInt(1_000_000_000),
	}
	suite.k.SetLiquidityPool(suite.ctx, pool)

	result := suite.k.HasDeepLiquidityWithNativeDenom(suite.ctx, otherDenom)
	suite.Require().False(result)
}

func (suite *IntegrationTestSuite) TestServiceDenom_HasDeepLiquidityWithNativeDenom_SufficientLiquidity() {
	nativeDenom := "ubze"
	otherDenom := "uother"
	params := denomTestParamsWithLiquidity(nativeDenom, math.NewInt(2_000_000_000))
	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	// Create pool with sufficient native liquidity
	pool := types.LiquidityPool{
		Id:           "ubze_uother",
		Base:         nativeDenom,
		Quote:        otherDenom,
		ReserveBase:  math.NewInt(10_000_000_000), // Greater than MinNativeLiquidityForModuleSwap
		ReserveQuote: math.NewInt(5_000_000_000),
	}
	suite.k.SetLiquidityPool(suite.ctx, pool)

	result := suite.k.HasDeepLiquidityWithNativeDenom(suite.ctx, otherDenom)
	suite.Require().True(result)
}

func (suite *IntegrationTestSuite) TestServiceDenom_HasDeepLiquidityWithNativeDenom_NativeInQuotePosition() {
	nativeDenom := "ubze"
	otherDenom := "aaa" // Comes before "ubze" alphabetically, so ubze will be in quote position
	params := denomTestParamsWithLiquidity(nativeDenom, math.NewInt(2_000_000_000))
	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	// Create pool with native denom in quote position (alphabetically sorted: aaa < ubze)
	pool := types.LiquidityPool{
		Id:           "aaa_ubze",
		Base:         otherDenom,  // "aaa"
		Quote:        nativeDenom, // "ubze"
		ReserveBase:  math.NewInt(5_000_000_000),
		ReserveQuote: math.NewInt(10_000_000_000), // Native reserves in quote position
	}
	suite.k.SetLiquidityPool(suite.ctx, pool)

	result := suite.k.HasDeepLiquidityWithNativeDenom(suite.ctx, otherDenom)
	suite.Require().True(result)
}

func (suite *IntegrationTestSuite) TestServiceDenom_HasDeepLiquidityWithNativeDenom_ExactlyAtThreshold() {
	nativeDenom := "ubze"
	otherDenom := "uother"
	minLiquidity := math.NewInt(5_000_000_000)
	params := denomTestParamsWithLiquidity(nativeDenom, minLiquidity)
	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	// Create pool with native liquidity exactly at the minimum threshold
	pool := types.LiquidityPool{
		Id:           "ubze_uother",
		Base:         nativeDenom,
		Quote:        otherDenom,
		ReserveBase:  minLiquidity, // Exactly at threshold
		ReserveQuote: math.NewInt(1_000_000_000),
	}
	suite.k.SetLiquidityPool(suite.ctx, pool)

	// Should return true because amount >= threshold (LT check passes when amount is NOT less than threshold)
	result := suite.k.HasDeepLiquidityWithNativeDenom(suite.ctx, otherDenom)
	suite.Require().True(result)
}

func (suite *IntegrationTestSuite) TestServiceDenom_HasDeepLiquidityWithNativeDenom_JustAboveThreshold() {
	nativeDenom := "ubze"
	otherDenom := "uother"
	minLiquidity := math.NewInt(5_000_000_000)
	params := denomTestParamsWithLiquidity(nativeDenom, minLiquidity)
	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	// Create pool with native liquidity just above the minimum threshold
	pool := types.LiquidityPool{
		Id:           "ubze_uother",
		Base:         nativeDenom,
		Quote:        otherDenom,
		ReserveBase:  minLiquidity.Add(math.NewInt(1)), // One unit above threshold
		ReserveQuote: math.NewInt(1_000_000_000),
	}
	suite.k.SetLiquidityPool(suite.ctx, pool)

	result := suite.k.HasDeepLiquidityWithNativeDenom(suite.ctx, otherDenom)
	suite.Require().True(result)
}
