package keeper_test

import (
	"errors"

	"github.com/bze-alphateam/bze/x/burner/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (suite *IntegrationTestSuite) TestFundBurner_ValidRequest() {
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

	// Mock bank keeper expectations
	suite.bank.EXPECT().
		SendCoinsFromAccountToModule(suite.ctx, creatorAddr, types.ModuleName, expectedCoins).
		Return(nil).
		Times(1)

	res, err := suite.msgServer.FundBurner(suite.ctx, msg)

	suite.Require().NoError(err)
	suite.Require().NotNil(res)
}

func (suite *IntegrationTestSuite) TestFundBurner_InvalidAmount() {
	msg := &types.MsgFundBurner{
		Creator: sdk.AccAddress("creator").String(),
		Amount:  "invalid-amount",
	}

	res, err := suite.msgServer.FundBurner(suite.ctx, msg)

	suite.Require().Error(err)
	suite.Require().Nil(res)
}

func (suite *IntegrationTestSuite) TestFundBurner_InvalidCreatorAddress() {
	msg := &types.MsgFundBurner{
		Creator: "invalid-address",
		Amount:  "1000utoken",
	}

	res, err := suite.msgServer.FundBurner(suite.ctx, msg)

	suite.Require().Error(err)
	suite.Require().Nil(res)
}

func (suite *IntegrationTestSuite) TestFundBurner_BankKeeperError() {
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

	// Mock bank keeper to return error
	bankError := errors.New("insufficient funds")
	suite.bank.EXPECT().
		SendCoinsFromAccountToModule(suite.ctx, creatorAddr, types.ModuleName, expectedCoins).
		Return(bankError).
		Times(1)

	res, err := suite.msgServer.FundBurner(suite.ctx, msg)

	suite.Require().Error(err)
	suite.Require().Nil(res)
	suite.Require().Equal(bankError, err)
}

func (suite *IntegrationTestSuite) TestFundBurner_EmptyAmount() {
	creator := sdk.AccAddress("creator").String()

	msg := &types.MsgFundBurner{
		Creator: creator,
		Amount:  "",
	}

	// Parse creator address
	creatorAddr, err := sdk.AccAddressFromBech32(creator)
	suite.Require().NoError(err)

	// Mock bank keeper expectations
	suite.bank.EXPECT().
		SendCoinsFromAccountToModule(suite.ctx, creatorAddr, types.ModuleName, nil).
		Return(nil).
		Times(1)

	res, err := suite.msgServer.FundBurner(suite.ctx, msg)

	suite.Require().NoError(err)
	suite.Require().NotNil(res)
}

func (suite *IntegrationTestSuite) TestFundBurner_MultipleDenoms() {
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

	// Mock bank keeper expectations
	suite.bank.EXPECT().
		SendCoinsFromAccountToModule(suite.ctx, creatorAddr, types.ModuleName, expectedCoins).
		Return(nil).
		Times(1)

	res, err := suite.msgServer.FundBurner(suite.ctx, msg)

	suite.Require().NoError(err)
	suite.Require().NotNil(res)
}
