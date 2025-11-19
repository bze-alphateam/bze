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
