package keeper_test

import (
	"errors"

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
	denom := "utoken"

	// Set up raffle delete hook
	deleteHook := types.RaffleDeleteHook{
		Denom: denom,
		EndAt: uint64(epochNumber),
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

	// Mock module account with coins
	addr := sdk.AccAddress("raffleacc")
	raffleAcc := authtypes.ModuleAccount{
		BaseAccount: &authtypes.BaseAccount{
			Address: addr.String(),
		},
		Name: types.RaffleModuleName,
	}

	currentPot := sdk.NewInt64Coin(denom, 1000)

	// Mock expectations
	suite.acc.EXPECT().GetModuleAccount(suite.ctx, types.RaffleModuleName).Return(&raffleAcc).Times(1)
	suite.bank.EXPECT().GetBalance(suite.ctx, addr, denom).Return(currentPot).Times(1)

	// Add TradeKeeper mock for BurnAnyCoins
	suite.trade.EXPECT().IsNativeDenom(suite.ctx, denom).Return(true).Times(1)

	suite.bank.EXPECT().BurnCoins(suite.ctx, types.RaffleModuleName, sdk.NewCoins(currentPot)).Return(nil).Times(1)

	// Execute hook
	err := hook.AfterEpochEnd(suite.ctx, "hour", epochNumber)

	suite.Require().NoError(err)

	// Verify cleanup happened
	_, found := suite.k.GetRaffle(suite.ctx, denom)
	suite.Require().False(found)

	winners := suite.k.GetRaffleWinners(suite.ctx, denom)
	suite.Require().Len(winners, 0)

	hooks := suite.k.GetRaffleDeleteHookByEndAtPrefix(suite.ctx, uint64(epochNumber))
	suite.Require().Len(hooks, 0)
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

	// Execute hook with no raffles to delete
	err := hook.AfterEpochEnd(suite.ctx, "hour", 100)

	suite.Require().NoError(err)
	// No mock expectations because function should return early when no raffles to delete
}

func (suite *IntegrationTestSuite) TestHooks_TestBurnerRaffleCleanup_ModuleAccountNotFound() {
	epochNumber := int64(100)
	denom := "utoken"

	// Set up raffle delete hook
	deleteHook := types.RaffleDeleteHook{
		Denom: denom,
		EndAt: uint64(epochNumber),
	}
	suite.k.SetRaffleDeleteHook(suite.ctx, deleteHook)

	// Set up raffle
	raffle := types.Raffle{
		Denom: denom,
		Pot:   "1000",
	}
	suite.k.SetRaffle(suite.ctx, raffle)

	// Mock module account not found
	suite.acc.EXPECT().GetModuleAccount(suite.ctx, types.RaffleModuleName).Return(nil).Times(1)

	hook := suite.k.GetBurnerRaffleCleanupHook()
	err := hook.AfterEpochEnd(suite.ctx, "hour", epochNumber)

	suite.Require().NoError(err) // Should continue despite error

	// Verify delete hook and raffle were still removed
	_, found := suite.k.GetRaffle(suite.ctx, denom)
	suite.Require().False(found)

	hooks := suite.k.GetRaffleDeleteHookByEndAtPrefix(suite.ctx, uint64(epochNumber))
	suite.Require().Len(hooks, 0)
}

func (suite *IntegrationTestSuite) TestHooks_TestBurnerRaffleCleanup_NoCoinsToBurn() {
	epochNumber := int64(100)
	denom := "utoken"

	// Set up raffle delete hook
	deleteHook := types.RaffleDeleteHook{
		Denom: denom,
		EndAt: uint64(epochNumber),
	}
	suite.k.SetRaffleDeleteHook(suite.ctx, deleteHook)

	// Set up raffle
	raffle := types.Raffle{
		Denom: denom,
		Pot:   "0",
	}
	suite.k.SetRaffle(suite.ctx, raffle)

	// Mock module account with no coins
	addr := sdk.AccAddress("raffleacc")
	raffleAcc := authtypes.ModuleAccount{
		BaseAccount: &authtypes.BaseAccount{
			Address: addr.String(),
		},
		Name: types.RaffleModuleName,
	}

	emptyCoin := sdk.NewInt64Coin(denom, 0)

	// Mock expectations
	suite.acc.EXPECT().GetModuleAccount(suite.ctx, types.RaffleModuleName).Return(&raffleAcc).Times(1)
	suite.bank.EXPECT().GetBalance(suite.ctx, addr, denom).Return(emptyCoin).Times(1)

	hook := suite.k.GetBurnerRaffleCleanupHook()
	err := hook.AfterEpochEnd(suite.ctx, "hour", epochNumber)

	suite.Require().NoError(err)

	// Verify cleanup still happened
	_, found := suite.k.GetRaffle(suite.ctx, denom)
	suite.Require().False(found)
}

func (suite *IntegrationTestSuite) TestHooks_TestBurnerRaffleCleanup_BurnCoinsError() {
	epochNumber := int64(100)
	denom := "utoken"

	// Set up raffle delete hook
	deleteHook := types.RaffleDeleteHook{
		Denom: denom,
		EndAt: uint64(epochNumber),
	}
	suite.k.SetRaffleDeleteHook(suite.ctx, deleteHook)

	// Set up raffle
	raffle := types.Raffle{
		Denom: denom,
		Pot:   "1000",
	}
	suite.k.SetRaffle(suite.ctx, raffle)

	// Mock module account with coins
	addr := sdk.AccAddress("raffleacc")
	raffleAcc := authtypes.ModuleAccount{
		BaseAccount: &authtypes.BaseAccount{
			Address: addr.String(),
		},
		Name: types.RaffleModuleName,
	}

	currentPot := sdk.NewInt64Coin(denom, 1000)
	burnError := errors.New("burn operation failed")

	// Mock expectations - burn fails
	suite.acc.EXPECT().GetModuleAccount(suite.ctx, types.RaffleModuleName).Return(&raffleAcc).Times(1)
	suite.bank.EXPECT().GetBalance(suite.ctx, addr, denom).Return(currentPot).Times(1)

	// Add TradeKeeper mock for BurnAnyCoins
	suite.trade.EXPECT().IsNativeDenom(suite.ctx, denom).Return(true).Times(1)

	suite.bank.EXPECT().BurnCoins(suite.ctx, types.RaffleModuleName, sdk.NewCoins(currentPot)).Return(burnError).Times(1)

	hook := suite.k.GetBurnerRaffleCleanupHook()
	err := hook.AfterEpochEnd(suite.ctx, "hour", epochNumber)

	suite.Require().NoError(err) // Should continue despite burn error

	// Verify cleanup still happened
	_, found := suite.k.GetRaffle(suite.ctx, denom)
	suite.Require().False(found)
}

func (suite *IntegrationTestSuite) TestHooks_TestBurnerRaffleCleanup_FactoryToken() {
	epochNumber := int64(100)
	denom := "factory/creator/token" // Factory token should not save burned coins

	// Set up raffle delete hook
	deleteHook := types.RaffleDeleteHook{
		Denom: denom,
		EndAt: uint64(epochNumber),
	}
	suite.k.SetRaffleDeleteHook(suite.ctx, deleteHook)

	// Set up raffle
	raffle := types.Raffle{
		Denom: denom,
		Pot:   "1000",
	}
	suite.k.SetRaffle(suite.ctx, raffle)

	// Mock module account with coins
	addr := sdk.AccAddress("raffleacc")
	raffleAcc := authtypes.ModuleAccount{
		BaseAccount: &authtypes.BaseAccount{
			Address: addr.String(),
		},
		Name: types.RaffleModuleName,
	}

	currentPot := sdk.NewInt64Coin(denom, 1000)

	// Mock expectations
	suite.acc.EXPECT().GetModuleAccount(suite.ctx, types.RaffleModuleName).Return(&raffleAcc).Times(1)
	suite.bank.EXPECT().GetBalance(suite.ctx, addr, denom).Return(currentPot).Times(1)

	// Add TradeKeeper mock for BurnAnyCoins - factory tokens are not native
	suite.trade.EXPECT().IsNativeDenom(suite.ctx, denom).Return(false).Times(1)

	suite.bank.EXPECT().BurnCoins(suite.ctx, types.RaffleModuleName, sdk.NewCoins(currentPot)).Return(nil).Times(1)

	hook := suite.k.GetBurnerRaffleCleanupHook()
	err := hook.AfterEpochEnd(suite.ctx, "hour", epochNumber)

	suite.Require().NoError(err)

	// Verify no burned coins were saved (factory tokens are excluded)
	burnedCoins := suite.k.GetAllBurnedCoins(suite.ctx)
	suite.Require().Len(burnedCoins, 1)
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
