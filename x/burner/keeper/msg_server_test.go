package keeper_test

import (
	"errors"

	"github.com/bze-alphateam/bze/x/burner/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"go.uber.org/mock/gomock"
)

func (suite *IntegrationTestSuite) TestMsgServer_FundBurner_ValidRequest() {
	creator := sdk.AccAddress("creator").String()
	amount := "1000utoken,500stake"

	msg := &types.MsgFundBurner{
		Creator: creator,
		Amount:  amount,
	}

	// Parse expected coins
	expectedCoins, err := sdk.ParseCoinsNormalized(amount)
	suite.Require().NoError(err)

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
	msg := &types.MsgFundBurner{
		Creator: sdk.AccAddress("creator").String(),
		Amount:  "invalid-amount",
	}

	res, err := suite.msgServer.FundBurner(suite.ctx, msg)

	suite.Require().Error(err)
	suite.Require().Nil(res)
}

func (suite *IntegrationTestSuite) TestMsgServer_FundBurner_InvalidCreatorAddress() {
	msg := &types.MsgFundBurner{
		Creator: "invalid-address",
		Amount:  "1000utoken",
	}

	// Mock trade keeper for filtering (called before address parsing)
	suite.trade.EXPECT().IsNativeDenom(suite.ctx, "utoken").Return(true).Times(1)

	res, err := suite.msgServer.FundBurner(suite.ctx, msg)

	suite.Require().Error(err)
	suite.Require().Nil(res)
}

func (suite *IntegrationTestSuite) TestMsgServer_FundBurner_BankKeeperError() {
	creator := sdk.AccAddress("creator").String()
	amount := "1000utoken"

	msg := &types.MsgFundBurner{
		Creator: creator,
		Amount:  amount,
	}

	// Parse expected coins
	expectedCoins, err := sdk.ParseCoinsNormalized(amount)
	suite.Require().NoError(err)

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

	msg := &types.MsgFundBurner{
		Creator: creator,
		Amount:  "",
	}

	// Empty amount should now fail validation since total.IsZero() will be true
	res, err := suite.msgServer.FundBurner(suite.ctx, msg)

	suite.Require().Error(err)
	suite.Require().Nil(res)
	suite.Require().Contains(err.Error(), "provided amounts can not be burned, locked or exchanged")
}

func (suite *IntegrationTestSuite) TestMsgServer_FundBurner_MultipleDenoms() {
	creator := sdk.AccAddress("creator").String()
	amount := "1000utoken,500stake,100atom"

	msg := &types.MsgFundBurner{
		Creator: creator,
		Amount:  amount,
	}

	// Parse expected coins
	expectedCoins, err := sdk.ParseCoinsNormalized(amount)
	suite.Require().NoError(err)

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
	amount := "1000ubze,500ulp_token"

	msg := &types.MsgFundBurner{
		Creator: creator,
		Amount:  amount,
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
	amount := "1000ibc/INVALID"

	msg := &types.MsgFundBurner{
		Creator: creator,
		Amount:  amount,
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
	amount := "1000ubze,500ibc/ABC123"

	msg := &types.MsgFundBurner{
		Creator: creator,
		Amount:  amount,
	}

	// Parse expected coins
	expectedCoins, err := sdk.ParseCoinsNormalized(amount)
	suite.Require().NoError(err)

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
	amount := "1000ulp_token"

	msg := &types.MsgFundBurner{
		Creator: creator,
		Amount:  amount,
	}

	// Parse expected coins
	expectedCoins, err := sdk.ParseCoinsNormalized(amount)
	suite.Require().NoError(err)

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
