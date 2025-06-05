package keeper_test

import (
	"errors"
	"github.com/bze-alphateam/bze/x/burner/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"go.uber.org/mock/gomock"
)

func (suite *IntegrationTestSuite) TestBurn_TestBurnAnyCoins_OnlyNativeCoins() {
	fromModule := "test_module"
	coins := sdk.NewCoins(
		sdk.NewInt64Coin("ubze", 1000),
		sdk.NewInt64Coin("uother", 500),
	)

	// Mock native denom checks
	suite.trade.EXPECT().IsNativeDenom(suite.ctx, "ubze").Return(true).Times(1)
	suite.trade.EXPECT().IsNativeDenom(suite.ctx, "uother").Return(true).Times(1)

	// Mock burn operation
	suite.bank.EXPECT().BurnCoins(suite.ctx, fromModule, coins).Return(nil).Times(1)

	err := suite.k.BurnAnyCoins(suite.ctx, fromModule, coins)
	suite.Require().NoError(err)
}

func (suite *IntegrationTestSuite) TestBurn_TestBurnAnyCoins_OnlyFactoryTokens() {
	fromModule := "test_module"
	coins := sdk.NewCoins(
		sdk.NewInt64Coin("factory/creator/token1", 1000),
		sdk.NewInt64Coin("factory/creator/token2", 500),
	)

	// Mock native denom checks (should return false for factory tokens)
	suite.trade.EXPECT().IsNativeDenom(suite.ctx, "factory/creator/token1").Return(false).Times(1)
	suite.trade.EXPECT().IsNativeDenom(suite.ctx, "factory/creator/token2").Return(false).Times(1)

	// Mock burn operation
	suite.bank.EXPECT().BurnCoins(suite.ctx, fromModule, coins).Return(nil).Times(1)

	err := suite.k.BurnAnyCoins(suite.ctx, fromModule, coins)
	suite.Require().NoError(err)
}

func (suite *IntegrationTestSuite) TestBurn_TestBurnAnyCoins_OnlyLPTokens() {
	fromModule := "test_module"
	coins := sdk.NewCoins(
		sdk.NewInt64Coin("ulp_token1", 1000),
		sdk.NewInt64Coin("ulp_token2", 500),
	)

	// Mock native denom checks
	suite.trade.EXPECT().IsNativeDenom(suite.ctx, "ulp_token1").Return(false).Times(1)
	suite.trade.EXPECT().IsNativeDenom(suite.ctx, "ulp_token2").Return(false).Times(1)

	// Mock send to black hole module
	suite.bank.EXPECT().SendCoinsFromModuleToModule(suite.ctx, fromModule, types.BlackHoleModuleName, coins).Return(nil).Times(1)

	err := suite.k.BurnAnyCoins(suite.ctx, fromModule, coins)
	suite.Require().NoError(err)
}

func (suite *IntegrationTestSuite) TestBurn_TestBurnAnyCoins_OnlyIBCCoins() {
	fromModule := "test_module"
	coins := sdk.NewCoins(
		sdk.NewInt64Coin("ibc/ABC123", 1000),
		sdk.NewInt64Coin("ibc/DEF456", 500),
	)

	swappedCoin := sdk.NewInt64Coin("ubze", 1400) // Total swapped value

	// Mock native denom checks
	suite.trade.EXPECT().IsNativeDenom(suite.ctx, "ibc/ABC123").Return(false).Times(1)
	suite.trade.EXPECT().IsNativeDenom(suite.ctx, "ibc/DEF456").Return(false).Times(1)

	// Mock swap capability checks
	suite.trade.EXPECT().CanSwapForNativeDenom(suite.ctx, "ibc/ABC123").Return(true).Times(1)
	suite.trade.EXPECT().CanSwapForNativeDenom(suite.ctx, "ibc/DEF456").Return(true).Times(1)

	// Mock swap operation - note: the method is called with the original coins, not individual coins
	suite.trade.EXPECT().ModuleSwapForNativeDenom(suite.ctx, fromModule, coins).Return(swappedCoin, nil).Times(1)

	// Mock burn operation with swapped coins
	suite.bank.EXPECT().BurnCoins(suite.ctx, fromModule, sdk.NewCoins(swappedCoin)).Return(nil).Times(1)

	err := suite.k.BurnAnyCoins(suite.ctx, fromModule, coins)
	suite.Require().NoError(err)
}

func (suite *IntegrationTestSuite) TestBurn_TestBurnAnyCoins_MixedCoins() {
	fromModule := "test_module"
	nativeCoin := sdk.NewInt64Coin("ubze", 1000)
	factoryCoin := sdk.NewInt64Coin("factory/creator/token", 500)
	lpCoin := sdk.NewInt64Coin("ulp_token1", 300)
	ibcCoin := sdk.NewInt64Coin("ibc/ABC123", 200)

	coins := sdk.NewCoins(nativeCoin, factoryCoin, lpCoin, ibcCoin)
	swappedCoin := sdk.NewInt64Coin("uother", 180) // Swapped IBC value - different denom to avoid duplicate
	expectedBurnCoins := sdk.NewCoins(nativeCoin, factoryCoin, swappedCoin)
	lockableCoins := sdk.NewCoins(lpCoin)

	// Mock native denom checks
	suite.trade.EXPECT().IsNativeDenom(suite.ctx, "ubze").Return(true).Times(1)
	suite.trade.EXPECT().IsNativeDenom(suite.ctx, "factory/creator/token").Return(false).Times(1)
	suite.trade.EXPECT().IsNativeDenom(suite.ctx, "ulp_token1").Return(false).Times(1)
	suite.trade.EXPECT().IsNativeDenom(suite.ctx, "ibc/ABC123").Return(false).Times(1)

	// Mock swap capability check for IBC coin
	suite.trade.EXPECT().CanSwapForNativeDenom(suite.ctx, "ibc/ABC123").Return(true).Times(1)

	// Mock swap operation
	suite.trade.EXPECT().ModuleSwapForNativeDenom(suite.ctx, fromModule, coins).Return(swappedCoin, nil).Times(1)

	// Mock send LP tokens to black hole
	suite.bank.EXPECT().SendCoinsFromModuleToModule(suite.ctx, fromModule, types.BlackHoleModuleName, lockableCoins).Return(nil).Times(1)

	// Mock burn operation
	suite.bank.EXPECT().BurnCoins(suite.ctx, fromModule, expectedBurnCoins).Return(nil).Times(1)

	err := suite.k.BurnAnyCoins(suite.ctx, fromModule, coins)
	suite.Require().NoError(err)
}

func (suite *IntegrationTestSuite) TestBurn_TestBurnAnyCoins_EmptyCoins() {
	fromModule := "test_module"
	coins := sdk.NewCoins()

	// Mock burn with empty coins - the method always calls BurnCoins even with empty set
	suite.bank.EXPECT().BurnCoins(suite.ctx, fromModule, gomock.Any()).Return(nil).Times(1)

	err := suite.k.BurnAnyCoins(suite.ctx, fromModule, coins)
	suite.Require().NoError(err)
}

func (suite *IntegrationTestSuite) TestBurn_TestBurnAnyCoins_ZeroAmountCoins() {
	fromModule := "test_module"
	coins := sdk.NewCoins(
		sdk.NewInt64Coin("ubze", 0),
		sdk.NewInt64Coin("uother", 1000),
	)

	// Only the positive coin should be processed
	suite.trade.EXPECT().IsNativeDenom(suite.ctx, "uother").Return(true).Times(1)

	expectedBurnCoins := sdk.NewCoins(sdk.NewInt64Coin("uother", 1000))
	suite.bank.EXPECT().BurnCoins(suite.ctx, fromModule, expectedBurnCoins).Return(nil).Times(1)

	err := suite.k.BurnAnyCoins(suite.ctx, fromModule, coins)
	suite.Require().NoError(err)
}

func (suite *IntegrationTestSuite) TestBurn_TestBurnAnyCoins_SwapError() {
	fromModule := "test_module"
	coins := sdk.NewCoins(sdk.NewInt64Coin("ibc/ABC123", 1000))
	swapError := errors.New("swap failed")

	// Mock native denom check
	suite.trade.EXPECT().IsNativeDenom(suite.ctx, "ibc/ABC123").Return(false).Times(1)

	// Mock swap capability check
	suite.trade.EXPECT().CanSwapForNativeDenom(suite.ctx, "ibc/ABC123").Return(true).Times(1)

	// Mock swap operation failure - returns empty coin and error
	suite.trade.EXPECT().ModuleSwapForNativeDenom(suite.ctx, fromModule, coins).Return(sdk.Coin{}, swapError).Times(1)

	err := suite.k.BurnAnyCoins(suite.ctx, fromModule, coins)
	suite.Require().Error(err)
	suite.Require().Equal(swapError, err)
}

func (suite *IntegrationTestSuite) TestBurn_TestBurnAnyCoins_SendToBlackHoleError() {
	fromModule := "test_module"
	coins := sdk.NewCoins(sdk.NewInt64Coin("ulp_token1", 1000))
	sendError := errors.New("send failed")

	// Mock native denom check
	suite.trade.EXPECT().IsNativeDenom(suite.ctx, "ulp_token1").Return(false).Times(1)

	// Mock send to black hole failure
	suite.bank.EXPECT().SendCoinsFromModuleToModule(suite.ctx, fromModule, types.BlackHoleModuleName, coins).Return(sendError).Times(1)

	err := suite.k.BurnAnyCoins(suite.ctx, fromModule, coins)
	suite.Require().Error(err)
	suite.Require().Equal(sendError, err)
}

func (suite *IntegrationTestSuite) TestBurn_TestBurnAnyCoins_BurnError() {
	fromModule := "test_module"
	coins := sdk.NewCoins(sdk.NewInt64Coin("ubze", 1000))
	burnError := errors.New("burn failed")

	// Mock native denom check
	suite.trade.EXPECT().IsNativeDenom(suite.ctx, "ubze").Return(true).Times(1)

	// Mock burn operation failure
	suite.bank.EXPECT().BurnCoins(suite.ctx, fromModule, coins).Return(burnError).Times(1)

	err := suite.k.BurnAnyCoins(suite.ctx, fromModule, coins)
	suite.Require().Error(err)
	suite.Require().Equal(burnError, err)
}

func (suite *IntegrationTestSuite) TestBurn_TestBurnAnyCoins_UnknownDenomFallsBackToLockable() {
	fromModule := "test_module"
	coins := sdk.NewCoins(sdk.NewInt64Coin("unknown/denom", 1000))

	// Mock native denom check
	suite.trade.EXPECT().IsNativeDenom(suite.ctx, "unknown/denom").Return(false).Times(1)

	// Mock swap capability check (returns false)
	suite.trade.EXPECT().CanSwapForNativeDenom(suite.ctx, "unknown/denom").Return(false).Times(1)

	// Should be treated as lockable and sent to black hole
	suite.bank.EXPECT().SendCoinsFromModuleToModule(suite.ctx, fromModule, types.BlackHoleModuleName, coins).Return(nil).Times(1)

	// Mock burn with empty coins
	suite.bank.EXPECT().BurnCoins(suite.ctx, fromModule, gomock.Any()).Return(nil).Times(1)

	err := suite.k.BurnAnyCoins(suite.ctx, fromModule, coins)
	suite.Require().NoError(err)
}

func (suite *IntegrationTestSuite) TestBurn_TestBurnAnyCoins_IBCNotSwappable() {
	fromModule := "test_module"
	coins := sdk.NewCoins(sdk.NewInt64Coin("ibc/NONSWAPPABLE", 1000))

	// Mock native denom check
	suite.trade.EXPECT().IsNativeDenom(suite.ctx, "ibc/NONSWAPPABLE").Return(false).Times(1)

	// Mock swap capability check (returns false)
	suite.trade.EXPECT().CanSwapForNativeDenom(suite.ctx, "ibc/NONSWAPPABLE").Return(false).Times(1)

	// Should be treated as lockable and sent to black hole
	suite.bank.EXPECT().SendCoinsFromModuleToModule(suite.ctx, fromModule, types.BlackHoleModuleName, coins).Return(nil).Times(1)

	// Mock burn with empty coins
	suite.bank.EXPECT().BurnCoins(suite.ctx, fromModule, gomock.Any()).Return(nil).Times(1)

	err := suite.k.BurnAnyCoins(suite.ctx, fromModule, coins)
	suite.Require().NoError(err)
}
