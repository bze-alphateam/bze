package keeper_test

import (
	"cosmossdk.io/math"
	"fmt"
	"github.com/bze-alphateam/bze/x/rewards/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
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

	// Mock community pool funding for fee
	suite.distr.EXPECT().
		FundCommunityPool(
			suite.ctx,
			sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(200))),
			creator,
		).
		Return(nil).
		Times(1)

	// Mock epoch keeper call for expiration calculation
	suite.epoch.EXPECT().
		GetEpochCountByIdentifier(suite.ctx, "hour").
		Return(int64(100)).
		Times(1)

	msg := &types.MsgCreateTradingReward{
		Creator:     creator.String(),
		PrizeAmount: "1000",
		PrizeDenom:  "ubze",
		Duration:    "30",
		MarketId:    "market-1",
		Slots:       "5",
	}

	response, err := suite.msgServer.CreateTradingReward(suite.ctx, msg)
	suite.Require().NoError(err)
	suite.Require().NotNil(response)
	suite.Require().NotEmpty(response.RewardId)

	// Verify trading reward was created
	tradingReward, found := suite.k.GetPendingTradingReward(suite.ctx, response.RewardId)
	suite.Require().True(found)
	suite.Require().Equal("1000", tradingReward.PrizeAmount)
	suite.Require().Equal("ubze", tradingReward.PrizeDenom)
	suite.Require().Equal("market-1", tradingReward.MarketId)
	suite.Require().Equal(uint32(30), tradingReward.Duration)
	suite.Require().Equal(uint32(5), tradingReward.Slots)

	// Verify expiration was created
	expirations := suite.k.GetAllPendingTradingRewardExpirationByExpireAt(suite.ctx, tradingReward.ExpireAt)
	suite.Require().Len(expirations, 1)
	suite.Require().Equal(response.RewardId, expirations[0].RewardId)
}

func (suite *IntegrationTestSuite) TestMsgServerTradingReward_CreateTradingRewardNilRequest() {
	response, err := suite.msgServer.CreateTradingReward(suite.ctx, nil)

	suite.Require().Error(err)
	suite.Require().Nil(response)
	suite.Require().Equal(sdkerrors.ErrInvalidRequest, err)
}

func (suite *IntegrationTestSuite) TestMsgServerTradingReward_CreateTradingRewardInvalidCreator() {
	msg := &types.MsgCreateTradingReward{
		Creator:     "invalid-address",
		PrizeAmount: "1000",
		PrizeDenom:  "ubze",
		Duration:    "30",
		MarketId:    "market-1",
		Slots:       "5",
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

	msg := &types.MsgCreateTradingReward{
		Creator:     creator.String(),
		PrizeAmount: "1000",
		PrizeDenom:  "invalid-denom",
		Duration:    "30",
		MarketId:    "market-1",
		Slots:       "5",
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

	msg := &types.MsgCreateTradingReward{
		Creator:     creator.String(),
		PrizeAmount: "1000",
		PrizeDenom:  "ubze",
		Duration:    "30",
		MarketId:    "invalid-market",
		Slots:       "5",
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

	msg := &types.MsgCreateTradingReward{
		Creator:     creator.String(),
		PrizeAmount: "1000",
		PrizeDenom:  "ubze",
		Duration:    "30",
		MarketId:    "market-1",
		Slots:       "5",
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

	msg := &types.MsgCreateTradingReward{
		Creator:     creator.String(),
		PrizeAmount: "1000",
		PrizeDenom:  "ubze",
		Duration:    "30",
		MarketId:    "market-1",
		Slots:       "5",
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

	msg := &types.MsgCreateTradingReward{
		Creator:     creator.String(),
		PrizeAmount: "1000",
		PrizeDenom:  "ubze",
		Duration:    "30",
		MarketId:    "market-1",
		Slots:       "5",
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

	msg := &types.MsgCreateTradingReward{
		Creator:     creator.String(),
		PrizeAmount: "1000",
		PrizeDenom:  "ubze",
		Duration:    "30",
		MarketId:    "market-1",
		Slots:       "5",
	}

	response, err := suite.msgServer.CreateTradingReward(suite.ctx, msg)

	suite.Require().Error(err)
	suite.Require().Nil(response)
	suite.Require().Equal(bankError, err)
}

func (suite *IntegrationTestSuite) TestMsgServerTradingReward_CreateTradingRewardCommunityPoolError() {
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

	// Mock community pool funding error
	poolError := fmt.Errorf("community pool funding failed")
	suite.distr.EXPECT().
		FundCommunityPool(
			suite.ctx,
			sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(200))),
			creator,
		).
		Return(poolError).
		Times(1)

	// Mock epoch keeper call for expiration calculation
	suite.epoch.EXPECT().
		GetEpochCountByIdentifier(suite.ctx, "hour").
		Return(int64(250)).
		Times(1)

	msg := &types.MsgCreateTradingReward{
		Creator:     creator.String(),
		PrizeAmount: "1000",
		PrizeDenom:  "ubze",
		Duration:    "30",
		MarketId:    "market-1",
		Slots:       "5",
	}

	response, err := suite.msgServer.CreateTradingReward(suite.ctx, msg)

	suite.Require().Error(err)
	suite.Require().Nil(response)
	suite.Require().Equal(poolError, err)
}
