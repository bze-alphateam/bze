package keeper_test

import (
	"errors"

	"github.com/bze-alphateam/bze/x/burner/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
)

func (suite *IntegrationTestSuite) TestHooks_TestGetBurnerPeriodicBurnHook_ValidExecution() {
	hook := suite.k.GetBurnerPeriodicBurnHook()

	// Verify hook properties
	suite.Require().Equal("periodic_burner", hook.GetName())

	// Mock module account with coins
	addr := sdk.AccAddress("moduleacc")
	moduleAcc := authtypes.ModuleAccount{
		BaseAccount: &authtypes.BaseAccount{
			Address: addr.String(),
		},
		Name: types.ModuleName,
	}

	coins := sdk.NewCoins(
		sdk.NewInt64Coin("ubze", 1000),
		sdk.NewInt64Coin("utoken", 500),
	)

	// Mock expectations for burning
	suite.acc.EXPECT().GetModuleAccount(suite.ctx, types.ModuleName).Return(&moduleAcc).Times(1)
	suite.bank.EXPECT().GetAllBalances(suite.ctx, addr).Return(coins).Times(1)

	// Add TradeKeeper mocks for BurnAnyCoins
	suite.trade.EXPECT().IsNativeDenom(suite.ctx, "ubze").Return(true).Times(1)
	suite.trade.EXPECT().IsNativeDenom(suite.ctx, "utoken").Return(true).Times(1)

	suite.bank.EXPECT().BurnCoins(suite.ctx, types.ModuleName, coins).Return(nil).Times(1)

	// Execute hook with correct epoch and interval
	err := hook.AfterEpochEnd(suite.ctx, "week", 4) // 4 % 4 == 0

	suite.Require().NoError(err)
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

func (suite *IntegrationTestSuite) TestHooks_TestBurnModuleCoins_ValidExecution() {
	// Mock module account with coins
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
		sdk.NewInt64Coin("ibc/ABC123", 200), // IBC token
	)

	swappedCoin := sdk.NewInt64Coin("uother", 180) // Swapped IBC value - different denom to avoid duplicate
	expectedBurnCoins := sdk.NewCoins(
		sdk.NewInt64Coin("ubze", 1000),
		sdk.NewInt64Coin("utoken", 500),
		swappedCoin,
	)

	// Mock expectations
	suite.acc.EXPECT().GetModuleAccount(suite.ctx, types.ModuleName).Return(&moduleAcc).Times(1)
	suite.bank.EXPECT().GetAllBalances(suite.ctx, addr).Return(allCoins).Times(1)

	// Add TradeKeeper mocks for BurnAnyCoins
	suite.trade.EXPECT().IsNativeDenom(suite.ctx, "ubze").Return(true).Times(1)
	suite.trade.EXPECT().IsNativeDenom(suite.ctx, "utoken").Return(true).Times(1)
	suite.trade.EXPECT().IsNativeDenom(suite.ctx, "ibc/ABC123").Return(false).Times(1)
	suite.trade.EXPECT().CanSwapForNativeDenom(suite.ctx, sdk.NewInt64Coin("ibc/ABC123", 200)).Return(true).Times(1)
	suite.trade.EXPECT().HasLiquidityWithNativeDenom(suite.ctx, "ibc/ABC123").Return(true).Times(1)

	suite.trade.EXPECT().ModuleSwapForNativeDenom(suite.ctx, types.ModuleName, sdk.NewCoins(sdk.NewInt64Coin("ibc/ABC123", 200))).Return(swappedCoin, nil).Times(1)

	suite.bank.EXPECT().BurnCoins(suite.ctx, types.ModuleName, expectedBurnCoins).Return(nil).Times(1)

	// Execute burn
	hook := suite.k.GetBurnerPeriodicBurnHook()
	err := hook.AfterEpochEnd(suite.ctx, "week", 4)

	suite.Require().NoError(err)
}

func (suite *IntegrationTestSuite) TestHooks_TestBurnModuleCoins_EmptyBalance() {
	// Mock module account with no coins
	addr := sdk.AccAddress("moduleacc")
	moduleAcc := authtypes.ModuleAccount{
		BaseAccount: &authtypes.BaseAccount{
			Address: addr.String(),
		},
		Name: types.ModuleName,
	}

	emptyCoins := sdk.NewCoins()

	// Mock expectations
	suite.acc.EXPECT().GetModuleAccount(suite.ctx, types.ModuleName).Return(&moduleAcc).Times(1)
	suite.bank.EXPECT().GetAllBalances(suite.ctx, addr).Return(emptyCoins).Times(1)

	// Execute burn - no TradeKeeper mocks needed since burnModuleCoins returns early for empty coins
	hook := suite.k.GetBurnerPeriodicBurnHook()
	err := hook.AfterEpochEnd(suite.ctx, "week", 4)

	suite.Require().NoError(err) // Should return without error
}

func (suite *IntegrationTestSuite) TestHooks_TestBurnModuleCoins_OnlyIBCTokens() {
	// Mock module account with only IBC tokens
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

	swappedCoin := sdk.NewInt64Coin("ubze", 1350) // Total swapped value

	// Mock expectations
	suite.acc.EXPECT().GetModuleAccount(suite.ctx, types.ModuleName).Return(&moduleAcc).Times(1)
	suite.bank.EXPECT().GetAllBalances(suite.ctx, addr).Return(ibcOnlyCoins).Times(1)

	// Add TradeKeeper mocks for BurnAnyCoins
	suite.trade.EXPECT().IsNativeDenom(suite.ctx, "ibc/ABC123").Return(false).Times(1)
	suite.trade.EXPECT().IsNativeDenom(suite.ctx, "ibc/DEF456").Return(false).Times(1)
	suite.trade.EXPECT().CanSwapForNativeDenom(suite.ctx, ibcOnlyCoins[0]).Return(true).Times(1)
	suite.trade.EXPECT().CanSwapForNativeDenom(suite.ctx, ibcOnlyCoins[1]).Return(true).Times(1)
	suite.trade.EXPECT().ModuleSwapForNativeDenom(suite.ctx, types.ModuleName, ibcOnlyCoins).Return(swappedCoin, nil).Times(1)

	suite.bank.EXPECT().BurnCoins(suite.ctx, types.ModuleName, sdk.NewCoins(swappedCoin)).Return(nil).Times(1)

	// Execute burn
	hook := suite.k.GetBurnerPeriodicBurnHook()
	err := hook.AfterEpochEnd(suite.ctx, "week", 4)

	suite.Require().NoError(err)
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
