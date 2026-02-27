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

// --- EnqueueRaffleCleanup tests ---

func (suite *IntegrationTestSuite) TestEnqueueRaffleCleanup_SetsQueue() {
	_, found := suite.k.GetRaffleCleanupQueue(suite.ctx)
	suite.Require().False(found)

	suite.k.EnqueueRaffleCleanup(suite.ctx, 100)

	queue, found := suite.k.GetRaffleCleanupQueue(suite.ctx)
	suite.Require().True(found)
	suite.Require().Len(queue.PendingEpochs, 1)
	suite.Require().Equal(uint64(100), queue.PendingEpochs[0])
}

func (suite *IntegrationTestSuite) TestEnqueueRaffleCleanup_Idempotent() {
	suite.k.EnqueueRaffleCleanup(suite.ctx, 100)
	suite.k.EnqueueRaffleCleanup(suite.ctx, 100)

	queue, found := suite.k.GetRaffleCleanupQueue(suite.ctx)
	suite.Require().True(found)
	suite.Require().Len(queue.PendingEpochs, 1)
	suite.Require().Equal(uint64(100), queue.PendingEpochs[0])
}

func (suite *IntegrationTestSuite) TestEnqueueRaffleCleanup_MultipleEpochs() {
	suite.k.EnqueueRaffleCleanup(suite.ctx, 100)
	suite.k.EnqueueRaffleCleanup(suite.ctx, 200)
	suite.k.EnqueueRaffleCleanup(suite.ctx, 300)

	queue, found := suite.k.GetRaffleCleanupQueue(suite.ctx)
	suite.Require().True(found)
	suite.Require().Len(queue.PendingEpochs, 3)
	suite.Require().Equal(uint64(100), queue.PendingEpochs[0])
	suite.Require().Equal(uint64(200), queue.PendingEpochs[1])
	suite.Require().Equal(uint64(300), queue.PendingEpochs[2])
}

// --- ProcessRaffleCleanupQueue tests ---

func (suite *IntegrationTestSuite) TestProcessRaffleCleanupQueue_NoQueue() {
	err := suite.k.ProcessRaffleCleanupQueue(suite.ctx)
	suite.Require().NoError(err)
}

func (suite *IntegrationTestSuite) TestProcessRaffleCleanupQueue_EmptyEpoch() {
	// Enqueue an epoch with no delete hooks
	suite.k.EnqueueRaffleCleanup(suite.ctx, 100)

	err := suite.k.ProcessRaffleCleanupQueue(suite.ctx)
	suite.Require().NoError(err)

	// Queue should be removed since the epoch had no raffles
	_, found := suite.k.GetRaffleCleanupQueue(suite.ctx)
	suite.Require().False(found)
}

func (suite *IntegrationTestSuite) TestProcessRaffleCleanupQueue_SingleRaffle() {
	epochNumber := uint64(100)
	denom := "utoken"

	// Set up raffle delete hook
	deleteHook := types.RaffleDeleteHook{
		Denom: denom,
		EndAt: epochNumber,
	}
	suite.k.SetRaffleDeleteHook(suite.ctx, deleteHook)

	// Set up raffle
	raffle := types.Raffle{
		Denom: denom,
		Pot:   "1000",
	}
	suite.k.SetRaffle(suite.ctx, raffle)

	// Set up raffle winner
	winner := types.RaffleWinner{
		Index:  "1",
		Denom:  denom,
		Amount: "100",
		Winner: "winner1",
	}
	suite.k.SetRaffleWinner(suite.ctx, winner)

	// Enqueue
	suite.k.EnqueueRaffleCleanup(suite.ctx, epochNumber)

	// Mock module account with coins
	addr := sdk.AccAddress("raffleacc")
	raffleAcc := authtypes.ModuleAccount{
		BaseAccount: &authtypes.BaseAccount{
			Address: addr.String(),
		},
		Name: types.RaffleModuleName,
	}

	currentPot := sdk.NewInt64Coin(denom, 1000)

	suite.acc.EXPECT().GetModuleAccount(suite.ctx, types.RaffleModuleName).Return(&raffleAcc).Times(1)
	suite.bank.EXPECT().GetBalance(suite.ctx, addr, denom).Return(currentPot).Times(1)
	suite.trade.EXPECT().IsNativeDenom(suite.ctx, denom).Return(true).Times(1)
	suite.bank.EXPECT().BurnCoins(suite.ctx, types.RaffleModuleName, sdk.NewCoins(currentPot)).Return(nil).Times(1)

	err := suite.k.ProcessRaffleCleanupQueue(suite.ctx)
	suite.Require().NoError(err)

	// Verify cleanup happened
	_, found := suite.k.GetRaffle(suite.ctx, denom)
	suite.Require().False(found)

	winners := suite.k.GetRaffleWinners(suite.ctx, denom)
	suite.Require().Len(winners, 0)

	hooks := suite.k.GetRaffleDeleteHookByEndAtPrefix(suite.ctx, epochNumber)
	suite.Require().Len(hooks, 0)

	// Queue should be removed after processing single raffle
	_, found = suite.k.GetRaffleCleanupQueue(suite.ctx)
	suite.Require().False(found)
}

func (suite *IntegrationTestSuite) TestProcessRaffleCleanupQueue_MultipleBatches() {
	epochNumber := uint64(100)

	// Create more raffles than MaxRafflesCleanupPerBlock
	numRaffles := types.MaxRafflesCleanupPerBlock + 5
	for i := 0; i < numRaffles; i++ {
		denom := fmt.Sprintf("denom%d", i)
		suite.k.SetRaffleDeleteHook(suite.ctx, types.RaffleDeleteHook{
			Denom: denom,
			EndAt: epochNumber,
		})
		suite.k.SetRaffle(suite.ctx, types.Raffle{
			Denom: denom,
			Pot:   "0",
		})
	}

	suite.k.EnqueueRaffleCleanup(suite.ctx, epochNumber)

	// Mock for each raffle in the batch - module account not found to simplify
	suite.acc.EXPECT().GetModuleAccount(suite.ctx, types.RaffleModuleName).Return(nil).Times(types.MaxRafflesCleanupPerBlock)

	err := suite.k.ProcessRaffleCleanupQueue(suite.ctx)
	suite.Require().NoError(err)

	// Queue should still exist since we had more raffles than one batch
	queue, found := suite.k.GetRaffleCleanupQueue(suite.ctx)
	suite.Require().True(found)
	suite.Require().Len(queue.PendingEpochs, 1)
	suite.Require().Equal(epochNumber, queue.PendingEpochs[0])

	// Verify remaining raffles
	remaining := suite.k.GetRaffleDeleteHookByEndAtPrefix(suite.ctx, epochNumber)
	suite.Require().Equal(numRaffles-types.MaxRafflesCleanupPerBlock, len(remaining))
}

func (suite *IntegrationTestSuite) TestProcessRaffleCleanupQueue_MultipleEpochs() {
	epoch1 := uint64(100)
	epoch2 := uint64(200)

	// Set up raffle for epoch 1
	suite.k.SetRaffleDeleteHook(suite.ctx, types.RaffleDeleteHook{Denom: "denom1", EndAt: epoch1})
	suite.k.SetRaffle(suite.ctx, types.Raffle{Denom: "denom1", Pot: "0"})

	// Set up raffle for epoch 2
	suite.k.SetRaffleDeleteHook(suite.ctx, types.RaffleDeleteHook{Denom: "denom2", EndAt: epoch2})
	suite.k.SetRaffle(suite.ctx, types.Raffle{Denom: "denom2", Pot: "0"})

	suite.k.EnqueueRaffleCleanup(suite.ctx, epoch1)
	suite.k.EnqueueRaffleCleanup(suite.ctx, epoch2)

	// Process first epoch
	suite.acc.EXPECT().GetModuleAccount(suite.ctx, types.RaffleModuleName).Return(nil).Times(1)

	err := suite.k.ProcessRaffleCleanupQueue(suite.ctx)
	suite.Require().NoError(err)

	// First epoch should be processed, second epoch remains
	queue, found := suite.k.GetRaffleCleanupQueue(suite.ctx)
	suite.Require().True(found)
	suite.Require().Len(queue.PendingEpochs, 1)
	suite.Require().Equal(epoch2, queue.PendingEpochs[0])

	// Process second epoch
	suite.acc.EXPECT().GetModuleAccount(suite.ctx, types.RaffleModuleName).Return(nil).Times(1)

	err = suite.k.ProcessRaffleCleanupQueue(suite.ctx)
	suite.Require().NoError(err)

	// Queue should be removed
	_, found = suite.k.GetRaffleCleanupQueue(suite.ctx)
	suite.Require().False(found)
}

func (suite *IntegrationTestSuite) TestProcessRaffleCleanupQueue_BurnError() {
	epochNumber := uint64(100)
	denom := "utoken"

	suite.k.SetRaffleDeleteHook(suite.ctx, types.RaffleDeleteHook{Denom: denom, EndAt: epochNumber})
	suite.k.SetRaffle(suite.ctx, types.Raffle{Denom: denom, Pot: "1000"})
	suite.k.EnqueueRaffleCleanup(suite.ctx, epochNumber)

	addr := sdk.AccAddress("raffleacc")
	raffleAcc := authtypes.ModuleAccount{
		BaseAccount: &authtypes.BaseAccount{Address: addr.String()},
		Name:        types.RaffleModuleName,
	}
	currentPot := sdk.NewInt64Coin(denom, 1000)

	suite.acc.EXPECT().GetModuleAccount(suite.ctx, types.RaffleModuleName).Return(&raffleAcc).Times(1)
	suite.bank.EXPECT().GetBalance(suite.ctx, addr, denom).Return(currentPot).Times(1)
	suite.trade.EXPECT().IsNativeDenom(suite.ctx, denom).Return(true).Times(1)
	suite.bank.EXPECT().BurnCoins(suite.ctx, types.RaffleModuleName, sdk.NewCoins(currentPot)).Return(errors.New("burn failed")).Times(1)

	err := suite.k.ProcessRaffleCleanupQueue(suite.ctx)
	suite.Require().NoError(err) // Processing continues despite burn error

	// Raffle and hooks should still be cleaned up
	_, found := suite.k.GetRaffle(suite.ctx, denom)
	suite.Require().False(found)

	hooks := suite.k.GetRaffleDeleteHookByEndAtPrefix(suite.ctx, epochNumber)
	suite.Require().Len(hooks, 0)

	// Queue should be removed
	_, found = suite.k.GetRaffleCleanupQueue(suite.ctx)
	suite.Require().False(found)
}

func (suite *IntegrationTestSuite) TestProcessRaffleCleanupQueue_ModuleAccountNotFound() {
	epochNumber := uint64(100)
	denom := "utoken"

	suite.k.SetRaffleDeleteHook(suite.ctx, types.RaffleDeleteHook{Denom: denom, EndAt: epochNumber})
	suite.k.SetRaffle(suite.ctx, types.Raffle{Denom: denom, Pot: "1000"})
	suite.k.EnqueueRaffleCleanup(suite.ctx, epochNumber)

	// Mock module account not found
	suite.acc.EXPECT().GetModuleAccount(suite.ctx, types.RaffleModuleName).Return(nil).Times(1)

	err := suite.k.ProcessRaffleCleanupQueue(suite.ctx)
	suite.Require().NoError(err) // Processing continues

	// Raffle and delete hook should still be cleaned up
	_, found := suite.k.GetRaffle(suite.ctx, denom)
	suite.Require().False(found)

	hooks := suite.k.GetRaffleDeleteHookByEndAtPrefix(suite.ctx, epochNumber)
	suite.Require().Len(hooks, 0)
}

func (suite *IntegrationTestSuite) TestProcessRaffleCleanupQueue_NoCoinsToBurn() {
	epochNumber := uint64(100)
	denom := "utoken"

	suite.k.SetRaffleDeleteHook(suite.ctx, types.RaffleDeleteHook{Denom: denom, EndAt: epochNumber})
	suite.k.SetRaffle(suite.ctx, types.Raffle{Denom: denom, Pot: "0"})
	suite.k.EnqueueRaffleCleanup(suite.ctx, epochNumber)

	addr := sdk.AccAddress("raffleacc")
	raffleAcc := authtypes.ModuleAccount{
		BaseAccount: &authtypes.BaseAccount{Address: addr.String()},
		Name:        types.RaffleModuleName,
	}
	emptyCoin := sdk.NewInt64Coin(denom, 0)

	suite.acc.EXPECT().GetModuleAccount(suite.ctx, types.RaffleModuleName).Return(&raffleAcc).Times(1)
	suite.bank.EXPECT().GetBalance(suite.ctx, addr, denom).Return(emptyCoin).Times(1)

	err := suite.k.ProcessRaffleCleanupQueue(suite.ctx)
	suite.Require().NoError(err)

	// Raffle should be cleaned up
	_, found := suite.k.GetRaffle(suite.ctx, denom)
	suite.Require().False(found)

	// Queue should be removed
	_, found = suite.k.GetRaffleCleanupQueue(suite.ctx)
	suite.Require().False(found)
}

func (suite *IntegrationTestSuite) TestProcessRaffleCleanupQueue_ExactBatchBoundary() {
	epochNumber := uint64(100)

	// Create exactly MaxRafflesCleanupPerBlock raffles
	for i := 0; i < types.MaxRafflesCleanupPerBlock; i++ {
		denom := fmt.Sprintf("denom%d", i)
		suite.k.SetRaffleDeleteHook(suite.ctx, types.RaffleDeleteHook{
			Denom: denom,
			EndAt: epochNumber,
		})
		suite.k.SetRaffle(suite.ctx, types.Raffle{
			Denom: denom,
			Pot:   "0",
		})
	}

	suite.k.EnqueueRaffleCleanup(suite.ctx, epochNumber)

	// First call: processes exactly MaxRafflesCleanupPerBlock raffles
	suite.acc.EXPECT().GetModuleAccount(suite.ctx, types.RaffleModuleName).Return(nil).Times(types.MaxRafflesCleanupPerBlock)

	err := suite.k.ProcessRaffleCleanupQueue(suite.ctx)
	suite.Require().NoError(err)

	// len(batch) == MaxRafflesCleanupPerBlock, so the epoch is NOT popped yet
	queue, found := suite.k.GetRaffleCleanupQueue(suite.ctx)
	suite.Require().True(found)
	suite.Require().Len(queue.PendingEpochs, 1)
	suite.Require().Equal(epochNumber, queue.PendingEpochs[0])

	// All delete hooks were removed, so no remaining raffles
	remaining := suite.k.GetRaffleDeleteHookByEndAtPrefix(suite.ctx, epochNumber)
	suite.Require().Len(remaining, 0)

	// Second call: batch is empty, epoch gets popped
	err = suite.k.ProcessRaffleCleanupQueue(suite.ctx)
	suite.Require().NoError(err)

	_, found = suite.k.GetRaffleCleanupQueue(suite.ctx)
	suite.Require().False(found)
}

func (suite *IntegrationTestSuite) TestProcessRaffleCleanupQueue_FullDrainAcrossMultipleBlocks() {
	epochNumber := uint64(100)

	// Create 2.5x the batch size to require 3 processing rounds + 1 empty round
	numRaffles := types.MaxRafflesCleanupPerBlock*2 + types.MaxRafflesCleanupPerBlock/2
	for i := 0; i < numRaffles; i++ {
		denom := fmt.Sprintf("denom%d", i)
		suite.k.SetRaffleDeleteHook(suite.ctx, types.RaffleDeleteHook{
			Denom: denom,
			EndAt: epochNumber,
		})
		suite.k.SetRaffle(suite.ctx, types.Raffle{
			Denom: denom,
			Pot:   "0",
		})
	}

	suite.k.EnqueueRaffleCleanup(suite.ctx, epochNumber)

	// Block 1: processes MaxRafflesCleanupPerBlock raffles
	suite.acc.EXPECT().GetModuleAccount(suite.ctx, types.RaffleModuleName).Return(nil).Times(types.MaxRafflesCleanupPerBlock)
	err := suite.k.ProcessRaffleCleanupQueue(suite.ctx)
	suite.Require().NoError(err)

	remaining := suite.k.GetRaffleDeleteHookByEndAtPrefix(suite.ctx, epochNumber)
	suite.Require().Equal(numRaffles-types.MaxRafflesCleanupPerBlock, len(remaining))

	queue, found := suite.k.GetRaffleCleanupQueue(suite.ctx)
	suite.Require().True(found)
	suite.Require().Len(queue.PendingEpochs, 1)

	// Block 2: processes another MaxRafflesCleanupPerBlock raffles
	suite.acc.EXPECT().GetModuleAccount(suite.ctx, types.RaffleModuleName).Return(nil).Times(types.MaxRafflesCleanupPerBlock)
	err = suite.k.ProcessRaffleCleanupQueue(suite.ctx)
	suite.Require().NoError(err)

	remaining = suite.k.GetRaffleDeleteHookByEndAtPrefix(suite.ctx, epochNumber)
	suite.Require().Equal(numRaffles-types.MaxRafflesCleanupPerBlock*2, len(remaining))

	queue, found = suite.k.GetRaffleCleanupQueue(suite.ctx)
	suite.Require().True(found)
	suite.Require().Len(queue.PendingEpochs, 1)

	// Block 3: processes remaining half-batch (< MaxRafflesCleanupPerBlock), epoch is popped
	lastBatch := numRaffles - types.MaxRafflesCleanupPerBlock*2
	suite.acc.EXPECT().GetModuleAccount(suite.ctx, types.RaffleModuleName).Return(nil).Times(lastBatch)
	err = suite.k.ProcessRaffleCleanupQueue(suite.ctx)
	suite.Require().NoError(err)

	remaining = suite.k.GetRaffleDeleteHookByEndAtPrefix(suite.ctx, epochNumber)
	suite.Require().Len(remaining, 0)

	// Queue fully drained
	_, found = suite.k.GetRaffleCleanupQueue(suite.ctx)
	suite.Require().False(found)
}

func (suite *IntegrationTestSuite) TestProcessRaffleCleanupQueue_NewEpochEnqueuedDuringProcessing() {
	epoch1 := uint64(100)
	epoch2 := uint64(200)

	// Create more raffles than one batch for epoch1
	numRafflesEpoch1 := types.MaxRafflesCleanupPerBlock + 3
	for i := 0; i < numRafflesEpoch1; i++ {
		denom := fmt.Sprintf("epoch1_denom%d", i)
		suite.k.SetRaffleDeleteHook(suite.ctx, types.RaffleDeleteHook{
			Denom: denom,
			EndAt: epoch1,
		})
		suite.k.SetRaffle(suite.ctx, types.Raffle{
			Denom: denom,
			Pot:   "0",
		})
	}

	suite.k.EnqueueRaffleCleanup(suite.ctx, epoch1)

	// Block 1: process first batch of epoch1
	suite.acc.EXPECT().GetModuleAccount(suite.ctx, types.RaffleModuleName).Return(nil).Times(types.MaxRafflesCleanupPerBlock)
	err := suite.k.ProcessRaffleCleanupQueue(suite.ctx)
	suite.Require().NoError(err)

	// Epoch1 still in queue with remaining raffles
	queue, found := suite.k.GetRaffleCleanupQueue(suite.ctx)
	suite.Require().True(found)
	suite.Require().Len(queue.PendingEpochs, 1)

	// Now a new epoch arrives and gets enqueued (simulates epoch hook firing mid-drain)
	suite.k.SetRaffleDeleteHook(suite.ctx, types.RaffleDeleteHook{Denom: "epoch2_denom0", EndAt: epoch2})
	suite.k.SetRaffle(suite.ctx, types.Raffle{Denom: "epoch2_denom0", Pot: "0"})
	suite.k.EnqueueRaffleCleanup(suite.ctx, epoch2)

	// Verify FIFO: epoch1 is still first, epoch2 is appended
	queue, found = suite.k.GetRaffleCleanupQueue(suite.ctx)
	suite.Require().True(found)
	suite.Require().Len(queue.PendingEpochs, 2)
	suite.Require().Equal(epoch1, queue.PendingEpochs[0])
	suite.Require().Equal(epoch2, queue.PendingEpochs[1])

	// Block 2: continues draining epoch1 (remaining 3 raffles < batch size, epoch1 popped)
	suite.acc.EXPECT().GetModuleAccount(suite.ctx, types.RaffleModuleName).Return(nil).Times(3)
	err = suite.k.ProcessRaffleCleanupQueue(suite.ctx)
	suite.Require().NoError(err)

	// Epoch1 fully processed, only epoch2 remains
	queue, found = suite.k.GetRaffleCleanupQueue(suite.ctx)
	suite.Require().True(found)
	suite.Require().Len(queue.PendingEpochs, 1)
	suite.Require().Equal(epoch2, queue.PendingEpochs[0])

	remaining := suite.k.GetRaffleDeleteHookByEndAtPrefix(suite.ctx, epoch1)
	suite.Require().Len(remaining, 0)

	// Block 3: processes epoch2
	suite.acc.EXPECT().GetModuleAccount(suite.ctx, types.RaffleModuleName).Return(nil).Times(1)
	err = suite.k.ProcessRaffleCleanupQueue(suite.ctx)
	suite.Require().NoError(err)

	// All done
	_, found = suite.k.GetRaffleCleanupQueue(suite.ctx)
	suite.Require().False(found)

	remaining = suite.k.GetRaffleDeleteHookByEndAtPrefix(suite.ctx, epoch2)
	suite.Require().Len(remaining, 0)
}

func (suite *IntegrationTestSuite) TestProcessRaffleCleanupQueue_FactoryToken() {
	epochNumber := uint64(100)
	denom := "factory/creator/token"

	suite.k.SetRaffleDeleteHook(suite.ctx, types.RaffleDeleteHook{Denom: denom, EndAt: epochNumber})
	suite.k.SetRaffle(suite.ctx, types.Raffle{Denom: denom, Pot: "1000"})
	suite.k.EnqueueRaffleCleanup(suite.ctx, epochNumber)

	addr := sdk.AccAddress("raffleacc")
	raffleAcc := authtypes.ModuleAccount{
		BaseAccount: &authtypes.BaseAccount{Address: addr.String()},
		Name:        types.RaffleModuleName,
	}
	currentPot := sdk.NewInt64Coin(denom, 1000)

	suite.acc.EXPECT().GetModuleAccount(suite.ctx, types.RaffleModuleName).Return(&raffleAcc).Times(1)
	suite.bank.EXPECT().GetBalance(suite.ctx, addr, denom).Return(currentPot).Times(1)
	suite.trade.EXPECT().IsNativeDenom(suite.ctx, denom).Return(false).Times(1)
	suite.bank.EXPECT().BurnCoins(suite.ctx, types.RaffleModuleName, sdk.NewCoins(currentPot)).Return(nil).Times(1)

	err := suite.k.ProcessRaffleCleanupQueue(suite.ctx)
	suite.Require().NoError(err)

	// Verify burned coins were saved (factory tokens are burned, not excluded)
	burnedCoins := suite.k.GetAllBurnedCoins(suite.ctx)
	suite.Require().Len(burnedCoins, 1)

	// Queue should be removed
	_, found := suite.k.GetRaffleCleanupQueue(suite.ctx)
	suite.Require().False(found)
}
