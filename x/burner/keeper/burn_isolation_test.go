package keeper_test

import (
	"errors"

	"github.com/bze-alphateam/bze/x/burner/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"go.uber.org/mock/gomock"
)

// panicOnAddLiquidity makes the trade keeper panic when adding coins as liquidity, the way an arithmetic overflow on
// an oversized amount does.
func panicOnAddLiquidity(_ sdk.Context, _ string, _ sdk.Coins) (sdk.Coins, sdk.Coins, error) {
	panic("integer overflow")
}

// expectLock expects coins to be sent from fromModule to the black hole once and reports whether it happened.
func (suite *IntegrationTestSuite) expectLock(fromModule string, coins sdk.Coins) *bool {
	locked := false
	suite.bank.EXPECT().
		SendCoinsFromModuleToModule(gomock.Any(), fromModule, types.BlackHoleModuleName, coins).
		DoAndReturn(func(_ sdk.Context, _, _ string, _ sdk.Coins) error {
			locked = true
			return nil
		}).
		Times(1)

	return &locked
}

// A coin whose liquidity add panics is locked instead of staying at the front of the burner account: the other
// coins of the batch are still added, burned or locked, and the queue finishes.
func (suite *IntegrationTestSuite) TestProcessQueue_FailingExchangeableCoinIsLocked() {
	suite.k.EnqueuePeriodicBurn(suite.ctx)
	addr := sdk.AccAddress("moduleacc")
	suite.burnerModuleAcc(addr)

	native := sdk.NewInt64Coin("ubze", 1000)
	failing := sdk.NewInt64Coin("ibc/AAA", 200)
	working := sdk.NewInt64Coin("ibc/BBB", 300)
	coins := sdk.NewCoins(native, failing, working)
	suite.bank.EXPECT().GetBalancesPage(gomock.Any(), addr, "", types.MaxDenomsBurnPerBlock+1).Return(coins, "").Times(1)

	suite.trade.EXPECT().IsNativeDenom(gomock.Any(), "ubze").Return(true).AnyTimes()
	suite.trade.EXPECT().IsNativeDenom(gomock.Any(), failing.Denom).Return(false).AnyTimes()
	suite.trade.EXPECT().IsNativeDenom(gomock.Any(), working.Denom).Return(false).AnyTimes()
	suite.trade.EXPECT().CanSwapForNativeDenom(gomock.Any(), failing).Return(true).Times(1)
	suite.trade.EXPECT().CanSwapForNativeDenom(gomock.Any(), working).Return(true).Times(1)
	suite.trade.EXPECT().
		ModuleAddLiquidityWithNativeDenom(gomock.Any(), types.ModuleName, sdk.NewCoins(failing)).
		DoAndReturn(panicOnAddLiquidity).
		Times(1)
	added := false
	suite.trade.EXPECT().
		ModuleAddLiquidityWithNativeDenom(gomock.Any(), types.ModuleName, sdk.NewCoins(working)).
		DoAndReturn(func(_ sdk.Context, _ string, c sdk.Coins) (sdk.Coins, sdk.Coins, error) {
			added = true
			return c, nil, nil
		}).
		Times(1)
	locked := suite.expectLock(types.ModuleName, sdk.NewCoins(failing))
	suite.bank.EXPECT().BurnCoins(gomock.Any(), types.ModuleName, sdk.NewCoins(native)).Return(nil).Times(1)

	suite.Require().NoError(suite.k.ProcessPeriodicBurnQueue(suite.ctx))
	suite.Require().True(added, "the other exchangeable coin is still added as liquidity")
	suite.Require().True(*locked, "the failing coin leaves the burner account through the black hole")

	_, found := suite.k.GetPeriodicBurnQueue(suite.ctx)
	suite.Require().False(found, "the queue finishes")
	burned := suite.k.GetAllBurnedCoins(suite.ctx)
	suite.Require().Len(burned, 1)
	suite.Require().Equal(sdk.NewCoins(native).String(), burned[0].Burned)
}

// A coin that add-liquidity refunds (it could not be added) is locked instead of returning to the burner account,
// where it would be retried every block. Refunded LP shares and native leftovers stay with the caller.
func (suite *IntegrationTestSuite) TestBurnAnyCoins_RefundedExchangeableCoinIsLocked() {
	fromModule := "test_module"
	ibc := sdk.NewInt64Coin("ibc/AAA", 1000)

	suite.trade.EXPECT().IsNativeDenom(gomock.Any(), ibc.Denom).Return(false).Times(1)
	suite.trade.EXPECT().CanSwapForNativeDenom(gomock.Any(), ibc).Return(true).Times(1)
	refunded := sdk.NewCoins(ibc, sdk.NewInt64Coin("ubze", 5), sdk.NewInt64Coin("ulp_1", 3))
	suite.trade.EXPECT().
		ModuleAddLiquidityWithNativeDenom(gomock.Any(), fromModule, sdk.NewCoins(ibc)).
		Return(nil, refunded, nil).
		Times(1)
	locked := suite.expectLock(fromModule, sdk.NewCoins(ibc))

	burned, err := suite.k.BurnAnyCoins(suite.ctx, fromModule, sdk.NewCoins(ibc))
	suite.Require().NoError(err)
	suite.Require().True(burned.IsZero())
	suite.Require().True(*locked)
}

// When add-liquidity returns an error nothing moved, so the coin is locked.
func (suite *IntegrationTestSuite) TestBurnAnyCoins_AddLiquidityErrorLocksCoin() {
	fromModule := "test_module"
	ibc := sdk.NewInt64Coin("ibc/AAA", 1000)

	suite.trade.EXPECT().IsNativeDenom(gomock.Any(), ibc.Denom).Return(false).Times(1)
	suite.trade.EXPECT().CanSwapForNativeDenom(gomock.Any(), ibc).Return(true).Times(1)
	suite.trade.EXPECT().
		ModuleAddLiquidityWithNativeDenom(gomock.Any(), fromModule, sdk.NewCoins(ibc)).
		Return(nil, nil, errors.New("add liquidity failed")).
		Times(1)
	locked := suite.expectLock(fromModule, sdk.NewCoins(ibc))

	_, err := suite.k.BurnAnyCoins(suite.ctx, fromModule, sdk.NewCoins(ibc))
	suite.Require().NoError(err)
	suite.Require().True(*locked)
}

// raffleCleanupFixture stores one raffle per denom ending at epoch, enqueues the epoch and returns the raffle
// module account address.
func (suite *IntegrationTestSuite) raffleCleanupFixture(epoch uint64, denoms ...string) sdk.AccAddress {
	for _, d := range denoms {
		suite.k.SetRaffleDeleteHook(suite.ctx, types.RaffleDeleteHook{Denom: d, EndAt: epoch})
		suite.k.SetRaffle(suite.ctx, types.Raffle{Denom: d, Pot: "1"})
		suite.k.SetRaffleWinner(suite.ctx, types.RaffleWinner{Index: "1", Denom: d, Amount: "1", Winner: "w"})
	}
	suite.k.EnqueueRaffleCleanup(suite.ctx, epoch)

	addr := sdk.AccAddress("raffleacc")
	raffleAcc := authtypes.ModuleAccount{
		BaseAccount: &authtypes.BaseAccount{Address: addr.String()},
		Name:        types.RaffleModuleName,
	}
	suite.acc.EXPECT().GetModuleAccount(gomock.Any(), types.RaffleModuleName).Return(&raffleAcc).AnyTimes()

	return addr
}

func (suite *IntegrationTestSuite) requireRaffleCleanedUp(epoch uint64, denoms ...string) {
	for _, d := range denoms {
		_, found := suite.k.GetRaffle(suite.ctx, d)
		suite.Require().False(found, "raffle %s removed", d)
		suite.Require().Empty(suite.k.GetRaffleWinners(suite.ctx, d), "winners of %s removed", d)
	}
	suite.Require().Empty(suite.k.GetRaffleDeleteHookByEndAtPrefix(suite.ctx, epoch))
	_, found := suite.k.GetRaffleCleanupQueue(suite.ctx)
	suite.Require().False(found, "the epoch completes")
}

// An oversized raffle pot whose liquidity add panics is locked; the other raffles of the epoch are still cleaned up
// and burned, and the epoch completes.
func (suite *IntegrationTestSuite) TestProcessRaffleCleanupQueue_OversizedPotDoesNotBlockEpoch() {
	epoch := uint64(100)
	bigDenom, factory := "ibc/BIG", "factory/creator/token"
	addr := suite.raffleCleanupFixture(epoch, bigDenom, factory)

	bigPot := sdk.NewInt64Coin(bigDenom, 1_000_000)
	factoryPot := sdk.NewInt64Coin(factory, 1000)
	suite.bank.EXPECT().GetBalance(gomock.Any(), addr, bigDenom).Return(bigPot).AnyTimes()
	suite.bank.EXPECT().GetBalance(gomock.Any(), addr, factory).Return(factoryPot).AnyTimes()
	suite.trade.EXPECT().IsNativeDenom(gomock.Any(), gomock.Any()).Return(false).AnyTimes()
	suite.trade.EXPECT().CanSwapForNativeDenom(gomock.Any(), bigPot).Return(true).AnyTimes()
	suite.trade.EXPECT().
		ModuleAddLiquidityWithNativeDenom(gomock.Any(), types.RaffleModuleName, sdk.NewCoins(bigPot)).
		DoAndReturn(panicOnAddLiquidity).
		AnyTimes()
	locked := suite.expectLock(types.RaffleModuleName, sdk.NewCoins(bigPot))
	suite.bank.EXPECT().BurnCoins(gomock.Any(), types.RaffleModuleName, sdk.NewCoins(factoryPot)).Return(nil).Times(1)

	suite.Require().NoError(suite.k.ProcessRaffleCleanupQueue(suite.ctx))

	suite.Require().True(*locked, "the oversized pot is locked")
	suite.requireRaffleCleanedUp(epoch, bigDenom, factory)
	burned := suite.k.GetAllBurnedCoins(suite.ctx)
	suite.Require().Len(burned, 1)
	suite.Require().Equal(sdk.NewCoins(factoryPot).String(), burned[0].Burned)
}

// A raffle whose cleanup panics outside add-liquidity is still removed and its pot locked; the other raffles of the
// epoch are processed and the epoch completes.
func (suite *IntegrationTestSuite) TestProcessRaffleCleanupQueue_FailingItemIsolated() {
	epoch := uint64(100)
	failing, working := "factory/creator/aaa", "factory/creator/bbb"
	addr := suite.raffleCleanupFixture(epoch, failing, working)

	failingPot := sdk.NewInt64Coin(failing, 500)
	workingPot := sdk.NewInt64Coin(working, 1000)
	suite.bank.EXPECT().GetBalance(gomock.Any(), addr, failing).Return(failingPot).AnyTimes()
	suite.bank.EXPECT().GetBalance(gomock.Any(), addr, working).Return(workingPot).AnyTimes()
	suite.trade.EXPECT().IsNativeDenom(gomock.Any(), gomock.Any()).Return(false).AnyTimes()
	suite.bank.EXPECT().
		BurnCoins(gomock.Any(), types.RaffleModuleName, sdk.NewCoins(failingPot)).
		DoAndReturn(func(_ sdk.Context, _ string, _ sdk.Coins) error {
			panic("unexpected failure")
		}).
		Times(1)
	suite.bank.EXPECT().BurnCoins(gomock.Any(), types.RaffleModuleName, sdk.NewCoins(workingPot)).Return(nil).Times(1)
	locked := suite.expectLock(types.RaffleModuleName, sdk.NewCoins(failingPot))

	suite.Require().NoError(suite.k.ProcessRaffleCleanupQueue(suite.ctx))

	suite.Require().True(*locked, "the failing raffle's pot is locked")
	suite.requireRaffleCleanedUp(epoch, failing, working)
	burned := suite.k.GetAllBurnedCoins(suite.ctx)
	suite.Require().Len(burned, 1)
	suite.Require().Equal(sdk.NewCoins(workingPot).String(), burned[0].Burned)
}
