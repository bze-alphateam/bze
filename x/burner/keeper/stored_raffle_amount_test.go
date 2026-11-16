package keeper_test

import (
	"github.com/bze-alphateam/bze/x/burner/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"go.uber.org/mock/gomock"
)

// Covers raffles stored before input validation.

const (
	storedRaffleDenom   = "utoken"
	storedTicketPrice   = "0100"
	storedPot           = "01000"
	expectedTicketPrice = int64(64)
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
	suite.bank.EXPECT().HasSupply(gomock.Any(), storedRaffleDenom).Return(true).Times(1)
	suite.epoch.EXPECT().SafeGetEpochCountByIdentifier(gomock.Any(), gomock.Any()).Return(int64(100), nil).Times(1)
	suite.bank.EXPECT().SpendableCoins(gomock.Any(), creator).Return(sdk.NewCoins(sdk.NewInt64Coin(storedRaffleDenom, 1000))).Times(1)
	suite.acc.EXPECT().GetModuleAccount(gomock.Any(), types.RaffleModuleName).Return(moduleAcc).Times(1)
	suite.bank.EXPECT().GetBalance(gomock.Any(), moduleAcc.GetAddress(), storedRaffleDenom).Return(sdk.NewInt64Coin(storedRaffleDenom, 512)).Times(1)

	var captured sdk.Coins
	suite.bank.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), creator, types.RaffleModuleName, gomock.Any()).
		DoAndReturn(func(_ sdk.Context, _ sdk.AccAddress, _ string, coins sdk.Coins) error {
			captured = coins
			return nil
		}).Times(1)

	_, err := suite.msgServer.JoinRaffle(suite.ctx, &types.MsgJoinRaffle{Creator: creator.String(), Denom: storedRaffleDenom, Tickets: 2})
	suite.Require().NoError(err)
	suite.Require().Equal(sdk.NewCoins(sdk.NewInt64Coin(storedRaffleDenom, 2*expectedTicketPrice)), captured,
		"stored ticket price")
	suite.Require().Len(suite.k.GetAllRaffleParticipants(suite.ctx), 2)
}

func (suite *IntegrationTestSuite) storedRaffleParticipant(chances uint64) sdk.AccAddress {
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
	participant := suite.storedRaffleParticipant(1_000_000)

	var won sdk.Coins
	suite.bank.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.RaffleModuleName, participant, gomock.Any()).
		DoAndReturn(func(_ sdk.Context, _ string, _ sdk.AccAddress, coins sdk.Coins) error {
			won = coins
			return nil
		}).Times(1)

	suite.k.WithdrawLuckyRaffleParticipants(suite.ctx, 10)

	suite.Require().Equal(sdk.NewCoins(sdk.NewInt64Coin(storedRaffleDenom, 256)), won,
		"stored pot")
	raffle, found := suite.k.GetRaffle(suite.ctx, storedRaffleDenom)
	suite.Require().True(found)
	suite.Require().Equal("256", raffle.Pot)
	suite.Require().Equal("256", raffle.TotalWon)
}

func (suite *IntegrationTestSuite) TestWithdrawLucky_StoredPotNonCanonical_Loser() {
	suite.storedRaffleParticipant(0)

	suite.k.WithdrawLuckyRaffleParticipants(suite.ctx, 10)

	raffle, found := suite.k.GetRaffle(suite.ctx, storedRaffleDenom)
	suite.Require().True(found)
	suite.Require().Equal("576", raffle.Pot, "stored pot and ticket price")
	suite.Require().Equal(storedTicketPrice, raffle.TicketPrice, "the ticket price is kept as stored")
}
