package keeper_test

import (
	"errors"
	"strconv"

	"github.com/bze-alphateam/bze/x/burner/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"go.uber.org/mock/gomock"
)

// raffleFinishedEvents counts the RaffleFinishedEvents of denom emitted on the suite context.
func (suite *IntegrationTestSuite) raffleFinishedEvents(denom string) int {
	count := 0
	for _, ev := range suite.ctx.EventManager().Events() {
		if ev.Type != "bze.burner.RaffleFinishedEvent" {
			continue
		}
		for _, attr := range ev.Attributes {
			if attr.Key == "denom" && attr.Value == strconv.Quote(denom) {
				count++
			}
		}
	}

	return count
}

// A finished raffle whose pot is burned emits its RaffleFinishedEvent and records the burn.
func (suite *IntegrationTestSuite) TestProcessRaffleCleanupQueue_BurnedPotEmitsFinished() {
	epoch := uint64(100)
	denom := "factory/creator/token"
	addr := suite.raffleCleanupFixture(epoch, denom)

	pot := sdk.NewInt64Coin(denom, 1000)
	suite.bank.EXPECT().GetBalance(gomock.Any(), addr, denom).Return(pot).AnyTimes()
	suite.trade.EXPECT().IsNativeDenom(gomock.Any(), gomock.Any()).Return(false).AnyTimes()
	suite.bank.EXPECT().BurnCoins(gomock.Any(), types.RaffleModuleName, sdk.NewCoins(pot)).Return(nil).Times(1)

	suite.Require().NoError(suite.k.ProcessRaffleCleanupQueue(suite.ctx))

	suite.requireRaffleCleanedUp(epoch, denom)
	suite.Require().Equal(1, suite.raffleFinishedEvents(denom))
	burned := suite.k.GetAllBurnedCoins(suite.ctx)
	suite.Require().Len(burned, 1)
	suite.Require().Equal(sdk.NewCoins(pot).String(), burned[0].Burned)
}

// A finished raffle whose pot can only be locked in the black hole burns nothing but still emits its
// RaffleFinishedEvent.
func (suite *IntegrationTestSuite) TestProcessRaffleCleanupQueue_LockedPotEmitsFinished() {
	epoch := uint64(100)
	denom := "ibc/AAA"
	addr := suite.raffleCleanupFixture(epoch, denom)

	pot := sdk.NewInt64Coin(denom, 1000)
	suite.bank.EXPECT().GetBalance(gomock.Any(), addr, denom).Return(pot).AnyTimes()
	suite.trade.EXPECT().IsNativeDenom(gomock.Any(), gomock.Any()).Return(false).AnyTimes()
	locked := suite.expectLock(types.RaffleModuleName, sdk.NewCoins(pot))

	suite.Require().NoError(suite.k.ProcessRaffleCleanupQueue(suite.ctx))

	suite.Require().True(*locked, "the pot is locked")
	suite.requireRaffleCleanedUp(epoch, denom)
	suite.Require().Equal(1, suite.raffleFinishedEvents(denom))
	suite.Require().Empty(suite.k.GetAllBurnedCoins(suite.ctx), "nothing burned, no burn record")
}

// A finished raffle with an empty pot still emits its RaffleFinishedEvent.
func (suite *IntegrationTestSuite) TestProcessRaffleCleanupQueue_EmptyPotEmitsFinished() {
	epoch := uint64(100)
	denom := "factory/creator/token"
	addr := suite.raffleCleanupFixture(epoch, denom)

	suite.bank.EXPECT().GetBalance(gomock.Any(), addr, denom).Return(sdk.NewInt64Coin(denom, 0)).AnyTimes()

	suite.Require().NoError(suite.k.ProcessRaffleCleanupQueue(suite.ctx))

	suite.requireRaffleCleanedUp(epoch, denom)
	suite.Require().Equal(1, suite.raffleFinishedEvents(denom))
	suite.Require().Empty(suite.k.GetAllBurnedCoins(suite.ctx))
}

// A finished raffle whose burn returns an error is still removed, so it still emits its RaffleFinishedEvent.
func (suite *IntegrationTestSuite) TestProcessRaffleCleanupQueue_BurnErrorEmitsFinished() {
	epoch := uint64(100)
	denom := "factory/creator/token"
	addr := suite.raffleCleanupFixture(epoch, denom)

	pot := sdk.NewInt64Coin(denom, 1000)
	suite.bank.EXPECT().GetBalance(gomock.Any(), addr, denom).Return(pot).AnyTimes()
	suite.trade.EXPECT().IsNativeDenom(gomock.Any(), gomock.Any()).Return(false).AnyTimes()
	suite.bank.EXPECT().BurnCoins(gomock.Any(), types.RaffleModuleName, sdk.NewCoins(pot)).Return(errors.New("burn failed")).Times(1)

	suite.Require().NoError(suite.k.ProcessRaffleCleanupQueue(suite.ctx))

	suite.requireRaffleCleanedUp(epoch, denom)
	suite.Require().Equal(1, suite.raffleFinishedEvents(denom))
	suite.Require().Empty(suite.k.GetAllBurnedCoins(suite.ctx))
}

// A raffle whose cleanup panics loses the events of its discarded cleanup; the fallback that removes it emits its
// RaffleFinishedEvent, once.
func (suite *IntegrationTestSuite) TestProcessRaffleCleanupQueue_FailedCleanupEmitsFinished() {
	epoch := uint64(100)
	denom := "factory/creator/token"
	addr := suite.raffleCleanupFixture(epoch, denom)

	pot := sdk.NewInt64Coin(denom, 1000)
	suite.bank.EXPECT().GetBalance(gomock.Any(), addr, denom).Return(pot).AnyTimes()
	suite.trade.EXPECT().IsNativeDenom(gomock.Any(), gomock.Any()).Return(false).AnyTimes()
	suite.bank.EXPECT().
		BurnCoins(gomock.Any(), types.RaffleModuleName, sdk.NewCoins(pot)).
		DoAndReturn(func(_ sdk.Context, _ string, _ sdk.Coins) error {
			panic("unexpected failure")
		}).
		Times(1)
	locked := suite.expectLock(types.RaffleModuleName, sdk.NewCoins(pot))

	suite.Require().NoError(suite.k.ProcessRaffleCleanupQueue(suite.ctx))

	suite.Require().True(*locked, "the failing raffle's pot is locked")
	suite.requireRaffleCleanedUp(epoch, denom)
	suite.Require().Equal(1, suite.raffleFinishedEvents(denom))
}
