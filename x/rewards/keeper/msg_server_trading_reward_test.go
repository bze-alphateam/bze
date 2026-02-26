package keeper_test

import (
	"cosmossdk.io/math"
	"fmt"
	"github.com/bze-alphateam/bze/x/rewards/types"
	"github.com/bze-alphateam/bze/x/rewards/v2types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"go.uber.org/mock/gomock"
)

func (suite *IntegrationTestSuite) TestMsgServerTradingReward_CreateTradingRewardSuccess() {
	creator := sdk.AccAddress("creator")

	// Set up params with fee
	params := types.Params{
		CreateStakingRewardFee: sdk.NewCoin("ubze", math.NewInt(100)),
		CreateTradingRewardFee: sdk.NewCoin("ubze", math.NewInt(200)),
	}
	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	// Mock bank keeper calls
	suite.bank.EXPECT().
		HasSupply(suite.ctx, "ubze").
		Return(true).
		Times(1)

	// Mock trade keeper call
	suite.trade.EXPECT().
		MarketExists(suite.ctx, "market-1").
		Return(true).
		Times(1)

	// Mock spendable coins check
	suite.bank.EXPECT().
		SpendableCoins(suite.ctx, creator).
		Return(sdk.NewCoins(
			sdk.NewCoin("ubze", math.NewInt(10000)),
		)).
		Times(1)

	// Mock sending coins to module
	suite.bank.EXPECT().
		SendCoinsFromAccountToModule(
			suite.ctx,
			creator,
			types.ModuleName,
			sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(5000))), // 1000 * 5 slots
		).
		Return(nil).
		Times(1)

	// Mock fee capture and swap
	suite.trade.EXPECT().
		CaptureAndSwapUserFee(
			suite.ctx,
			creator,
			sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(200))),
			types.ModuleName,
		).
		Return(sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(200))), nil).
		Times(1)

	// Mock sending fee to fee collector
	suite.bank.EXPECT().
		SendCoinsFromModuleToModule(
			suite.ctx,
			types.ModuleName,
			gomock.Any(),
			sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(200))),
		).
		Return(nil).
		Times(1)

	// Mock epoch keeper call for expiration calculation
	suite.epoch.EXPECT().
		GetEpochCountByIdentifier(suite.ctx, "hour").
		Return(int64(100)).
		Times(1)

	msg := &v2types.MsgCreateTradingReward{
		Creator:     creator.String(),
		PrizeAmount: math.NewInt(1000),
		PrizeDenom:  "ubze",
		Duration:    30,
		MarketId:    "market-1",
		Slots:       5,
	}

	counter := suite.k.GetTradingRewardsCounter(suite.ctx)
	response, err := suite.msgServer.CreateTradingReward(suite.ctx, msg)
	suite.Require().NoError(err)
	suite.Require().NotNil(response)
	suite.Require().NotEmpty(response.RewardId)
	newCounter := suite.k.GetTradingRewardsCounter(suite.ctx)
	suite.Require().Equal(counter+1, newCounter)

	// Verify trading reward was created
	tradingReward, found := suite.k.GetPendingTradingReward(suite.ctx, response.RewardId)
	suite.Require().True(found)
	suite.Require().Equal("1000", tradingReward.PrizeAmount)
	suite.Require().Equal("ubze", tradingReward.PrizeDenom)
	suite.Require().Equal("market-1", tradingReward.MarketId)
	suite.Require().Equal(uint32(30), tradingReward.Duration)
	suite.Require().Equal(uint32(5), tradingReward.Slots)

	// Verify pending expiration uses fixed 30-day timeout, not user-specified Duration
	suite.Require().Equal(uint32(100+(30*24)), tradingReward.ExpireAt) // epoch(100) + 30 days * 24 hours

	// Verify expiration was created
	expirations := suite.k.GetAllPendingTradingRewardExpirationByExpireAt(suite.ctx, tradingReward.ExpireAt)
	suite.Require().Len(expirations, 1)
	suite.Require().Equal(response.RewardId, expirations[0].RewardId)
}

func (suite *IntegrationTestSuite) TestMsgServerTradingReward_CreateTradingRewardPendingExpirationIgnoresDuration() {
	creator := sdk.AccAddress("creator")

	// Set up params with zero fee for simplicity
	params := types.Params{
		CreateStakingRewardFee: sdk.NewCoin("ubze", math.ZeroInt()),
		CreateTradingRewardFee: sdk.NewCoin("ubze", math.ZeroInt()),
	}
	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	suite.bank.EXPECT().
		HasSupply(suite.ctx, "ubze").
		Return(true).
		Times(1)

	suite.trade.EXPECT().
		MarketExists(suite.ctx, "market-1").
		Return(true).
		Times(1)

	suite.bank.EXPECT().
		SpendableCoins(suite.ctx, creator).
		Return(sdk.NewCoins(
			sdk.NewCoin("ubze", math.NewInt(100000)),
		)).
		Times(1)

	suite.bank.EXPECT().
		SendCoinsFromAccountToModule(
			suite.ctx,
			creator,
			types.ModuleName,
			sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(5000))), // 1000 * 5 slots
		).
		Return(nil).
		Times(1)

	// Mock epoch keeper - current epoch is 300
	suite.epoch.EXPECT().
		GetEpochCountByIdentifier(suite.ctx, "hour").
		Return(int64(300)).
		Times(1)

	// Create with Duration=7, but pending expiration should still be 30 days
	msg := &v2types.MsgCreateTradingReward{
		Creator:     creator.String(),
		PrizeAmount: math.NewInt(1000),
		PrizeDenom:  "ubze",
		Duration:    7,
		MarketId:    "market-1",
		Slots:       5,
	}

	response, err := suite.msgServer.CreateTradingReward(suite.ctx, msg)
	suite.Require().NoError(err)
	suite.Require().NotNil(response)

	// Verify pending expiration uses fixed 30-day timeout, NOT the 7-day user Duration
	tradingReward, found := suite.k.GetPendingTradingReward(suite.ctx, response.RewardId)
	suite.Require().True(found)
	suite.Require().Equal(uint32(7), tradingReward.Duration)
	suite.Require().Equal(uint32(300+(30*24)), tradingReward.ExpireAt) // epoch(300) + 30 days * 24 = 1020
}

func (suite *IntegrationTestSuite) TestMsgServerTradingReward_CreateTradingRewardNilRequest() {
	response, err := suite.msgServer.CreateTradingReward(suite.ctx, (*v2types.MsgCreateTradingReward)(nil))

	suite.Require().Error(err)
	suite.Require().Nil(response)
	suite.Require().Equal(sdkerrors.ErrInvalidRequest, err)
}

func (suite *IntegrationTestSuite) TestMsgServerTradingReward_CreateTradingRewardInvalidCreator() {
	msg := &v2types.MsgCreateTradingReward{
		Creator:     "invalid-address",
		PrizeAmount: math.NewInt(1000),
		PrizeDenom:  "ubze",
		Duration:    30,
		MarketId:    "market-1",
		Slots:       5,
	}

	response, err := suite.msgServer.CreateTradingReward(suite.ctx, msg)

	suite.Require().Error(err)
	suite.Require().Nil(response)
}

func (suite *IntegrationTestSuite) TestMsgServerTradingReward_CreateTradingRewardInvalidPrizeDenom() {
	creator := sdk.AccAddress("creator")

	// Mock bank keeper to return false for prize denom
	suite.bank.EXPECT().
		HasSupply(suite.ctx, "invalid-denom").
		Return(false).
		Times(1)

	msg := &v2types.MsgCreateTradingReward{
		Creator:     creator.String(),
		PrizeAmount: math.NewInt(1000),
		PrizeDenom:  "invalid-denom",
		Duration:    30,
		MarketId:    "market-1",
		Slots:       5,
	}

	response, err := suite.msgServer.CreateTradingReward(suite.ctx, msg)

	suite.Require().Error(err)
	suite.Require().Nil(response)
	suite.Require().Equal(types.ErrInvalidPrizeDenom, err)
}

func (suite *IntegrationTestSuite) TestMsgServerTradingReward_CreateTradingRewardInvalidMarketId() {
	creator := sdk.AccAddress("creator")

	// Mock bank keeper call
	suite.bank.EXPECT().
		HasSupply(suite.ctx, "ubze").
		Return(true).
		Times(1)

	// Mock trade keeper to return false for market existence
	suite.trade.EXPECT().
		MarketExists(suite.ctx, "invalid-market").
		Return(false).
		Times(1)

	msg := &v2types.MsgCreateTradingReward{
		Creator:     creator.String(),
		PrizeAmount: math.NewInt(1000),
		PrizeDenom:  "ubze",
		Duration:    30,
		MarketId:    "invalid-market",
		Slots:       5,
	}

	response, err := suite.msgServer.CreateTradingReward(suite.ctx, msg)

	suite.Require().Error(err)
	suite.Require().Nil(response)
	suite.Require().Equal(types.ErrInvalidMarketId, err)
}

func (suite *IntegrationTestSuite) TestMsgServerTradingReward_CreateTradingRewardAlreadyExists() {
	creator := sdk.AccAddress("creator")

	// Set up existing market mapping
	existingMapping := types.MarketIdTradingRewardId{
		RewardId: "existing-reward",
		MarketId: "market-1",
	}
	suite.k.SetMarketIdRewardId(suite.ctx, existingMapping)

	// Mock bank keeper call
	suite.bank.EXPECT().
		HasSupply(suite.ctx, "ubze").
		Return(true).
		Times(1)

	// Mock trade keeper call
	suite.trade.EXPECT().
		MarketExists(suite.ctx, "market-1").
		Return(true).
		Times(1)

	msg := &v2types.MsgCreateTradingReward{
		Creator:     creator.String(),
		PrizeAmount: math.NewInt(1000),
		PrizeDenom:  "ubze",
		Duration:    30,
		MarketId:    "market-1",
		Slots:       5,
	}

	response, err := suite.msgServer.CreateTradingReward(suite.ctx, msg)

	suite.Require().Error(err)
	suite.Require().Nil(response)
	suite.Require().Equal(types.ErrRewardAlreadyExists, err)
}

func (suite *IntegrationTestSuite) TestMsgServerTradingReward_CreateTradingRewardInsufficientFunds() {
	creator := sdk.AccAddress("creator")

	// Set up params with fee
	params := types.Params{
		CreateStakingRewardFee: sdk.NewCoin("ubze", math.NewInt(100)),
		CreateTradingRewardFee: sdk.NewCoin("ubze", math.NewInt(200)),
	}
	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	// Mock bank keeper calls
	suite.bank.EXPECT().
		HasSupply(suite.ctx, "ubze").
		Return(true).
		Times(1)

	// Mock trade keeper call
	suite.trade.EXPECT().
		MarketExists(suite.ctx, "market-1").
		Return(true).
		Times(1)

	// Mock insufficient spendable coins
	suite.bank.EXPECT().
		SpendableCoins(suite.ctx, creator).
		Return(sdk.NewCoins(
			sdk.NewCoin("ubze", math.NewInt(100)), // Not enough for 5000 + 200 fee
		)).
		Times(1)

	msg := &v2types.MsgCreateTradingReward{
		Creator:     creator.String(),
		PrizeAmount: math.NewInt(1000),
		PrizeDenom:  "ubze",
		Duration:    30,
		MarketId:    "market-1",
		Slots:       5,
	}

	response, err := suite.msgServer.CreateTradingReward(suite.ctx, msg)

	suite.Require().Error(err)
	suite.Require().Nil(response)
	suite.Require().Contains(err.Error(), "user balance is too low")
}

func (suite *IntegrationTestSuite) TestMsgServerTradingReward_CreateTradingRewardNoFee() {
	creator := sdk.AccAddress("creator")

	// Set up params with zero fee
	params := types.Params{
		CreateStakingRewardFee: sdk.NewCoin("ubze", math.ZeroInt()),
		CreateTradingRewardFee: sdk.NewCoin("ubze", math.ZeroInt()),
	}
	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	// Mock bank keeper calls
	suite.bank.EXPECT().
		HasSupply(suite.ctx, "ubze").
		Return(true).
		Times(1)

	// Mock trade keeper call
	suite.trade.EXPECT().
		MarketExists(suite.ctx, "market-1").
		Return(true).
		Times(1)

	// Mock spendable coins check
	suite.bank.EXPECT().
		SpendableCoins(suite.ctx, creator).
		Return(sdk.NewCoins(
			sdk.NewCoin("ubze", math.NewInt(10000)),
		)).
		Times(1)

	// Mock sending coins to module (no fee)
	suite.bank.EXPECT().
		SendCoinsFromAccountToModule(
			suite.ctx,
			creator,
			types.ModuleName,
			sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(5000))), // 1000 * 5 slots
		).
		Return(nil).
		Times(1)

	// No community pool funding expectation since fee is zero

	// Mock epoch keeper call for expiration calculation
	suite.epoch.EXPECT().
		GetEpochCountByIdentifier(suite.ctx, "hour").
		Return(int64(150)).
		Times(1)

	msg := &v2types.MsgCreateTradingReward{
		Creator:     creator.String(),
		PrizeAmount: math.NewInt(1000),
		PrizeDenom:  "ubze",
		Duration:    30,
		MarketId:    "market-1",
		Slots:       5,
	}

	response, err := suite.msgServer.CreateTradingReward(suite.ctx, msg)
	suite.Require().NoError(err)
	suite.Require().NotNil(response)
	suite.Require().NotEmpty(response.RewardId)
}

func (suite *IntegrationTestSuite) TestMsgServerTradingReward_CreateTradingRewardBankError() {
	creator := sdk.AccAddress("creator")

	// Set up params with fee
	params := types.Params{
		CreateStakingRewardFee: sdk.NewCoin("ubze", math.NewInt(100)),
		CreateTradingRewardFee: sdk.NewCoin("ubze", math.NewInt(200)),
	}
	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	// Mock bank keeper calls
	suite.bank.EXPECT().
		HasSupply(suite.ctx, "ubze").
		Return(true).
		Times(1)

	// Mock trade keeper call
	suite.trade.EXPECT().
		MarketExists(suite.ctx, "market-1").
		Return(true).
		Times(1)

	// Mock spendable coins check
	suite.bank.EXPECT().
		SpendableCoins(suite.ctx, creator).
		Return(sdk.NewCoins(
			sdk.NewCoin("ubze", math.NewInt(10000)),
		)).
		Times(1)

	// Mock bank error when sending coins
	bankError := fmt.Errorf("bank transfer failed")
	suite.bank.EXPECT().
		SendCoinsFromAccountToModule(
			suite.ctx,
			creator,
			types.ModuleName,
			sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(5000))),
		).
		Return(bankError).
		Times(1)

	// Mock epoch keeper call for expiration calculation
	suite.epoch.EXPECT().
		GetEpochCountByIdentifier(suite.ctx, "hour").
		Return(int64(200)).
		Times(1)

	msg := &v2types.MsgCreateTradingReward{
		Creator:     creator.String(),
		PrizeAmount: math.NewInt(1000),
		PrizeDenom:  "ubze",
		Duration:    30,
		MarketId:    "market-1",
		Slots:       5,
	}

	response, err := suite.msgServer.CreateTradingReward(suite.ctx, msg)

	suite.Require().Error(err)
	suite.Require().Nil(response)
	suite.Require().Equal(bankError, err)
}

func (suite *IntegrationTestSuite) TestMsgServerTradingReward_CreateTradingRewardFeeSwapError() {
	creator := sdk.AccAddress("creator")

	// Set up params with fee
	params := types.Params{
		CreateStakingRewardFee: sdk.NewCoin("ubze", math.NewInt(100)),
		CreateTradingRewardFee: sdk.NewCoin("ubze", math.NewInt(200)),
	}
	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	// Mock bank keeper calls
	suite.bank.EXPECT().
		HasSupply(suite.ctx, "ubze").
		Return(true).
		Times(1)

	// Mock trade keeper call
	suite.trade.EXPECT().
		MarketExists(suite.ctx, "market-1").
		Return(true).
		Times(1)

	// Mock spendable coins check
	suite.bank.EXPECT().
		SpendableCoins(suite.ctx, creator).
		Return(sdk.NewCoins(
			sdk.NewCoin("ubze", math.NewInt(10000)),
		)).
		Times(1)

	// Mock successful coin transfer
	suite.bank.EXPECT().
		SendCoinsFromAccountToModule(
			suite.ctx,
			creator,
			types.ModuleName,
			sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(5000))),
		).
		Return(nil).
		Times(1)

	// Mock fee swap error
	swapError := fmt.Errorf("fee swap failed")
	suite.trade.EXPECT().
		CaptureAndSwapUserFee(
			suite.ctx,
			creator,
			sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(200))),
			types.ModuleName,
		).
		Return(nil, swapError).
		Times(1)

	// Mock epoch keeper call for expiration calculation
	suite.epoch.EXPECT().
		GetEpochCountByIdentifier(suite.ctx, "hour").
		Return(int64(250)).
		Times(1)

	msg := &v2types.MsgCreateTradingReward{
		Creator:     creator.String(),
		PrizeAmount: math.NewInt(1000),
		PrizeDenom:  "ubze",
		Duration:    30,
		MarketId:    "market-1",
		Slots:       5,
	}

	response, err := suite.msgServer.CreateTradingReward(suite.ctx, msg)

	suite.Require().Error(err)
	suite.Require().Nil(response)
	suite.Require().Equal(swapError, err)
}
