package keeper_test

import (
	"errors"
	"fmt"

	"cosmossdk.io/math"
	"github.com/bze-alphateam/bze/x/tradebin/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"go.uber.org/mock/gomock"
)

func (suite *IntegrationTestSuite) TestServiceDenom_IsNativeDenom_True() {
	nativeDenom := "ubze"
	params := types.Params{NativeDenom: nativeDenom}
	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	result := suite.k.IsNativeDenom(suite.ctx, nativeDenom)
	suite.Require().True(result)
}

func (suite *IntegrationTestSuite) TestServiceDenom_IsNativeDenom_False() {
	nativeDenom := "ubze"
	params := types.Params{NativeDenom: nativeDenom}
	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	result := suite.k.IsNativeDenom(suite.ctx, "uother")
	suite.Require().False(result)
}

func (suite *IntegrationTestSuite) TestServiceDenom_CanSwapForNativeDenom_SameDenom() {
	nativeDenom := "ubze"
	params := types.Params{NativeDenom: nativeDenom}
	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	// Should return false when trying to swap native denom for itself
	result := suite.k.CanSwapForNativeDenom(suite.ctx, sdk.NewInt64Coin(nativeDenom, 1000))
	suite.Require().False(result)
}

func (suite *IntegrationTestSuite) TestServiceDenom_CanSwapForNativeDenom_PoolNotExists() {
	nativeDenom := "ubze"
	otherDenom := "uother"
	params := types.Params{NativeDenom: nativeDenom}
	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	// Pool doesn't exist - should return false
	result := suite.k.CanSwapForNativeDenom(suite.ctx, sdk.NewInt64Coin(otherDenom, 1000))
	suite.Require().False(result)
}

func (suite *IntegrationTestSuite) TestServiceDenom_CanSwapForNativeDenom_InsufficientLiquidity() {
	nativeDenom := "ubze"
	otherDenom := "uother"
	params := types.Params{NativeDenom: nativeDenom}
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
	params := types.Params{NativeDenom: nativeDenom}
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
	params := types.Params{NativeDenom: ""}
	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	coins := sdk.NewCoins(sdk.NewInt64Coin("uother", 1000))

	_, err = suite.k.ModuleSwapForNativeDenom(suite.ctx, "test_module", coins)
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "native denom not set")
}

func (suite *IntegrationTestSuite) TestServiceDenom_ModuleSwapForNativeDenom_SwapNativeToNative() {
	nativeDenom := "ubze"
	params := types.Params{NativeDenom: nativeDenom}
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

	_, err = suite.k.ModuleSwapForNativeDenom(suite.ctx, "test_module", coins)
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "cannot swap native coin to native coin")
}

func (suite *IntegrationTestSuite) TestServiceDenom_ModuleSwapForNativeDenom_PoolNotExists() {
	nativeDenom := "ubze"
	otherDenom := "uother"
	params := types.Params{NativeDenom: nativeDenom}
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

	_, err = suite.k.ModuleSwapForNativeDenom(suite.ctx, "test_module", coins)
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "cannot find liquidity pool")
}

func (suite *IntegrationTestSuite) TestServiceDenom_ModuleSwapForNativeDenom_InsufficientLiquidity() {
	nativeDenom := "ubze"
	otherDenom := "uother"
	params := types.Params{NativeDenom: nativeDenom}
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

	_, err = suite.k.ModuleSwapForNativeDenom(suite.ctx, "test_module", coins)
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "not enough liquidity available")
}

func (suite *IntegrationTestSuite) TestServiceDenom_ModuleSwapForNativeDenom_SwapTokensError() {
	nativeDenom := "ubze"
	otherDenom := "uother"
	params := types.Params{NativeDenom: nativeDenom}
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

	_, err = suite.k.ModuleSwapForNativeDenom(suite.ctx, "test_module", coins)
	suite.Require().Error(err)
}

func (suite *IntegrationTestSuite) TestServiceDenom_ModuleSwapForNativeDenom_SendCoinsFromModuleError() {
	nativeDenom := "ubze"
	otherDenom := "uother"
	params := types.Params{NativeDenom: nativeDenom}
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
	suite.distrMock.EXPECT().FundCommunityPool(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(1)

	_, err = suite.k.ModuleSwapForNativeDenom(suite.ctx, "test_module", coins)
	suite.Require().Error(err)
	suite.Require().Equal(sendError, err)
}

func (suite *IntegrationTestSuite) TestServiceDenom_ModuleSwapForNativeDenom_SendSwapResultError() {
	nativeDenom := "ubze"
	otherDenom := "uother"
	params := types.Params{NativeDenom: nativeDenom}
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

	// Create pool with sufficient liquidity
	pool := types.LiquidityPool{
		Id:           "ubze_uother",
		Base:         nativeDenom,
		Quote:        otherDenom,
		ReserveBase:  math.NewInt(100_000_000_000),
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
	suite.distrMock.EXPECT().FundCommunityPool(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(1)

	_, err = suite.k.ModuleSwapForNativeDenom(suite.ctx, "test_module", coins)
	suite.Require().Error(err)
	suite.Require().Equal(sendError, err)
}

func (suite *IntegrationTestSuite) TestServiceDenom_ModuleSwapForNativeDenom_Success() {
	nativeDenom := "ubze"
	otherDenom := "uother"
	params := types.Params{NativeDenom: nativeDenom}
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

	// Create pool with sufficient liquidity
	pool := types.LiquidityPool{
		Id:           "ubze_uother",
		Base:         nativeDenom,
		Quote:        otherDenom,
		ReserveBase:  math.NewInt(100_000_000_000),
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
	suite.distrMock.EXPECT().FundCommunityPool(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(1)

	var result sdk.Coin
	result, err = suite.k.ModuleSwapForNativeDenom(suite.ctx, "test_module", coins)
	suite.Require().NoError(err)
	suite.Require().Equal(nativeDenom, result.Denom)
	suite.Require().True(result.Amount.IsPositive())
}

// ============ ModuleAddLiquidityWithNativeDenom Tests ============

func (suite *IntegrationTestSuite) TestServiceDenom_ModuleAddLiquidityWithNativeDenom_EmptyNativeDenom() {
	// Set empty native denom
	params := types.Params{NativeDenom: ""}
	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	coins := sdk.NewCoins(sdk.NewInt64Coin("uother", 1000))

	_, _, err = suite.k.ModuleAddLiquidityWithNativeDenom(suite.ctx, "test_module", coins)
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "native denom not set")
}

func (suite *IntegrationTestSuite) TestServiceDenom_ModuleAddLiquidityWithNativeDenom_CoinIsNativeDenom() {
	nativeDenom := "ubze"
	params := types.Params{NativeDenom: nativeDenom}
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
	params := types.Params{NativeDenom: nativeDenom}
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
	params := types.Params{NativeDenom: nativeDenom}
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
	params := types.Params{NativeDenom: nativeDenom}
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
	params := types.Params{NativeDenom: nativeDenom}
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

	// Mock LP token send to caller module
	suite.bankMock.EXPECT().SendCoinsFromModuleToModule(gomock.Any(), types.ModuleName, "test_module", gomock.Any()).Return(nil).Times(1)

	// Mock leftover native refund (combined with refundedCoins)
	suite.bankMock.EXPECT().SendCoinsFromModuleToModule(gomock.Any(), types.ModuleName, "test_module", gomock.Any()).Return(nil).MaxTimes(1)

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
	params := types.Params{NativeDenom: nativeDenom}
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
	suite.bankMock.EXPECT().MintCoins(gomock.Any(), types.ModuleName, gomock.Any()).DoAndReturn(
		func(_ sdk.Context, _ string, mintedCoins sdk.Coins) error {
			suite.T().Logf("Minted LP tokens: %s", mintedCoins)
			return nil
		},
	).Times(1)

	// Mock LP token send to caller module
	var receivedLPTokens sdk.Coins
	suite.bankMock.EXPECT().SendCoinsFromModuleToModule(gomock.Any(), types.ModuleName, "test_module", gomock.Any()).DoAndReturn(
		func(_ sdk.Context, _ string, _ string, sentCoins sdk.Coins) error {
			suite.T().Logf("LP tokens sent to module: %s", sentCoins)
			receivedLPTokens = sentCoins
			return nil
		},
	).Times(1)

	// Mock leftover native refund (combined in refundedCoins)
	var receivedLeftover sdk.Coins
	suite.bankMock.EXPECT().SendCoinsFromModuleToModule(gomock.Any(), types.ModuleName, "test_module", gomock.Any()).DoAndReturn(
		func(_ sdk.Context, _ string, _ string, sentCoins sdk.Coins) error {
			suite.T().Logf("Refunded coins sent to module: %s", sentCoins)
			receivedLeftover = sentCoins
			return nil
		},
	).MaxTimes(1)

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
	suite.T().Logf("Received LP tokens: %s", receivedLPTokens)
	suite.T().Logf("Received leftover: %s", receivedLeftover)

	// Verify LP tokens were received
	suite.Require().NotEmpty(receivedLPTokens, "Should receive LP tokens")
	suite.Require().True(receivedLPTokens[0].Amount.IsPositive(), "LP tokens should be positive")

	// Verify refunded coins (if any) are in native denom
	if len(refundedCoins) > 0 {
		suite.Require().Equal(nativeDenom, refundedCoins[0].Denom, "Refunded coins should be in native denom")
		suite.T().Logf("Refunded amount: %s (%.2f%% of input value)",
			refundedCoins[0].Amount,
			float64(refundedCoins[0].Amount.Int64())/float64(inputAmount.Int64())*100)
	}

	// Verify quote reserve increased
	suite.Require().True(updatedPool.ReserveQuote.GT(initialQuoteReserve),
		"Quote reserve should increase")

	// Base reserve may decrease slightly if refunds are returned
	// The decrease should match the refunded amount (accounting for swaps)
	baseChange := updatedPool.ReserveBase.Sub(initialBaseReserve)
	if len(refundedCoins) > 0 && baseChange.IsNegative() {
		// If base decreased, the refund should account for it
		suite.T().Logf("Base decreased by %s, refunded: %s", baseChange.Abs(), refundedCoins[0].Amount)
		// The refund should be reasonable (< 1% of input)
		maxReasonableRefund := inputAmount.Quo(math.NewInt(100))
		suite.Require().True(refundedCoins[0].Amount.LTE(maxReasonableRefund),
			"Refund should be small (< 1%% of input)")
	}
}

func (suite *IntegrationTestSuite) TestServiceDenom_ModuleAddLiquidityWithNativeDenom_MultipleCoins() {
	nativeDenom := "ubze"
	token1 := "utoken1"
	token2 := "utoken2"
	params := types.Params{NativeDenom: nativeDenom}
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

	// Mock LP token sends (2 times) + combined refund (up to 1 time)
	// Total: could be 2-3 sends depending on whether there are refunds
	suite.bankMock.EXPECT().SendCoinsFromModuleToModule(gomock.Any(), types.ModuleName, "test_module", gomock.Any()).Return(nil).MinTimes(2).MaxTimes(3)

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
	params := types.Params{NativeDenom: nativeDenom}
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

	// Mock LP token send + combined refund send
	suite.bankMock.EXPECT().SendCoinsFromModuleToModule(gomock.Any(), types.ModuleName, "test_module", gomock.Any()).Return(nil).MinTimes(1).MaxTimes(2)

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
	params := types.Params{NativeDenom: nativeDenom}
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

	// Create pools for both denoms
	pool1 := types.LiquidityPool{
		Id:           "ubze_uother1",
		Base:         nativeDenom,
		Quote:        otherDenom1,
		ReserveBase:  math.NewInt(100_000_000_000),
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
		ReserveBase:  math.NewInt(100_000_000_000),
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
	suite.distrMock.EXPECT().FundCommunityPool(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(1)

	var result sdk.Coin
	result, err = suite.k.ModuleSwapForNativeDenom(suite.ctx, "test_module", coins)
	suite.Require().NoError(err)
	suite.Require().Equal(nativeDenom, result.Denom)
	suite.Require().True(result.Amount.IsPositive())
}
