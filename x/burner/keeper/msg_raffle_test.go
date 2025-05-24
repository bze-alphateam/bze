package keeper_test

import (
	"errors"

	"github.com/bze-alphateam/bze/x/burner/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"go.uber.org/mock/gomock"
)

func (suite *IntegrationTestSuite) TestStartRaffle_ValidRequest() {
	creator := sdk.AccAddress("creator").String()
	denom := "utoken"
	pot := "1000"
	duration := "7"

	msg := &types.MsgStartRaffle{
		Creator:     creator,
		Pot:         pot,
		Duration:    duration,
		Chances:     "100",
		Ratio:       "0.1",
		TicketPrice: "10",
		Denom:       denom,
	}

	creatorAddr, err := sdk.AccAddressFromBech32(creator)
	suite.Require().NoError(err)

	potCoin := sdk.NewInt64Coin(denom, 1000)
	spendableCoins := sdk.NewCoins(sdk.NewInt64Coin(denom, 2000))

	// Mock expectations
	suite.bank.EXPECT().HasSupply(suite.ctx, denom).Return(true).Times(1)
	suite.bank.EXPECT().SpendableCoins(suite.ctx, creatorAddr).Return(spendableCoins).Times(1)
	suite.acc.EXPECT().GetModuleAccount(suite.ctx, types.RaffleModuleName).Return(&authtypes.ModuleAccount{}).Times(1)
	suite.bank.EXPECT().SendCoinsFromAccountToModule(suite.ctx, creatorAddr, types.RaffleModuleName, sdk.NewCoins(potCoin)).Return(nil).Times(1)
	suite.epoch.EXPECT().GetEpochCountByIdentifier(suite.ctx, gomock.Any()).Return(int64(100)).Times(1)

	res, err := suite.msgServer.StartRaffle(suite.ctx, msg)

	suite.Require().NoError(err)
	suite.Require().NotNil(res)

	// Verify raffle was stored
	raffle, found := suite.k.GetRaffle(suite.ctx, denom)
	suite.Require().True(found)
	suite.Require().Equal(pot, raffle.Pot)
	suite.Require().Equal(denom, raffle.Denom)
}

func (suite *IntegrationTestSuite) TestStartRaffle_DenomNotExists() {
	msg := &types.MsgStartRaffle{
		Creator: sdk.AccAddress("creator").String(),
		Denom:   "nonexistent",
	}

	suite.bank.EXPECT().HasSupply(suite.ctx, "nonexistent").Return(false).Times(1)

	res, err := suite.msgServer.StartRaffle(suite.ctx, msg)

	suite.Require().Error(err)
	suite.Require().Nil(res)
	suite.Require().Contains(err.Error(), "denom nonexistent does not exist")
}

func (suite *IntegrationTestSuite) TestStartRaffle_RaffleAlreadyExists() {
	denom := "utoken"

	// Set existing raffle
	existingRaffle := types.Raffle{Denom: denom}
	suite.k.SetRaffle(suite.ctx, existingRaffle)

	msg := &types.MsgStartRaffle{
		Creator: sdk.AccAddress("creator").String(),
		Denom:   denom,
	}

	suite.bank.EXPECT().HasSupply(suite.ctx, denom).Return(true).Times(1)

	res, err := suite.msgServer.StartRaffle(suite.ctx, msg)

	suite.Require().Error(err)
	suite.Require().Nil(res)
	suite.Require().Contains(err.Error(), "raffle already running for this coin")
}

func (suite *IntegrationTestSuite) TestStartRaffle_InsufficientBalance() {
	creator := sdk.AccAddress("creator").String()
	denom := "utoken"

	msg := &types.MsgStartRaffle{
		Creator:     creator,
		Pot:         "1000",
		Duration:    "7",
		Chances:     "100",
		Ratio:       "0.1",
		TicketPrice: "10",
		Denom:       denom,
	}

	creatorAddr, err := sdk.AccAddressFromBech32(creator)
	suite.Require().NoError(err)

	// Insufficient balance
	spendableCoins := sdk.NewCoins(sdk.NewInt64Coin(denom, 500))

	suite.bank.EXPECT().HasSupply(suite.ctx, denom).Return(true).Times(1)
	suite.bank.EXPECT().SpendableCoins(suite.ctx, creatorAddr).Return(spendableCoins).Times(1)
	suite.epoch.EXPECT().GetEpochCountByIdentifier(suite.ctx, gomock.Any()).Return(int64(100)).Times(1)

	res, err := suite.msgServer.StartRaffle(suite.ctx, msg)

	suite.Require().Error(err)
	suite.Require().Nil(res)
	suite.Require().Contains(err.Error(), "not enough balance")
}

func (suite *IntegrationTestSuite) TestStartRaffle_BankKeeperError() {
	creator := sdk.AccAddress("creator").String()
	denom := "utoken"

	msg := &types.MsgStartRaffle{
		Creator:     creator,
		Pot:         "1000",
		Duration:    "7",
		Chances:     "100",
		Ratio:       "0.1",
		TicketPrice: "10",
		Denom:       denom,
	}

	creatorAddr, err := sdk.AccAddressFromBech32(creator)
	suite.Require().NoError(err)

	potCoin := sdk.NewInt64Coin(denom, 1000)
	spendableCoins := sdk.NewCoins(sdk.NewInt64Coin(denom, 2000))
	bankError := errors.New("bank transfer failed")

	suite.bank.EXPECT().HasSupply(suite.ctx, denom).Return(true).Times(1)
	suite.bank.EXPECT().SpendableCoins(suite.ctx, creatorAddr).Return(spendableCoins).Times(1)
	suite.acc.EXPECT().GetModuleAccount(suite.ctx, types.RaffleModuleName).Return(&authtypes.ModuleAccount{}).Times(1)
	suite.bank.EXPECT().SendCoinsFromAccountToModule(suite.ctx, creatorAddr, types.RaffleModuleName, sdk.NewCoins(potCoin)).Return(bankError).Times(1)
	suite.epoch.EXPECT().GetEpochCountByIdentifier(suite.ctx, gomock.Any()).Return(int64(100)).Times(1)

	res, err := suite.msgServer.StartRaffle(suite.ctx, msg)

	suite.Require().Error(err)
	suite.Require().Nil(res)
	suite.Require().Contains(err.Error(), "could not capture pot")
}

func (suite *IntegrationTestSuite) TestJoinRaffle_ValidRequest() {
	creator := sdk.AccAddress("creator").String()
	denom := "utoken"
	tickets := uint64(2)

	// Set up existing raffle
	raffle := types.Raffle{
		Denom:       denom,
		TicketPrice: "10",
		EndAt:       200,
	}
	suite.k.SetRaffle(suite.ctx, raffle)

	msg := &types.MsgJoinRaffle{
		Creator: creator,
		Denom:   denom,
		Tickets: tickets,
	}

	creatorAddr, err := sdk.AccAddressFromBech32(creator)
	suite.Require().NoError(err)

	ticketCost := sdk.NewInt64Coin(denom, 20) // 10 * 2 tickets
	spendableCoins := sdk.NewCoins(sdk.NewInt64Coin(denom, 1000))
	addr2 := sdk.AccAddress("addr2_______________")
	moduleAcc := &authtypes.ModuleAccount{
		BaseAccount: &authtypes.BaseAccount{
			Address:       addr2.String(),
			PubKey:        nil,
			AccountNumber: 0,
			Sequence:      0,
		},
		Name:        "test",
		Permissions: nil,
	}
	moduleBalance := sdk.NewInt64Coin(denom, 5000)

	suite.bank.EXPECT().HasSupply(suite.ctx, denom).Return(true).Times(1)
	suite.epoch.EXPECT().GetEpochCountByIdentifier(suite.ctx, gomock.Any()).Return(int64(100)).Times(1)
	suite.bank.EXPECT().SpendableCoins(suite.ctx, creatorAddr).Return(spendableCoins).Times(1)
	suite.acc.EXPECT().GetModuleAccount(suite.ctx, types.RaffleModuleName).Return(moduleAcc).Times(1)
	suite.bank.EXPECT().GetBalance(suite.ctx, moduleAcc.GetAddress(), denom).Return(moduleBalance).Times(1)
	suite.bank.EXPECT().SendCoinsFromAccountToModule(suite.ctx, creatorAddr, types.RaffleModuleName, sdk.NewCoins(ticketCost)).Return(nil).Times(1)

	res, err := suite.msgServer.JoinRaffle(suite.ctx, msg)

	suite.Require().NoError(err)
	suite.Require().NotNil(res)

	// Verify participants were added
	participants := suite.k.GetAllRaffleParticipants(suite.ctx)
	suite.Require().Len(participants, 2)
}

func (suite *IntegrationTestSuite) TestJoinRaffle_DenomNotExists() {
	msg := &types.MsgJoinRaffle{
		Creator: sdk.AccAddress("creator").String(),
		Denom:   "nonexistent",
		Tickets: 1,
	}

	suite.bank.EXPECT().HasSupply(suite.ctx, "nonexistent").Return(false).Times(1)

	res, err := suite.msgServer.JoinRaffle(suite.ctx, msg)

	suite.Require().Error(err)
	suite.Require().Nil(res)
	suite.Require().Contains(err.Error(), "denom nonexistent does not exist")
}

func (suite *IntegrationTestSuite) TestJoinRaffle_RaffleNotFound() {
	denom := "utoken"
	msg := &types.MsgJoinRaffle{
		Creator: sdk.AccAddress("creator").String(),
		Denom:   denom,
		Tickets: 1,
	}

	suite.bank.EXPECT().HasSupply(suite.ctx, denom).Return(true).Times(1)

	res, err := suite.msgServer.JoinRaffle(suite.ctx, msg)

	suite.Require().Error(err)
	suite.Require().Nil(res)
	suite.Require().Contains(err.Error(), "raffle not found for provided denom")
}

func (suite *IntegrationTestSuite) TestJoinRaffle_RaffleExpired() {
	denom := "utoken"

	// Set up expired raffle
	raffle := types.Raffle{
		Denom: denom,
		EndAt: 50, // Current epoch is 100, so this is expired
	}
	suite.k.SetRaffle(suite.ctx, raffle)

	msg := &types.MsgJoinRaffle{
		Creator: sdk.AccAddress("creator").String(),
		Denom:   denom,
		Tickets: 1,
	}

	suite.bank.EXPECT().HasSupply(suite.ctx, denom).Return(true).Times(1)
	suite.epoch.EXPECT().GetEpochCountByIdentifier(suite.ctx, gomock.Any()).Return(int64(100)).Times(1)

	res, err := suite.msgServer.JoinRaffle(suite.ctx, msg)

	suite.Require().Error(err)
	suite.Require().Nil(res)
	suite.Require().Contains(err.Error(), "raffle has expired")
}

func (suite *IntegrationTestSuite) TestJoinRaffle_InsufficientBalance() {
	creator := sdk.AccAddress("creator").String()
	denom := "utoken"

	raffle := types.Raffle{
		Denom:       denom,
		TicketPrice: "100",
		EndAt:       200,
	}
	suite.k.SetRaffle(suite.ctx, raffle)

	msg := &types.MsgJoinRaffle{
		Creator: creator,
		Denom:   denom,
		Tickets: 1,
	}

	creatorAddr, err := sdk.AccAddressFromBech32(creator)
	suite.Require().NoError(err)

	// Insufficient balance
	spendableCoins := sdk.NewCoins(sdk.NewInt64Coin(denom, 50))

	suite.bank.EXPECT().HasSupply(suite.ctx, denom).Return(true).Times(1)
	suite.epoch.EXPECT().GetEpochCountByIdentifier(suite.ctx, gomock.Any()).Return(int64(100)).Times(1)
	suite.bank.EXPECT().SpendableCoins(suite.ctx, creatorAddr).Return(spendableCoins).Times(1)

	res, err := suite.msgServer.JoinRaffle(suite.ctx, msg)

	suite.Require().Error(err)
	suite.Require().Nil(res)
	suite.Require().Contains(err.Error(), "not enough balance")
}

func (suite *IntegrationTestSuite) TestJoinRaffle_NoPot() {
	creator := sdk.AccAddress("creator").String()
	denom := "utoken"

	raffle := types.Raffle{
		Denom:       denom,
		TicketPrice: "10",
		EndAt:       200,
	}
	suite.k.SetRaffle(suite.ctx, raffle)

	msg := &types.MsgJoinRaffle{
		Creator: creator,
		Denom:   denom,
		Tickets: 1,
	}

	creatorAddr, err := sdk.AccAddressFromBech32(creator)
	suite.Require().NoError(err)

	spendableCoins := sdk.NewCoins(sdk.NewInt64Coin(denom, 1000))
	addr2 := sdk.AccAddress("addr2_______________")
	moduleAcc := &authtypes.ModuleAccount{
		BaseAccount: &authtypes.BaseAccount{
			Address:       addr2.String(),
			PubKey:        nil,
			AccountNumber: 0,
			Sequence:      0,
		},
		Name:        "test",
		Permissions: nil,
	}
	// Empty balance
	moduleBalance := sdk.NewInt64Coin(denom, 0)

	suite.bank.EXPECT().HasSupply(suite.ctx, denom).Return(true).Times(1)
	suite.epoch.EXPECT().GetEpochCountByIdentifier(suite.ctx, gomock.Any()).Return(int64(100)).Times(1)
	suite.bank.EXPECT().SpendableCoins(suite.ctx, creatorAddr).Return(spendableCoins).Times(1)
	suite.acc.EXPECT().GetModuleAccount(suite.ctx, types.RaffleModuleName).Return(moduleAcc).Times(1)
	suite.bank.EXPECT().GetBalance(suite.ctx, moduleAcc.GetAddress(), denom).Return(moduleBalance).Times(1)

	res, err := suite.msgServer.JoinRaffle(suite.ctx, msg)

	suite.Require().Error(err)
	suite.Require().Nil(res)
	suite.Require().Contains(err.Error(), "no pot to participate to")
}
