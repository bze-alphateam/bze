package keeper_test

import (
	"math/big"

	"cosmossdk.io/math"
	"github.com/bze-alphateam/bze/x/burner/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"go.uber.org/mock/gomock"
)

// maxUint256 is the largest value a math.Int can hold.
func maxUint256() math.Int {
	return math.NewIntFromBigInt(new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 256), big.NewInt(1)))
}

// winningRaffle stores a raffle every ticket wins (10% of the pot) with the given total won.
func (suite *IntegrationTestSuite) winningRaffle(denom, totalWon string) {
	suite.k.SetRaffle(suite.ctx, types.Raffle{
		Denom:       denom,
		Pot:         "100000",
		Chances:     1_000_000,
		Ratio:       "0.1",
		TicketPrice: "10",
		TotalWon:    totalWon,
	})
}

func (suite *IntegrationTestSuite) ticket(index uint64, denom string, participant sdk.AccAddress) {
	suite.k.SetRaffleParticipant(suite.ctx, types.RaffleParticipant{
		Index:       index,
		Denom:       denom,
		Participant: participant.String(),
		ExecuteAt:   10,
	})
}

// A total won counter already at the largest Int doesn't stop the draw: the winner is paid, the pot shrinks, the
// counter stays where it is, and the other tickets at that height still resolve.
func (suite *IntegrationTestSuite) TestWithdrawLucky_TotalWonAtMaxInt() {
	full, other := "ufull", "uother"
	suite.winningRaffle(full, maxUint256().String())
	suite.winningRaffle(other, "0")

	alice, bob := sdk.AccAddress("alice"), sdk.AccAddress("bob")
	suite.ticket(0, full, alice)
	suite.ticket(1, other, bob)

	suite.bank.EXPECT().
		SendCoinsFromModuleToAccount(gomock.Any(), types.RaffleModuleName, alice, sdk.NewCoins(sdk.NewInt64Coin(full, 10000))).
		Return(nil).Times(1)
	suite.bank.EXPECT().
		SendCoinsFromModuleToAccount(gomock.Any(), types.RaffleModuleName, bob, sdk.NewCoins(sdk.NewInt64Coin(other, 10000))).
		Return(nil).Times(1)

	suite.Require().NotPanics(func() {
		suite.k.WithdrawLuckyRaffleParticipants(suite.ctx, 10)
	})

	raffle, found := suite.k.GetRaffle(suite.ctx, full)
	suite.Require().True(found)
	suite.Require().Equal(uint64(1), raffle.Winners)
	suite.Require().Equal("90000", raffle.Pot)
	suite.Require().Equal(maxUint256().String(), raffle.TotalWon, "the counter is left unchanged")
	suite.Require().Len(suite.k.GetRaffleWinners(suite.ctx, full), 1)

	raffle, found = suite.k.GetRaffle(suite.ctx, other)
	suite.Require().True(found)
	suite.Require().Equal(uint64(1), raffle.Winners)
	suite.Require().Equal("10000", raffle.TotalWon)
	suite.Require().Equal("90000", raffle.Pot)

	suite.Require().Empty(suite.k.GetAllRaffleParticipants(suite.ctx))
}

// A ticket whose resolution panics only voids itself: the other tickets at that height still resolve and are
// persisted, and the failing ticket is removed.
func (suite *IntegrationTestSuite) TestWithdrawLucky_FailingTicketIsolated() {
	failing, other := "ufail", "uother"
	suite.winningRaffle(failing, "0")
	suite.winningRaffle(other, "0")

	alice, bob := sdk.AccAddress("alice"), sdk.AccAddress("bob")
	suite.ticket(0, failing, alice)
	suite.ticket(1, other, bob)

	suite.bank.EXPECT().
		SendCoinsFromModuleToAccount(gomock.Any(), types.RaffleModuleName, alice, gomock.Any()).
		DoAndReturn(func(_ sdk.Context, _ string, _ sdk.AccAddress, _ sdk.Coins) error {
			panic("unexpected failure")
		}).Times(1)
	suite.bank.EXPECT().
		SendCoinsFromModuleToAccount(gomock.Any(), types.RaffleModuleName, bob, sdk.NewCoins(sdk.NewInt64Coin(other, 10000))).
		Return(nil).Times(1)

	suite.Require().NotPanics(func() {
		suite.k.WithdrawLuckyRaffleParticipants(suite.ctx, 10)
	})

	raffle, found := suite.k.GetRaffle(suite.ctx, failing)
	suite.Require().True(found)
	suite.Require().Equal(uint64(0), raffle.Winners, "the failing draw is not persisted")
	suite.Require().Equal("100000", raffle.Pot)

	raffle, found = suite.k.GetRaffle(suite.ctx, other)
	suite.Require().True(found)
	suite.Require().Equal(uint64(1), raffle.Winners)
	suite.Require().Equal("10000", raffle.TotalWon)

	suite.Require().Empty(suite.k.GetAllRaffleParticipants(suite.ctx), "every ticket of the height is removed")
}
