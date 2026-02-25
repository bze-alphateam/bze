package keeper_test

import (
	"cosmossdk.io/math"
	"errors"

	"github.com/bze-alphateam/bze/x/burner/types"
	v2types "github.com/bze-alphateam/bze/x/burner/v2types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"go.uber.org/mock/gomock"
)

func (suite *IntegrationTestSuite) TestMsgServer_FundBurner_ValidRequest() {
	creator := sdk.AccAddress("creator").String()
	expectedCoins := sdk.NewCoins(sdk.NewCoin("stake", math.NewInt(500)), sdk.NewCoin("utoken", math.NewInt(1000)))

	msg := &v2types.MsgFundBurner{
		Creator: creator,
		Amount:  expectedCoins,
	}

	// Parse creator address
	creatorAddr, err := sdk.AccAddressFromBech32(creator)
	suite.Require().NoError(err)

	// Mock trade keeper for filtering - both are native coins
	suite.trade.EXPECT().IsNativeDenom(suite.ctx, "utoken").Return(true).Times(1)
	suite.trade.EXPECT().IsNativeDenom(suite.ctx, "stake").Return(true).Times(1)

	// Mock bank keeper - should send to burner module (not black hole since these are burnable)
	suite.bank.EXPECT().
		SendCoinsFromAccountToModule(suite.ctx, creatorAddr, types.ModuleName, expectedCoins).
		Return(nil).
		Times(1)

	res, err := suite.msgServer.FundBurner(suite.ctx, msg)

	suite.Require().NoError(err)
	suite.Require().NotNil(res)
}

func (suite *IntegrationTestSuite) TestMsgServer_FundBurner_InvalidAmount() {
	msg := &v2types.MsgFundBurner{
		Creator: sdk.AccAddress("creator").String(),
		Amount:  sdk.Coins{},
	}

	res, err := suite.msgServer.FundBurner(suite.ctx, msg)

	suite.Require().Error(err)
	suite.Require().Nil(res)
}

func (suite *IntegrationTestSuite) TestMsgServer_FundBurner_InvalidCreatorAddress() {
	msg := &v2types.MsgFundBurner{
		Creator: "invalid-address",
		Amount:  sdk.NewCoins(sdk.NewCoin("utoken", math.NewInt(1000))),
	}

	// Mock trade keeper for filtering (called before address parsing)
	suite.trade.EXPECT().IsNativeDenom(suite.ctx, "utoken").Return(true).Times(1)

	res, err := suite.msgServer.FundBurner(suite.ctx, msg)

	suite.Require().Error(err)
	suite.Require().Nil(res)
}

func (suite *IntegrationTestSuite) TestMsgServer_FundBurner_BankKeeperError() {
	creator := sdk.AccAddress("creator").String()
	expectedCoins := sdk.NewCoins(sdk.NewCoin("utoken", math.NewInt(1000)))

	msg := &v2types.MsgFundBurner{
		Creator: creator,
		Amount:  expectedCoins,
	}

	// Parse creator address
	creatorAddr, err := sdk.AccAddressFromBech32(creator)
	suite.Require().NoError(err)

	// Mock trade keeper for filtering - native coin
	suite.trade.EXPECT().IsNativeDenom(suite.ctx, "utoken").Return(true).Times(1)

	// Mock bank keeper to return error
	bankError := errors.New("insufficient funds")
	suite.bank.EXPECT().
		SendCoinsFromAccountToModule(suite.ctx, creatorAddr, types.ModuleName, expectedCoins).
		Return(bankError).
		Times(1)

	res, err := suite.msgServer.FundBurner(suite.ctx, msg)

	suite.Require().Error(err)
	suite.Require().Nil(res)
	suite.Require().Contains(err.Error(), "failed to send coins to burner module")
}

func (suite *IntegrationTestSuite) TestMsgServer_FundBurner_EmptyAmount() {
	creator := sdk.AccAddress("creator").String()

	msg := &v2types.MsgFundBurner{
		Creator: creator,
		Amount:  sdk.Coins{},
	}

	// Empty amount should fail validation since coins are not positive
	res, err := suite.msgServer.FundBurner(suite.ctx, msg)

	suite.Require().Error(err)
	suite.Require().Nil(res)
	suite.Require().Contains(err.Error(), "provided amounts are not positive")
}

func (suite *IntegrationTestSuite) TestMsgServer_FundBurner_MultipleDenoms() {
	creator := sdk.AccAddress("creator").String()
	expectedCoins := sdk.NewCoins(
		sdk.NewCoin("atom", math.NewInt(100)),
		sdk.NewCoin("stake", math.NewInt(500)),
		sdk.NewCoin("utoken", math.NewInt(1000)),
	)

	msg := &v2types.MsgFundBurner{
		Creator: creator,
		Amount:  expectedCoins,
	}

	// Parse creator address
	creatorAddr, err := sdk.AccAddressFromBech32(creator)
	suite.Require().NoError(err)

	// Mock trade keeper for filtering - all are native coins
	suite.trade.EXPECT().IsNativeDenom(suite.ctx, "atom").Return(true).Times(1)
	suite.trade.EXPECT().IsNativeDenom(suite.ctx, "stake").Return(true).Times(1)
	suite.trade.EXPECT().IsNativeDenom(suite.ctx, "utoken").Return(true).Times(1)

	// Mock bank keeper expectations - all go to burner module
	suite.bank.EXPECT().
		SendCoinsFromAccountToModule(suite.ctx, creatorAddr, types.ModuleName, expectedCoins).
		Return(nil).
		Times(1)

	res, err := suite.msgServer.FundBurner(suite.ctx, msg)

	suite.Require().NoError(err)
	suite.Require().NotNil(res)
}

func (suite *IntegrationTestSuite) TestMsgServer_FundBurner_WithLPTokens() {
	creator := sdk.AccAddress("creator").String()
	coins := sdk.NewCoins(
		sdk.NewCoin("ubze", math.NewInt(1000)),
		sdk.NewCoin("ulp_token", math.NewInt(500)),
	)

	msg := &v2types.MsgFundBurner{
		Creator: creator,
		Amount:  coins,
	}

	// Parse creator address
	creatorAddr, err := sdk.AccAddressFromBech32(creator)
	suite.Require().NoError(err)

	burnableCoin := sdk.NewCoins(sdk.NewInt64Coin("ubze", 1000))
	lockableCoin := sdk.NewCoins(sdk.NewInt64Coin("ulp_token", 500))

	// Mock trade keeper for filtering
	suite.trade.EXPECT().IsNativeDenom(suite.ctx, "ubze").Return(true).Times(1)
	suite.trade.EXPECT().IsNativeDenom(suite.ctx, "ulp_token").Return(false).Times(1)

	// Mock bank keeper - LP tokens go to BlackHole, native go to burner module
	suite.bank.EXPECT().
		SendCoinsFromAccountToModule(suite.ctx, creatorAddr, types.BlackHoleModuleName, lockableCoin).
		Return(nil).
		Times(1)
	suite.bank.EXPECT().
		SendCoinsFromAccountToModule(suite.ctx, creatorAddr, types.ModuleName, burnableCoin).
		Return(nil).
		Times(1)

	res, err := suite.msgServer.FundBurner(suite.ctx, msg)

	suite.Require().NoError(err)
	suite.Require().NotNil(res)
}

func (suite *IntegrationTestSuite) TestMsgServer_FundBurner_OnlyUnprocessableCoins() {
	creator := sdk.AccAddress("creator").String()

	msg := &v2types.MsgFundBurner{
		Creator: creator,
		Amount:  sdk.NewCoins(sdk.NewCoin("ibc/INVALID", math.NewInt(1000))),
	}

	// Mock trade keeper for filtering - coin cannot be processed
	suite.trade.EXPECT().IsNativeDenom(suite.ctx, "ibc/INVALID").Return(false).Times(1)
	suite.trade.EXPECT().CanSwapForNativeDenom(suite.ctx, gomock.Any()).Return(false).Times(1)

	// Should fail validation since no coins can be burned/locked/exchanged
	res, err := suite.msgServer.FundBurner(suite.ctx, msg)

	suite.Require().Error(err)
	suite.Require().Nil(res)
	suite.Require().Contains(err.Error(), "provided amounts can not be burned, locked or exchanged")
}

func (suite *IntegrationTestSuite) TestMsgServer_FundBurner_WithExchangeableIBC() {
	creator := sdk.AccAddress("creator").String()
	expectedCoins := sdk.NewCoins(
		sdk.NewCoin("ibc/ABC123", math.NewInt(500)),
		sdk.NewCoin("ubze", math.NewInt(1000)),
	)

	msg := &v2types.MsgFundBurner{
		Creator: creator,
		Amount:  expectedCoins,
	}

	// Parse creator address
	creatorAddr, err := sdk.AccAddressFromBech32(creator)
	suite.Require().NoError(err)

	toBurnerModule := expectedCoins // Both burnable and exchangeable go to burner module

	// Mock trade keeper for filtering
	suite.trade.EXPECT().IsNativeDenom(suite.ctx, "ibc/ABC123").Return(false).Times(1)
	suite.trade.EXPECT().IsNativeDenom(suite.ctx, "ubze").Return(true).Times(1)
	suite.trade.EXPECT().CanSwapForNativeDenom(suite.ctx, gomock.Any()).Return(true).Times(1)

	// Mock bank keeper - both coins go to burner module
	suite.bank.EXPECT().
		SendCoinsFromAccountToModule(suite.ctx, creatorAddr, types.ModuleName, toBurnerModule).
		Return(nil).
		Times(1)

	res, err := suite.msgServer.FundBurner(suite.ctx, msg)

	suite.Require().NoError(err)
	suite.Require().NotNil(res)
}

func (suite *IntegrationTestSuite) TestMsgServer_FundBurner_LockableError() {
	creator := sdk.AccAddress("creator").String()
	expectedCoins := sdk.NewCoins(sdk.NewCoin("ulp_token", math.NewInt(1000)))

	msg := &v2types.MsgFundBurner{
		Creator: creator,
		Amount:  expectedCoins,
	}

	// Parse creator address
	creatorAddr, err := sdk.AccAddressFromBech32(creator)
	suite.Require().NoError(err)

	// Mock trade keeper for filtering - LP token
	suite.trade.EXPECT().IsNativeDenom(suite.ctx, "ulp_token").Return(false).Times(1)

	// Mock bank keeper to return error when sending to BlackHole
	lockError := errors.New("failed to lock")
	suite.bank.EXPECT().
		SendCoinsFromAccountToModule(suite.ctx, creatorAddr, types.BlackHoleModuleName, expectedCoins).
		Return(lockError).
		Times(1)

	res, err := suite.msgServer.FundBurner(suite.ctx, msg)

	suite.Require().Error(err)
	suite.Require().Nil(res)
	suite.Require().Contains(err.Error(), "failed to send coins to locker")
}
