package keeper_test

import (
	"github.com/bze-alphateam/bze/x/burner/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"go.uber.org/mock/gomock"
)

// Raffles saved before canonical amount validation may hold the pot or the ticket price in a
// non-canonical form, such as "0100". These tests pin how such a stored raffle is read today: the join
// and the end block parse the stored strings with math.NewIntFromString, so a ticket price of "0100"
// counts as 64 and a pot of "01000" counts as 512.

const (
	storedRaffleDenom       = "utoken"
	storedTicketPrice       = "0100"  // read as 64
	storedPot               = "01000" // read as 512
	storedTicketPriceReadAs = int64(64)
)

func (suite *IntegrationTestSuite) TestJoinRaffle_StoredTicketPriceNonCanonical() {
	creator := sdk.AccAddress("creator")
	suite.k.SetRaffle(suite.ctx, types.Raffle{
		Denom:       storedRaffleDenom,
		Pot:         storedPot,
		TicketPrice: storedTicketPrice,
		Ratio:       "0.5",
		EndAt:       200,
		TotalWon:    "0",
	})

	moduleAcc := &authtypes.ModuleAccount{BaseAccount: &authtypes.BaseAccount{Address: sdk.AccAddress("raffle_module").String()}}
	suite.bank.EXPECT().HasSupply(suite.ctx, storedRaffleDenom).Return(true).Times(1)
	suite.epoch.EXPECT().SafeGetEpochCountByIdentifier(suite.ctx, gomock.Any()).Return(int64(100), nil).Times(1)
	suite.bank.EXPECT().SpendableCoins(suite.ctx, creator).Return(sdk.NewCoins(sdk.NewInt64Coin(storedRaffleDenom, 1000))).Times(1)
	suite.acc.EXPECT().GetModuleAccount(suite.ctx, types.RaffleModuleName).Return(moduleAcc).Times(1)
	suite.bank.EXPECT().GetBalance(suite.ctx, moduleAcc.GetAddress(), storedRaffleDenom).Return(sdk.NewInt64Coin(storedRaffleDenom, 512)).Times(1)

	var captured sdk.Coins
	suite.bank.EXPECT().SendCoinsFromAccountToModule(suite.ctx, creator, types.RaffleModuleName, gomock.Any()).
		DoAndReturn(func(_ sdk.Context, _ sdk.AccAddress, _ string, coins sdk.Coins) error {
			captured = coins
			return nil
		}).Times(1)

	_, err := suite.msgServer.JoinRaffle(suite.ctx, &types.MsgJoinRaffle{Creator: creator.String(), Denom: storedRaffleDenom, Tickets: 2})
	suite.Require().NoError(err)
	suite.Require().Equal(sdk.NewCoins(sdk.NewInt64Coin(storedRaffleDenom, 2*storedTicketPriceReadAs)), captured,
		"stored ticket price is read as stored before canonical validation")
	suite.Require().Len(suite.k.GetAllRaffleParticipants(suite.ctx), 2)
}

func (suite *IntegrationTestSuite) storedNonCanonicalRaffleParticipant(chances uint64) sdk.AccAddress {
	participant := sdk.AccAddress("participant")
	suite.k.SetRaffle(suite.ctx, types.Raffle{
		Denom:       storedRaffleDenom,
		Pot:         storedPot,
		Chances:     chances,
		Ratio:       "0.5",
		TicketPrice: storedTicketPrice,
		TotalWon:    "0",
	})
	suite.k.SetRaffleParticipant(suite.ctx, types.RaffleParticipant{
		Index:       0,
		Denom:       storedRaffleDenom,
		Participant: participant.String(),
		ExecuteAt:   10,
	})

	return participant
}

func (suite *IntegrationTestSuite) TestWithdrawLucky_StoredPotNonCanonical_Winner() {
	participant := suite.storedNonCanonicalRaffleParticipant(1_000_000)

	var won sdk.Coins
	suite.bank.EXPECT().SendCoinsFromModuleToAccount(suite.ctx, types.RaffleModuleName, participant, gomock.Any()).
		DoAndReturn(func(_ sdk.Context, _ string, _ sdk.AccAddress, coins sdk.Coins) error {
			won = coins
			return nil
		}).Times(1)

	suite.k.WithdrawLuckyRaffleParticipants(suite.ctx, 10)

	// 512 * 0.5
	suite.Require().Equal(sdk.NewCoins(sdk.NewInt64Coin(storedRaffleDenom, 256)), won,
		"stored pot is read as stored before canonical validation")
	raffle, found := suite.k.GetRaffle(suite.ctx, storedRaffleDenom)
	suite.Require().True(found)
	suite.Require().Equal("256", raffle.Pot)
	suite.Require().Equal("256", raffle.TotalWon)
}

func (suite *IntegrationTestSuite) TestWithdrawLucky_StoredPotNonCanonical_Loser() {
	suite.storedNonCanonicalRaffleParticipant(0)

	suite.k.WithdrawLuckyRaffleParticipants(suite.ctx, 10)

	// 512 + 64
	raffle, found := suite.k.GetRaffle(suite.ctx, storedRaffleDenom)
	suite.Require().True(found)
	suite.Require().Equal("576", raffle.Pot, "stored pot and ticket price are read as stored before canonical validation")
	suite.Require().Equal(storedTicketPrice, raffle.TicketPrice, "the ticket price is kept as stored")
}
