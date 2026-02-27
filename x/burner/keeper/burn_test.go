package keeper_test

import (
	"errors"
	"fmt"

	"github.com/bze-alphateam/bze/x/burner/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"go.uber.org/mock/gomock"
)

// --- EnqueuePeriodicBurn tests ---

func (suite *IntegrationTestSuite) TestEnqueue_SetsQueue() {
	_, found := suite.k.GetPeriodicBurnQueue(suite.ctx)
	suite.Require().False(found)

	suite.k.EnqueuePeriodicBurn(suite.ctx)

	queue, found := suite.k.GetPeriodicBurnQueue(suite.ctx)
	suite.Require().True(found)
	suite.Require().True(queue.Pending)
}

func (suite *IntegrationTestSuite) TestEnqueue_Idempotent() {
	suite.k.EnqueuePeriodicBurn(suite.ctx)

	queue, found := suite.k.GetPeriodicBurnQueue(suite.ctx)
	suite.Require().True(found)
	suite.Require().True(queue.Pending)

	// Second enqueue should not change anything
	suite.k.EnqueuePeriodicBurn(suite.ctx)

	queue2, found := suite.k.GetPeriodicBurnQueue(suite.ctx)
	suite.Require().True(found)
	suite.Require().True(queue2.Pending)
}

func (suite *IntegrationTestSuite) TestEnqueue_AfterRemoval() {
	suite.k.EnqueuePeriodicBurn(suite.ctx)
	suite.k.RemovePeriodicBurnQueue(suite.ctx)

	// After removal, enqueue should work again
	suite.k.EnqueuePeriodicBurn(suite.ctx)

	queue, found := suite.k.GetPeriodicBurnQueue(suite.ctx)
	suite.Require().True(found)
	suite.Require().True(queue.Pending)
}

// --- ProcessPeriodicBurnQueue tests ---

func (suite *IntegrationTestSuite) TestProcessQueue_NoQueue() {
	// No queue set - should return nil
	err := suite.k.ProcessPeriodicBurnQueue(suite.ctx)
	suite.Require().NoError(err)
}

func (suite *IntegrationTestSuite) TestProcessQueue_EmptyBalance() {
	suite.k.EnqueuePeriodicBurn(suite.ctx)

	addr := sdk.AccAddress("moduleacc")
	moduleAcc := authtypes.ModuleAccount{
		BaseAccount: &authtypes.BaseAccount{Address: addr.String()},
		Name:        types.ModuleName,
	}

	suite.acc.EXPECT().GetModuleAccount(gomock.Any(), types.ModuleName).Return(&moduleAcc).Times(1)
	suite.bank.EXPECT().GetAllBalances(gomock.Any(), addr).Return(sdk.NewCoins()).Times(1)

	err := suite.k.ProcessPeriodicBurnQueue(suite.ctx)
	suite.Require().NoError(err)

	// Queue should be removed since balance was empty
	_, found := suite.k.GetPeriodicBurnQueue(suite.ctx)
	suite.Require().False(found)
}

func (suite *IntegrationTestSuite) TestProcessQueue_SingleBatch() {
	suite.k.EnqueuePeriodicBurn(suite.ctx)

	addr := sdk.AccAddress("moduleacc")
	moduleAcc := authtypes.ModuleAccount{
		BaseAccount: &authtypes.BaseAccount{Address: addr.String()},
		Name:        types.ModuleName,
	}

	coins := sdk.NewCoins(
		sdk.NewInt64Coin("ubze", 1000),
		sdk.NewInt64Coin("utoken", 500),
	)

	suite.acc.EXPECT().GetModuleAccount(gomock.Any(), types.ModuleName).Return(&moduleAcc).Times(1)
	suite.bank.EXPECT().GetAllBalances(gomock.Any(), addr).Return(coins).Times(1)
	suite.trade.EXPECT().IsNativeDenom(gomock.Any(), "ubze").Return(true).Times(1)
	suite.trade.EXPECT().IsNativeDenom(gomock.Any(), "utoken").Return(true).Times(1)
	suite.bank.EXPECT().BurnCoins(gomock.Any(), types.ModuleName, coins).Return(nil).Times(1)

	err := suite.k.ProcessPeriodicBurnQueue(suite.ctx)
	suite.Require().NoError(err)

	// Queue should be cleared after single batch
	_, found := suite.k.GetPeriodicBurnQueue(suite.ctx)
	suite.Require().False(found)
}

func (suite *IntegrationTestSuite) TestProcessQueue_MultipleBatches() {
	suite.k.EnqueuePeriodicBurn(suite.ctx)

	addr := sdk.AccAddress("moduleacc")
	moduleAcc := authtypes.ModuleAccount{
		BaseAccount: &authtypes.BaseAccount{Address: addr.String()},
		Name:        types.ModuleName,
	}

	// Create more coins than MaxDenomsBurnPerBlock
	var allCoins sdk.Coins
	for i := 0; i < types.MaxDenomsBurnPerBlock+5; i++ {
		allCoins = allCoins.Add(sdk.NewInt64Coin(fmt.Sprintf("denom%03d", i), 100))
	}

	firstBatch := allCoins[:types.MaxDenomsBurnPerBlock]

	suite.acc.EXPECT().GetModuleAccount(gomock.Any(), types.ModuleName).Return(&moduleAcc).Times(1)
	suite.bank.EXPECT().GetAllBalances(gomock.Any(), addr).Return(allCoins).Times(1)

	// Mock for each denom in first batch
	for _, c := range firstBatch {
		suite.trade.EXPECT().IsNativeDenom(gomock.Any(), c.Denom).Return(true).Times(1)
	}
	suite.bank.EXPECT().BurnCoins(gomock.Any(), types.ModuleName, firstBatch).Return(nil).Times(1)

	err := suite.k.ProcessPeriodicBurnQueue(suite.ctx)
	suite.Require().NoError(err)

	// Queue should still be pending since we had more coins than one batch
	queue, found := suite.k.GetPeriodicBurnQueue(suite.ctx)
	suite.Require().True(found)
	suite.Require().True(queue.Pending)
}

func (suite *IntegrationTestSuite) TestProcessQueue_BurnErrorRetries() {
	suite.k.EnqueuePeriodicBurn(suite.ctx)

	addr := sdk.AccAddress("moduleacc")
	moduleAcc := authtypes.ModuleAccount{
		BaseAccount: &authtypes.BaseAccount{Address: addr.String()},
		Name:        types.ModuleName,
	}

	coins := sdk.NewCoins(sdk.NewInt64Coin("ubze", 1000))

	suite.acc.EXPECT().GetModuleAccount(gomock.Any(), types.ModuleName).Return(&moduleAcc).Times(1)
	suite.bank.EXPECT().GetAllBalances(gomock.Any(), addr).Return(coins).Times(1)
	suite.trade.EXPECT().IsNativeDenom(gomock.Any(), "ubze").Return(true).Times(1)
	suite.bank.EXPECT().BurnCoins(gomock.Any(), types.ModuleName, coins).Return(errors.New("burn failed")).Times(1)

	err := suite.k.ProcessPeriodicBurnQueue(suite.ctx)
	suite.Require().Error(err)

	// Queue should still be pending (caller wraps in ApplyFuncIfNoError which reverts state)
	queue, found := suite.k.GetPeriodicBurnQueue(suite.ctx)
	suite.Require().True(found)
	suite.Require().True(queue.Pending)
}

func (suite *IntegrationTestSuite) TestProcessQueue_MixedCoinTypes() {
	suite.k.EnqueuePeriodicBurn(suite.ctx)

	addr := sdk.AccAddress("moduleacc")
	moduleAcc := authtypes.ModuleAccount{
		BaseAccount: &authtypes.BaseAccount{Address: addr.String()},
		Name:        types.ModuleName,
	}

	nativeCoin := sdk.NewInt64Coin("ubze", 1000)
	lpCoin := sdk.NewInt64Coin("ulp_token1", 300)
	ibcCoin := sdk.NewInt64Coin("ibc/ABC123", 200)
	coins := sdk.NewCoins(nativeCoin, lpCoin, ibcCoin)

	addedCoins := sdk.NewCoins(sdk.NewInt64Coin("ibc/ABC123", 180))
	refundedCoins := sdk.NewCoins()

	suite.acc.EXPECT().GetModuleAccount(gomock.Any(), types.ModuleName).Return(&moduleAcc).Times(1)
	suite.bank.EXPECT().GetAllBalances(gomock.Any(), addr).Return(coins).Times(1)

	suite.trade.EXPECT().IsNativeDenom(gomock.Any(), "ubze").Return(true).Times(1)
	suite.trade.EXPECT().IsNativeDenom(gomock.Any(), "ulp_token1").Return(false).Times(1)
	suite.trade.EXPECT().IsNativeDenom(gomock.Any(), "ibc/ABC123").Return(false).Times(1)
	suite.trade.EXPECT().CanSwapForNativeDenom(gomock.Any(), ibcCoin).Return(true).Times(1)
	suite.trade.EXPECT().ModuleAddLiquidityWithNativeDenom(gomock.Any(), types.ModuleName, sdk.NewCoins(ibcCoin)).Return(addedCoins, refundedCoins, nil).Times(1)
	suite.bank.EXPECT().SendCoinsFromModuleToModule(gomock.Any(), types.ModuleName, types.BlackHoleModuleName, sdk.NewCoins(lpCoin)).Return(nil).Times(1)
	suite.bank.EXPECT().BurnCoins(gomock.Any(), types.ModuleName, sdk.NewCoins(nativeCoin)).Return(nil).Times(1)

	err := suite.k.ProcessPeriodicBurnQueue(suite.ctx)
	suite.Require().NoError(err)

	// Queue should be cleared
	_, found := suite.k.GetPeriodicBurnQueue(suite.ctx)
	suite.Require().False(found)
}

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

	_, err := suite.k.BurnAnyCoins(suite.ctx, fromModule, coins)
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

	_, err := suite.k.BurnAnyCoins(suite.ctx, fromModule, coins)
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

	_, err := suite.k.BurnAnyCoins(suite.ctx, fromModule, coins)
	suite.Require().NoError(err)
}

func (suite *IntegrationTestSuite) TestBurn_TestBurnAnyCoins_OnlyIBCCoins() {
	fromModule := "test_module"
	coins := sdk.NewCoins(
		sdk.NewInt64Coin("ibc/ABC123", 1000),
		sdk.NewInt64Coin("ibc/DEF456", 500),
	)

	addedCoins := sdk.NewCoins(
		sdk.NewInt64Coin("ibc/ABC123", 900),
		sdk.NewInt64Coin("ibc/DEF456", 450),
	)
	refundedCoins := sdk.NewCoins() // No refunds in this case

	// Mock native denom checks
	suite.trade.EXPECT().IsNativeDenom(suite.ctx, "ibc/ABC123").Return(false).Times(1)
	suite.trade.EXPECT().IsNativeDenom(suite.ctx, "ibc/DEF456").Return(false).Times(1)

	// Mock swap capability checks
	suite.trade.EXPECT().CanSwapForNativeDenom(suite.ctx, coins[0]).Return(true).Times(1)
	suite.trade.EXPECT().CanSwapForNativeDenom(suite.ctx, coins[1]).Return(true).Times(1)

	// Mock add liquidity operation - now uses ModuleAddLiquidityWithNativeDenom instead of swap
	suite.trade.EXPECT().ModuleAddLiquidityWithNativeDenom(suite.ctx, fromModule, coins).Return(addedCoins, refundedCoins, nil).Times(1)

	// No burn operation expected - IBC coins are added to liquidity, LP tokens are locked

	_, err := suite.k.BurnAnyCoins(suite.ctx, fromModule, coins)
	suite.Require().NoError(err)
}

func (suite *IntegrationTestSuite) TestBurn_TestBurnAnyCoins_MixedCoins() {
	fromModule := "test_module"
	nativeCoin := sdk.NewInt64Coin("ubze", 1000)
	factoryCoin := sdk.NewInt64Coin("factory/creator/token", 500)
	lpCoin := sdk.NewInt64Coin("ulp_token1", 300)
	ibcCoin := sdk.NewInt64Coin("ibc/ABC123", 200)

	coins := sdk.NewCoins(nativeCoin, factoryCoin, lpCoin, ibcCoin)
	addedCoins := sdk.NewCoins(sdk.NewInt64Coin("ibc/ABC123", 180)) // IBC coins added to liquidity
	refundedCoins := sdk.NewCoins()                                 // No refunds
	expectedBurnCoins := sdk.NewCoins(nativeCoin, factoryCoin)
	lockableCoins := sdk.NewCoins(lpCoin)

	// Mock native denom checks
	suite.trade.EXPECT().IsNativeDenom(suite.ctx, "ubze").Return(true).Times(1)
	suite.trade.EXPECT().IsNativeDenom(suite.ctx, "factory/creator/token").Return(false).Times(1)
	suite.trade.EXPECT().IsNativeDenom(suite.ctx, "ulp_token1").Return(false).Times(1)
	suite.trade.EXPECT().IsNativeDenom(suite.ctx, "ibc/ABC123").Return(false).Times(1)

	// Mock swap capability check for IBC coin
	suite.trade.EXPECT().CanSwapForNativeDenom(suite.ctx, ibcCoin).Return(true).Times(1)

	// Mock add liquidity operation
	suite.trade.EXPECT().ModuleAddLiquidityWithNativeDenom(suite.ctx, fromModule, sdk.NewCoins(ibcCoin)).Return(addedCoins, refundedCoins, nil).Times(1)

	// Mock send LP tokens to black hole
	suite.bank.EXPECT().SendCoinsFromModuleToModule(suite.ctx, fromModule, types.BlackHoleModuleName, lockableCoins).Return(nil).Times(1)

	// Mock burn operation (only native and factory tokens)
	suite.bank.EXPECT().BurnCoins(suite.ctx, fromModule, expectedBurnCoins).Return(nil).Times(1)

	_, err := suite.k.BurnAnyCoins(suite.ctx, fromModule, coins)
	suite.Require().NoError(err)
}

func (suite *IntegrationTestSuite) TestBurn_TestBurnAnyCoins_EmptyCoins() {
	fromModule := "test_module"
	coins := sdk.NewCoins()

	// With empty coins, no burn operation should be called since IsAllPositive() returns false for empty coins
	// No mocks needed as the function should return early

	_, err := suite.k.BurnAnyCoins(suite.ctx, fromModule, coins)
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

	_, err := suite.k.BurnAnyCoins(suite.ctx, fromModule, coins)
	suite.Require().NoError(err)
}

func (suite *IntegrationTestSuite) TestBurn_TestBurnAnyCoins_AddLiquidityError() {
	fromModule := "test_module"
	coins := sdk.NewCoins(sdk.NewInt64Coin("ibc/ABC123", 1000))
	addLiquidityError := errors.New("add liquidity failed")

	// Mock native denom check
	suite.trade.EXPECT().IsNativeDenom(suite.ctx, "ibc/ABC123").Return(false).Times(1)

	// Mock swap capability check
	suite.trade.EXPECT().CanSwapForNativeDenom(suite.ctx, coins[0]).Return(true).Times(1)

	// Mock add liquidity operation failure - the function continues even on error
	suite.trade.EXPECT().ModuleAddLiquidityWithNativeDenom(suite.ctx, fromModule, coins).Return(nil, nil, addLiquidityError).Times(1)

	// Should not error - the function logs the error and continues
	_, err := suite.k.BurnAnyCoins(suite.ctx, fromModule, coins)
	suite.Require().NoError(err)
}

func (suite *IntegrationTestSuite) TestBurn_TestBurnAnyCoins_SendToBlackHoleError() {
	fromModule := "test_module"
	coins := sdk.NewCoins(sdk.NewInt64Coin("ulp_token1", 1000))
	sendError := errors.New("send failed")

	// Mock native denom check
	suite.trade.EXPECT().IsNativeDenom(suite.ctx, "ulp_token1").Return(false).Times(1)

	// Mock send to black hole failure
	suite.bank.EXPECT().SendCoinsFromModuleToModule(suite.ctx, fromModule, types.BlackHoleModuleName, coins).Return(sendError).Times(1)

	_, err := suite.k.BurnAnyCoins(suite.ctx, fromModule, coins)
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

	_, err := suite.k.BurnAnyCoins(suite.ctx, fromModule, coins)
	suite.Require().Error(err)
	suite.Require().Equal(burnError, err)
}

func (suite *IntegrationTestSuite) TestBurn_TestBurnAnyCoins_UnknownDenomLocked() {
	fromModule := "test_module"
	coins := sdk.NewCoins(sdk.NewInt64Coin("unknown/denom", 1000))

	// Mock native denom check
	suite.trade.EXPECT().IsNativeDenom(suite.ctx, "unknown/denom").Return(false).Times(1)

	// Mock swap capability check (returns false)
	suite.trade.EXPECT().CanSwapForNativeDenom(suite.ctx, coins[0]).Return(false).Times(1)

	// Unknown denoms that are not swappable are now locked (sent to black hole)
	suite.bank.EXPECT().SendCoinsFromModuleToModule(suite.ctx, fromModule, types.BlackHoleModuleName, coins).Return(nil).Times(1)

	burned, err := suite.k.BurnAnyCoins(suite.ctx, fromModule, coins)
	suite.Require().NoError(err)
	// Unknown denoms are locked, not burned - so burned should be empty
	suite.Require().True(burned.IsZero())
}

func (suite *IntegrationTestSuite) TestBurn_TestBurnAnyCoins_IBCNotSwappable() {
	fromModule := "test_module"
	ibcCoin := sdk.NewInt64Coin("ibc/NONSWAPPABLE", 1000)
	coins := sdk.NewCoins(ibcCoin)

	// Mock native denom check
	suite.trade.EXPECT().IsNativeDenom(suite.ctx, "ibc/NONSWAPPABLE").Return(false).Times(1)

	// Mock swap capability check (returns false)
	suite.trade.EXPECT().CanSwapForNativeDenom(suite.ctx, ibcCoin).Return(false).Times(1)

	// IBC coins that cannot be swapped are now locked (sent to black hole)
	suite.bank.EXPECT().SendCoinsFromModuleToModule(suite.ctx, fromModule, types.BlackHoleModuleName, coins).Return(nil).Times(1)

	burned, err := suite.k.BurnAnyCoins(suite.ctx, fromModule, coins)
	suite.Require().NoError(err)
	// Non-swappable IBC coins are locked, not burned
	suite.Require().True(burned.IsZero())
}
