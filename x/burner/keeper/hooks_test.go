package keeper_test

import (
	"github.com/bze-alphateam/bze/x/burner/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"go.uber.org/mock/gomock"
)

func (suite *IntegrationTestSuite) TestHooks_TestGetBurnerPeriodicBurnHook_ValidExecution() {
	hook := suite.k.GetBurnerPeriodicBurnHook()

	// Verify hook properties
	suite.Require().Equal("periodic_burner", hook.GetName())

	// Queue should not exist before hook
	_, found := suite.k.GetPeriodicBurnQueue(suite.ctx)
	suite.Require().False(found)

	// Execute hook with correct epoch and interval - should only enqueue, not process
	err := hook.AfterEpochEnd(suite.ctx, "week", 4) // 4 % 4 == 0
	suite.Require().NoError(err)

	// Verify queue was set to pending (not processed directly)
	queue, found := suite.k.GetPeriodicBurnQueue(suite.ctx)
	suite.Require().True(found)
	suite.Require().True(queue.Pending)
}

func (suite *IntegrationTestSuite) TestHooks_TestGetBurnerPeriodicBurnHook_WrongEpoch() {
	hook := suite.k.GetBurnerPeriodicBurnHook()

	// Execute hook with wrong epoch identifier (should be "week")
	err := hook.AfterEpochEnd(suite.ctx, "wrong_epoch", 4)

	suite.Require().NoError(err) // Should return nil without error
	// No mock expectations because function should return early
}

func (suite *IntegrationTestSuite) TestHooks_TestGetBurnerPeriodicBurnHook_WrongInterval() {
	hook := suite.k.GetBurnerPeriodicBurnHook()
	params := suite.k.GetParams(suite.ctx)
	params.PeriodicBurningWeeks = 4
	suite.Require().NoError(suite.k.SetParams(suite.ctx, params))

	// Execute hook with epoch number not divisible by BurnInterval (4)
	err := hook.AfterEpochEnd(suite.ctx, "week", 3) // 3 % 4 != 0

	suite.Require().NoError(err) // Should return nil without error
	// No mock expectations because function should return early
}

func (suite *IntegrationTestSuite) TestHooks_TestGetBurnerRaffleCleanupHook_ValidExecution() {
	hook := suite.k.GetBurnerRaffleCleanupHook()

	// Verify hook properties
	suite.Require().Equal("burner_raffle_cleanup", hook.GetName())

	epochNumber := int64(100)

	// Queue should not exist before hook
	_, found := suite.k.GetRaffleCleanupQueue(suite.ctx)
	suite.Require().False(found)

	// Execute hook - should only enqueue, no bank/account/trade mocks needed
	err := hook.AfterEpochEnd(suite.ctx, "hour", epochNumber)
	suite.Require().NoError(err)

	// Verify queue was set with the epoch
	queue, found := suite.k.GetRaffleCleanupQueue(suite.ctx)
	suite.Require().True(found)
	suite.Require().Len(queue.PendingEpochs, 1)
	suite.Require().Equal(uint64(epochNumber), queue.PendingEpochs[0])
}

func (suite *IntegrationTestSuite) TestHooks_TestGetBurnerRaffleCleanupHook_WrongEpoch() {
	hook := suite.k.GetBurnerRaffleCleanupHook()

	// Execute hook with wrong epoch identifier (should be "hour")
	err := hook.AfterEpochEnd(suite.ctx, "wrong_epoch", 100)

	suite.Require().NoError(err) // Should return nil without error
	// No mock expectations because function should return early
}

func (suite *IntegrationTestSuite) TestHooks_TestGetBurnerRaffleCleanupHook_NoRafflesToDelete() {
	hook := suite.k.GetBurnerRaffleCleanupHook()

	// Execute hook with no raffles to delete - hook still enqueues
	err := hook.AfterEpochEnd(suite.ctx, "hour", 100)

	suite.Require().NoError(err)

	// Verify epoch was enqueued (empty case is handled by queue processor)
	queue, found := suite.k.GetRaffleCleanupQueue(suite.ctx)
	suite.Require().True(found)
	suite.Require().Len(queue.PendingEpochs, 1)
}

func (suite *IntegrationTestSuite) TestHooks_PeriodicBurnHook_EnqueueAndProcess() {
	// Hook should only enqueue - verify no bank/trade operations happen during hook
	hook := suite.k.GetBurnerPeriodicBurnHook()
	err := hook.AfterEpochEnd(suite.ctx, "week", 4)
	suite.Require().NoError(err)

	// Verify queue was set
	queue, found := suite.k.GetPeriodicBurnQueue(suite.ctx)
	suite.Require().True(found)
	suite.Require().True(queue.Pending)

	// Now test processing via ProcessPeriodicBurnQueue
	addr := sdk.AccAddress("moduleacc")
	moduleAcc := authtypes.ModuleAccount{
		BaseAccount: &authtypes.BaseAccount{
			Address: addr.String(),
		},
		Name: types.ModuleName,
	}

	allCoins := sdk.NewCoins(
		sdk.NewInt64Coin("ubze", 1000),
		sdk.NewInt64Coin("utoken", 500),
		sdk.NewInt64Coin("ibc/ABC123", 200),
	)

	addedCoins := sdk.NewCoins(sdk.NewInt64Coin("ibc/ABC123", 180))
	refundedCoins := sdk.NewCoins()
	expectedBurnCoins := sdk.NewCoins(
		sdk.NewInt64Coin("ubze", 1000),
		sdk.NewInt64Coin("utoken", 500),
	)

	suite.acc.EXPECT().GetModuleAccount(gomock.Any(), types.ModuleName).Return(&moduleAcc).Times(1)
	suite.bank.EXPECT().GetAllBalances(gomock.Any(), addr).Return(allCoins).Times(1)

	suite.trade.EXPECT().IsNativeDenom(gomock.Any(), "ubze").Return(true).Times(1)
	suite.trade.EXPECT().IsNativeDenom(gomock.Any(), "utoken").Return(true).Times(1)
	suite.trade.EXPECT().IsNativeDenom(gomock.Any(), "ibc/ABC123").Return(false).Times(1)
	suite.trade.EXPECT().CanSwapForNativeDenom(gomock.Any(), sdk.NewInt64Coin("ibc/ABC123", 200)).Return(true).Times(1)

	suite.trade.EXPECT().ModuleAddLiquidityWithNativeDenom(gomock.Any(), types.ModuleName, sdk.NewCoins(sdk.NewInt64Coin("ibc/ABC123", 200))).Return(addedCoins, refundedCoins, nil).Times(1)
	suite.bank.EXPECT().BurnCoins(gomock.Any(), types.ModuleName, expectedBurnCoins).Return(nil).Times(1)

	err = suite.k.ProcessPeriodicBurnQueue(suite.ctx)
	suite.Require().NoError(err)

	// Queue should be cleared after processing
	_, found = suite.k.GetPeriodicBurnQueue(suite.ctx)
	suite.Require().False(found)
}

func (suite *IntegrationTestSuite) TestHooks_PeriodicBurnHook_EmptyBalance() {
	// Enqueue via hook
	hook := suite.k.GetBurnerPeriodicBurnHook()
	err := hook.AfterEpochEnd(suite.ctx, "week", 4)
	suite.Require().NoError(err)

	// Process queue with empty balance
	addr := sdk.AccAddress("moduleacc")
	moduleAcc := authtypes.ModuleAccount{
		BaseAccount: &authtypes.BaseAccount{
			Address: addr.String(),
		},
		Name: types.ModuleName,
	}

	suite.acc.EXPECT().GetModuleAccount(gomock.Any(), types.ModuleName).Return(&moduleAcc).Times(1)
	suite.bank.EXPECT().GetAllBalances(gomock.Any(), addr).Return(sdk.NewCoins()).Times(1)

	err = suite.k.ProcessPeriodicBurnQueue(suite.ctx)
	suite.Require().NoError(err)

	// Queue should be cleared since balance was empty
	_, found := suite.k.GetPeriodicBurnQueue(suite.ctx)
	suite.Require().False(found)
}

func (suite *IntegrationTestSuite) TestHooks_PeriodicBurnHook_OnlyIBCTokens() {
	// Enqueue via hook
	hook := suite.k.GetBurnerPeriodicBurnHook()
	err := hook.AfterEpochEnd(suite.ctx, "week", 4)
	suite.Require().NoError(err)

	// Process queue with only IBC tokens
	addr := sdk.AccAddress("moduleacc")
	moduleAcc := authtypes.ModuleAccount{
		BaseAccount: &authtypes.BaseAccount{
			Address: addr.String(),
		},
		Name: types.ModuleName,
	}

	ibcOnlyCoins := sdk.NewCoins(
		sdk.NewInt64Coin("ibc/ABC123", 1000),
		sdk.NewInt64Coin("ibc/DEF456", 500),
	)

	addedCoins := sdk.NewCoins(
		sdk.NewInt64Coin("ibc/ABC123", 900),
		sdk.NewInt64Coin("ibc/DEF456", 450),
	)
	refundedCoins := sdk.NewCoins()

	suite.acc.EXPECT().GetModuleAccount(gomock.Any(), types.ModuleName).Return(&moduleAcc).Times(1)
	suite.bank.EXPECT().GetAllBalances(gomock.Any(), addr).Return(ibcOnlyCoins).Times(1)

	suite.trade.EXPECT().IsNativeDenom(gomock.Any(), "ibc/ABC123").Return(false).Times(1)
	suite.trade.EXPECT().IsNativeDenom(gomock.Any(), "ibc/DEF456").Return(false).Times(1)
	suite.trade.EXPECT().CanSwapForNativeDenom(gomock.Any(), ibcOnlyCoins[0]).Return(true).Times(1)
	suite.trade.EXPECT().CanSwapForNativeDenom(gomock.Any(), ibcOnlyCoins[1]).Return(true).Times(1)
	suite.trade.EXPECT().ModuleAddLiquidityWithNativeDenom(gomock.Any(), types.ModuleName, ibcOnlyCoins).Return(addedCoins, refundedCoins, nil).Times(1)

	err = suite.k.ProcessPeriodicBurnQueue(suite.ctx)
	suite.Require().NoError(err)

	// Queue should be cleared
	_, found := suite.k.GetPeriodicBurnQueue(suite.ctx)
	suite.Require().False(found)
}

func (suite *IntegrationTestSuite) TestHooks_TestEpochHook_BeforeEpochStart() {
	// Test that BeforeEpochStart does nothing for both hooks
	periodicHook := suite.k.GetBurnerPeriodicBurnHook()
	err := periodicHook.BeforeEpochStart(suite.ctx, "any_epoch", 123)
	suite.Require().NoError(err)

	raffleHook := suite.k.GetBurnerRaffleCleanupHook()
	err = raffleHook.BeforeEpochStart(suite.ctx, "any_epoch", 456)
	suite.Require().NoError(err)
}

func (suite *IntegrationTestSuite) TestHooks_TestEpochHook_Names() {
	// Test hook names
	periodicHook := suite.k.GetBurnerPeriodicBurnHook()
	suite.Require().Equal("periodic_burner", periodicHook.GetName())

	raffleHook := suite.k.GetBurnerRaffleCleanupHook()
	suite.Require().Equal("burner_raffle_cleanup", raffleHook.GetName())
}
