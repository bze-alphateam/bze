package keeper_test

import (
	"errors"

	"github.com/bze-alphateam/bze/x/burner/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"go.uber.org/mock/gomock"
)

func (suite *IntegrationTestSuite) TestMsgRaffle_StartRaffle_ValidRequest() {
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

func (suite *IntegrationTestSuite) TestMsgRaffle_StartRaffle_DenomNotExists() {
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

func (suite *IntegrationTestSuite) TestMsgRaffle_StartRaffle_RaffleAlreadyExists() {
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

func (suite *IntegrationTestSuite) TestMsgRaffle_StartRaffle_InsufficientBalance() {
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

func (suite *IntegrationTestSuite) TestMsgRaffle_StartRaffle_BankKeeperError() {
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

func (suite *IntegrationTestSuite) TestMsgRaffle_JoinRaffle_ValidRequest() {
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

func (suite *IntegrationTestSuite) TestMsgRaffle_JoinRaffle_DenomNotExists() {
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

func (suite *IntegrationTestSuite) TestMsgRaffle_JoinRaffle_RaffleNotFound() {
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

func (suite *IntegrationTestSuite) TestMsgRaffle_JoinRaffle_RaffleExpired() {
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
	suite.Require().Contains(err.Error(), "raffle is expired or too close to expiration")
}

func (suite *IntegrationTestSuite) TestMsgRaffle_JoinRaffle_RaffleExpiredAtCurrentEpoch() {
	denom := "utoken"

	// Set up raffle where EndAt equals current epoch (final epoch - cleanup fires here)
	raffle := types.Raffle{
		Denom: denom,
		EndAt: 100, // Current epoch is 100, so this is the cleanup epoch
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
	suite.Require().Contains(err.Error(), "raffle is expired or too close to expiration")
}

func (suite *IntegrationTestSuite) TestMsgRaffle_JoinRaffle_RaffleExpiresNextEpoch() {
	denom := "utoken"

	// Set up raffle where EndAt is one epoch ahead of current (only 1 epoch remaining,
	// within the 2-epoch buffer — should be rejected)
	raffle := types.Raffle{
		Denom:       denom,
		TicketPrice: "10",
		EndAt:       101, // Current epoch is 100, one epoch remaining
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
	suite.Require().Contains(err.Error(), "raffle is expired or too close to expiration")
}

func (suite *IntegrationTestSuite) TestMsgRaffle_JoinRaffle_RaffleExpiresTwoEpochsAway() {
	denom := "utoken"

	// Set up raffle where EndAt is two epochs ahead of current (2 epochs remaining,
	// within the buffer — should be rejected)
	raffle := types.Raffle{
		Denom:       denom,
		TicketPrice: "10",
		EndAt:       102, // Current epoch is 100, two epochs remaining
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
	suite.Require().Contains(err.Error(), "raffle is expired or too close to expiration")
}

func (suite *IntegrationTestSuite) TestMsgRaffle_JoinRaffle_AllowedThreeEpochsBeforeExpiry() {
	creator := sdk.AccAddress("creator").String()
	denom := "utoken"
	tickets := uint64(1)

	// Set up raffle where EndAt is three epochs ahead of current (3 epochs remaining,
	// this is the boundary where joins are first permitted)
	raffle := types.Raffle{
		Denom:       denom,
		TicketPrice: "10",
		EndAt:       103, // Current epoch is 100, three epochs remaining
	}
	suite.k.SetRaffle(suite.ctx, raffle)

	msg := &types.MsgJoinRaffle{
		Creator: creator,
		Denom:   denom,
		Tickets: tickets,
	}

	creatorAddr, err := sdk.AccAddressFromBech32(creator)
	suite.Require().NoError(err)

	ticketCost := sdk.NewInt64Coin(denom, 10)
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

	// Verify participant was added
	participants := suite.k.GetAllRaffleParticipants(suite.ctx)
	suite.Require().Len(participants, 1)
}

func (suite *IntegrationTestSuite) TestMsgRaffle_JoinRaffle_InsufficientBalance() {
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

func (suite *IntegrationTestSuite) TestMsgRaffle_JoinRaffle_NoPot() {
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

func (suite *IntegrationTestSuite) TestMsgRaffle_JoinRaffle_TooManyParticipants() {
	denom := "utoken"

	// Set up existing raffle
	raffle := types.Raffle{
		Denom:       denom,
		TicketPrice: "10",
		EndAt:       200,
	}
	suite.k.SetRaffle(suite.ctx, raffle)

	// Pre-fill 200 participants at BlockHeight + 2 (= 0 + 2 = 2)
	execAt := suite.ctx.BlockHeight() + 2
	for i := uint64(0); i < 200; i++ {
		suite.k.SetRaffleParticipant(suite.ctx, types.RaffleParticipant{
			Index:       i,
			Denom:       denom,
			Participant: "addr1",
			ExecuteAt:   execAt,
		})
	}

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
	suite.Require().Contains(err.Error(), "too many participants")
}
