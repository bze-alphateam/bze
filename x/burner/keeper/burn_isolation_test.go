package keeper_test

import (
	"github.com/bze-alphateam/bze/x/burner/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"go.uber.org/mock/gomock"
)

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

// The periodic burn locks exchangeable IBC coins in the black hole instead of adding them to their pool: no swap check
// and no liquidity add is made, the native coin is still burned and the queue finishes.
func (suite *IntegrationTestSuite) TestProcessQueue_ExchangeableIbcCoinsAreLocked() {
	suite.k.EnqueuePeriodicBurn(suite.ctx)
	addr := sdk.AccAddress("moduleacc")
	suite.burnerModuleAcc(addr)

	native := sdk.NewInt64Coin("ubze", 1000)
	ibcA := sdk.NewInt64Coin("ibc/AAA", 200)
	ibcB := sdk.NewInt64Coin("ibc/BBB", 300)
	coins := sdk.NewCoins(native, ibcA, ibcB)
	suite.bank.EXPECT().GetBalancesPage(gomock.Any(), addr, "", types.MaxDenomsBurnPerBlock+1).Return(coins, "").Times(1)

	suite.trade.EXPECT().IsNativeDenom(gomock.Any(), "ubze").Return(true).AnyTimes()
	suite.trade.EXPECT().IsNativeDenom(gomock.Any(), ibcA.Denom).Return(false).AnyTimes()
	suite.trade.EXPECT().IsNativeDenom(gomock.Any(), ibcB.Denom).Return(false).AnyTimes()
	// no swap check and no liquidity add: a call would fail the test
	locked := suite.expectLock(types.ModuleName, sdk.NewCoins(ibcA, ibcB))
	suite.bank.EXPECT().BurnCoins(gomock.Any(), types.ModuleName, sdk.NewCoins(native)).Return(nil).Times(1)

	suite.Require().NoError(suite.k.ProcessPeriodicBurnQueue(suite.ctx))
	suite.Require().True(*locked, "the IBC coins leave the burner account through the black hole")

	_, found := suite.k.GetPeriodicBurnQueue(suite.ctx)
	suite.Require().False(found, "the queue finishes")
	burned := suite.k.GetAllBurnedCoins(suite.ctx)
	suite.Require().Len(burned, 1)
	suite.Require().Equal(sdk.NewCoins(native).String(), burned[0].Burned)
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

// A raffle pot of an exchangeable IBC coin is locked in the black hole, never added to its pool; the other raffles of
// the epoch are still cleaned up and burned, and the epoch completes.
func (suite *IntegrationTestSuite) TestProcessRaffleCleanupQueue_IbcPotIsLocked() {
	epoch := uint64(100)
	bigDenom, factory := "ibc/BIG", "factory/creator/token"
	addr := suite.raffleCleanupFixture(epoch, bigDenom, factory)

	bigPot := sdk.NewInt64Coin(bigDenom, 1_000_000)
	factoryPot := sdk.NewInt64Coin(factory, 1000)
	suite.bank.EXPECT().GetBalance(gomock.Any(), addr, bigDenom).Return(bigPot).AnyTimes()
	suite.bank.EXPECT().GetBalance(gomock.Any(), addr, factory).Return(factoryPot).AnyTimes()
	suite.trade.EXPECT().IsNativeDenom(gomock.Any(), gomock.Any()).Return(false).AnyTimes()
	// no swap check and no liquidity add: a call would fail the test
	locked := suite.expectLock(types.RaffleModuleName, sdk.NewCoins(bigPot))
	suite.bank.EXPECT().BurnCoins(gomock.Any(), types.RaffleModuleName, sdk.NewCoins(factoryPot)).Return(nil).Times(1)

	suite.Require().NoError(suite.k.ProcessRaffleCleanupQueue(suite.ctx))

	suite.Require().True(*locked, "the IBC pot is locked")
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
