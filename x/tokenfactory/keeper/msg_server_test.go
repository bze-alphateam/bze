package keeper_test

import (
	"errors"

	"cosmossdk.io/math"
	"github.com/bze-alphateam/bze/x/tokenfactory/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"go.uber.org/mock/gomock"
)

func (suite *IntegrationTestSuite) TestCreateDenom_ValidRequest() {
	creator := sdk.AccAddress("creator").String()
	subdenom := "testtoken"
	expectedDenom := "factory/" + creator + "/" + subdenom

	// Set up params with create denom fee
	params := types.Params{
		CreateDenomFee: sdk.NewInt64Coin("ubze", 1000),
	}
	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	creatorAddr, err := sdk.AccAddressFromBech32(creator)
	suite.Require().NoError(err)

	// Mock expectations for validation
	suite.bank.EXPECT().HasSupply(suite.ctx, subdenom).Return(false).Times(1)
	suite.bank.EXPECT().GetDenomMetaData(suite.ctx, expectedDenom).Return(banktypes.Metadata{}, false).Times(1)

	// Mock expectations for charging fee
	expectedFee := sdk.NewCoins(sdk.NewInt64Coin("ubze", 1000))
	suite.trade.EXPECT().CaptureAndSwapUserFee(suite.ctx, creatorAddr, expectedFee, types.ModuleName).Return(expectedFee, nil).Times(1)
	suite.bank.EXPECT().SendCoinsFromModuleToModule(suite.ctx, types.ModuleName, gomock.Any(), expectedFee).Return(nil).Times(1)

	// Mock expectations for CreateDenomAfterValidation
	suite.bank.EXPECT().GetDenomMetaData(suite.ctx, expectedDenom).Return(banktypes.Metadata{}, false).Times(1)
	expectedMetadata := banktypes.Metadata{
		DenomUnits: []*banktypes.DenomUnit{{
			Denom:    expectedDenom,
			Exponent: 0,
		}},
		Base: expectedDenom,
	}
	suite.bank.EXPECT().SetDenomMetaData(suite.ctx, expectedMetadata).Times(1)

	msg := &types.MsgCreateDenom{
		Creator:  creator,
		Subdenom: subdenom,
	}

	res, err := suite.msgServer.CreateDenom(suite.ctx, msg)

	suite.Require().NoError(err)
	suite.Require().NotNil(res)
	suite.Require().Equal(expectedDenom, res.NewDenom)

	// Verify denom authority was set
	authority, err := suite.k.GetDenomAuthority(suite.ctx, expectedDenom)
	suite.Require().NoError(err)
	suite.Require().Equal(creator, authority.Admin)
}

func (suite *IntegrationTestSuite) TestCreateDenom_ValidationError() {
	creator := sdk.AccAddress("creator").String()
	subdenom := "invalid_subdenom" // Contains underscore

	// Mock validation error
	suite.bank.EXPECT().HasSupply(suite.ctx, subdenom).Return(false).Times(1)

	msg := &types.MsgCreateDenom{
		Creator:  creator,
		Subdenom: subdenom,
	}

	res, err := suite.msgServer.CreateDenom(suite.ctx, msg)

	suite.Require().Error(err)
	suite.Require().Nil(res)
	suite.Require().Equal(types.ErrInvalidSubdenom, err)
}

func (suite *IntegrationTestSuite) TestCreateDenom_ChargingError() {
	creator := sdk.AccAddress("creator").String()
	subdenom := "testtoken"
	expectedDenom := "factory/" + creator + "/" + subdenom

	// Set up params with create denom fee
	params := types.Params{
		CreateDenomFee: sdk.NewInt64Coin("ubze", 1000),
	}
	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	creatorAddr, err := sdk.AccAddressFromBech32(creator)
	suite.Require().NoError(err)

	// Mock validation success
	suite.bank.EXPECT().HasSupply(suite.ctx, subdenom).Return(false).Times(1)
	suite.bank.EXPECT().GetDenomMetaData(suite.ctx, expectedDenom).Return(banktypes.Metadata{}, false).Times(1)

	// Mock charging error
	expectedFee := sdk.NewCoins(sdk.NewInt64Coin("ubze", 1000))
	chargingError := errors.New("insufficient funds")
	suite.trade.EXPECT().CaptureAndSwapUserFee(suite.ctx, creatorAddr, expectedFee, types.ModuleName).Return(nil, chargingError).Times(1)

	msg := &types.MsgCreateDenom{
		Creator:  creator,
		Subdenom: subdenom,
	}

	res, err := suite.msgServer.CreateDenom(suite.ctx, msg)

	suite.Require().Error(err)
	suite.Require().Nil(res)
	suite.Require().Equal(chargingError, err)
}

func (suite *IntegrationTestSuite) TestMint_ValidRequest() {
	creator := sdk.AccAddress("creator").String()
	denom := "factory/" + creator + "/testtoken"

	// Set up denom authority
	denomAuth := types.DenomAuthority{
		Admin: creator,
	}
	err := suite.k.SetDenomAuthority(suite.ctx, denom, denomAuth)
	suite.Require().NoError(err)

	creatorAddr, err := sdk.AccAddressFromBech32(creator)
	suite.Require().NoError(err)

	expectedCoin := sdk.NewInt64Coin(denom, 1000)
	expectedCoins := sdk.NewCoins(expectedCoin)

	// Mock expectations
	suite.bank.EXPECT().GetDenomMetaData(suite.ctx, denom).Return(banktypes.Metadata{Base: denom}, true).Times(1)
	suite.bank.EXPECT().MintCoins(suite.ctx, types.ModuleName, expectedCoins).Return(nil).Times(1)
	suite.bank.EXPECT().SendCoinsFromModuleToAccount(suite.ctx, types.ModuleName, creatorAddr, expectedCoins).Return(nil).Times(1)

	msg := &types.MsgMint{
		Creator: creator,
		Coins:   expectedCoin,
	}

	res, err := suite.msgServer.Mint(suite.ctx, msg)

	suite.Require().NoError(err)
	suite.Require().NotNil(res)
}

func (suite *IntegrationTestSuite) TestMint_InvalidAmount() {
	creator := sdk.AccAddress("creator").String()

	msg := &types.MsgMint{
		Creator: creator,
		Coins:   sdk.Coin{Denom: "utoken", Amount: math.ZeroInt()},
	}

	res, err := suite.msgServer.Mint(suite.ctx, msg)

	suite.Require().Error(err)
	suite.Require().Nil(res)
	suite.Require().Contains(err.Error(), "invalid amount")
}

func (suite *IntegrationTestSuite) TestMint_DenomDoesNotExist() {
	creator := sdk.AccAddress("creator").String()
	denom := "factory/" + creator + "/nonexistent"

	// Mock denom doesn't exist
	suite.bank.EXPECT().GetDenomMetaData(suite.ctx, denom).Return(banktypes.Metadata{}, false).Times(1)

	msg := &types.MsgMint{
		Creator: creator,
		Coins:   sdk.NewInt64Coin(denom, 1000),
	}

	res, err := suite.msgServer.Mint(suite.ctx, msg)

	suite.Require().Error(err)
	suite.Require().Nil(res)
	suite.Require().Contains(err.Error(), "denom does not exist")
}

func (suite *IntegrationTestSuite) TestMint_Unauthorized() {
	creator := sdk.AccAddress("creator").String()
	unauthorized := sdk.AccAddress("unauthorized").String()
	denom := "factory/" + creator + "/testtoken"

	// Set up denom authority with different admin
	denomAuth := types.DenomAuthority{
		Admin: creator,
	}
	err := suite.k.SetDenomAuthority(suite.ctx, denom, denomAuth)
	suite.Require().NoError(err)

	// Mock denom exists
	suite.bank.EXPECT().GetDenomMetaData(suite.ctx, denom).Return(banktypes.Metadata{Base: denom}, true).Times(1)

	msg := &types.MsgMint{
		Creator: unauthorized,
		Coins:   sdk.NewInt64Coin(denom, 1000),
	}

	res, err := suite.msgServer.Mint(suite.ctx, msg)

	suite.Require().Error(err)
	suite.Require().Nil(res)
	suite.Require().Equal(types.ErrUnauthorized, err)
}

func (suite *IntegrationTestSuite) TestBurn_ValidRequest() {
	creator := sdk.AccAddress("creator").String()
	denom := "factory/" + creator + "/testtoken"

	// Set up denom authority
	denomAuth := types.DenomAuthority{
		Admin: creator,
	}
	err := suite.k.SetDenomAuthority(suite.ctx, denom, denomAuth)
	suite.Require().NoError(err)

	creatorAddr, err := sdk.AccAddressFromBech32(creator)
	suite.Require().NoError(err)

	expectedCoin := sdk.NewInt64Coin(denom, 1000)
	expectedCoins := sdk.NewCoins(expectedCoin)

	// Mock expectations
	suite.bank.EXPECT().SendCoinsFromAccountToModule(suite.ctx, creatorAddr, types.ModuleName, expectedCoins).Return(nil).Times(1)
	suite.bank.EXPECT().BurnCoins(suite.ctx, types.ModuleName, expectedCoins).Return(nil).Times(1)

	msg := &types.MsgBurn{
		Creator: creator,
		Coins:   expectedCoin,
	}

	res, err := suite.msgServer.Burn(suite.ctx, msg)

	suite.Require().NoError(err)
	suite.Require().NotNil(res)
}

func (suite *IntegrationTestSuite) TestBurn_InvalidAmount() {
	creator := sdk.AccAddress("creator").String()

	msg := &types.MsgBurn{
		Creator: creator,
		Coins:   sdk.Coin{Denom: "utoken", Amount: math.ZeroInt()},
	}

	res, err := suite.msgServer.Burn(suite.ctx, msg)

	suite.Require().Error(err)
	suite.Require().Nil(res)
	suite.Require().Contains(err.Error(), "invalid amount")
}

func (suite *IntegrationTestSuite) TestBurn_Unauthorized() {
	creator := sdk.AccAddress("creator").String()
	unauthorized := sdk.AccAddress("unauthorized").String()
	denom := "factory/" + creator + "/testtoken"

	// Set up denom authority with different admin
	denomAuth := types.DenomAuthority{
		Admin: creator,
	}
	err := suite.k.SetDenomAuthority(suite.ctx, denom, denomAuth)
	suite.Require().NoError(err)

	msg := &types.MsgBurn{
		Creator: unauthorized,
		Coins:   sdk.NewInt64Coin(denom, 1000),
	}

	res, err := suite.msgServer.Burn(suite.ctx, msg)

	suite.Require().Error(err)
	suite.Require().Nil(res)
	suite.Require().Equal(types.ErrUnauthorized, err)
}

func (suite *IntegrationTestSuite) TestChangeAdmin_ValidRequest() {
	creator := sdk.AccAddress("creator").String()
	newAdmin := sdk.AccAddress("newadmin").String()
	denom := "factory/" + creator + "/testtoken"

	// Set up denom authority
	denomAuth := types.DenomAuthority{
		Admin: creator,
	}
	err := suite.k.SetDenomAuthority(suite.ctx, denom, denomAuth)
	suite.Require().NoError(err)

	msg := &types.MsgChangeAdmin{
		Creator:  creator,
		Denom:    denom,
		NewAdmin: newAdmin,
	}

	res, err := suite.msgServer.ChangeAdmin(suite.ctx, msg)

	suite.Require().NoError(err)
	suite.Require().NotNil(res)

	// Verify admin was changed
	updatedAuth, err := suite.k.GetDenomAuthority(suite.ctx, denom)
	suite.Require().NoError(err)
	suite.Require().Equal(newAdmin, updatedAuth.Admin)
}

func (suite *IntegrationTestSuite) TestChangeAdmin_EmptyNewAdmin() {
	creator := sdk.AccAddress("creator").String()
	denom := "factory/" + creator + "/testtoken"

	// Set up denom authority
	denomAuth := types.DenomAuthority{
		Admin: creator,
	}
	err := suite.k.SetDenomAuthority(suite.ctx, denom, denomAuth)
	suite.Require().NoError(err)

	msg := &types.MsgChangeAdmin{
		Creator:  creator,
		Denom:    denom,
		NewAdmin: "", // Empty new admin
	}

	res, err := suite.msgServer.ChangeAdmin(suite.ctx, msg)

	suite.Require().NoError(err)
	suite.Require().NotNil(res)

	// Verify admin was changed to empty
	updatedAuth, err := suite.k.GetDenomAuthority(suite.ctx, denom)
	suite.Require().NoError(err)
	suite.Require().Equal("", updatedAuth.Admin)
}

func (suite *IntegrationTestSuite) TestChangeAdmin_InvalidNewAdmin() {
	creator := sdk.AccAddress("creator").String()
	denom := "factory/" + creator + "/testtoken"

	// Set up denom authority
	denomAuth := types.DenomAuthority{
		Admin: creator,
	}
	err := suite.k.SetDenomAuthority(suite.ctx, denom, denomAuth)
	suite.Require().NoError(err)

	msg := &types.MsgChangeAdmin{
		Creator:  creator,
		Denom:    denom,
		NewAdmin: "invalid-address",
	}

	res, err := suite.msgServer.ChangeAdmin(suite.ctx, msg)

	suite.Require().Error(err)
	suite.Require().Nil(res)
	suite.Require().Contains(err.Error(), "decoding bech32 failed")
}

func (suite *IntegrationTestSuite) TestChangeAdmin_Unauthorized() {
	creator := sdk.AccAddress("creator").String()
	unauthorized := sdk.AccAddress("unauthorized").String()
	newAdmin := sdk.AccAddress("newadmin").String()
	denom := "factory/" + creator + "/testtoken"

	// Set up denom authority
	denomAuth := types.DenomAuthority{
		Admin: creator,
	}
	err := suite.k.SetDenomAuthority(suite.ctx, denom, denomAuth)
	suite.Require().NoError(err)

	msg := &types.MsgChangeAdmin{
		Creator:  unauthorized,
		Denom:    denom,
		NewAdmin: newAdmin,
	}

	res, err := suite.msgServer.ChangeAdmin(suite.ctx, msg)

	suite.Require().Error(err)
	suite.Require().Nil(res)
	suite.Require().Equal(types.ErrUnauthorized, err)
}

func (suite *IntegrationTestSuite) TestSetDenomMetadata_ValidRequest() {
	creator := sdk.AccAddress("creator").String()
	denom := "factory/" + creator + "/testtoken"

	// Set up denom authority
	denomAuth := types.DenomAuthority{
		Admin: creator,
	}
	err := suite.k.SetDenomAuthority(suite.ctx, denom, denomAuth)
	suite.Require().NoError(err)

	metadata := banktypes.Metadata{
		Base:        denom,
		Display:     "testtoken",
		Name:        "Test Token",
		Symbol:      "TEST",
		Description: "A test token",
		DenomUnits: []*banktypes.DenomUnit{
			{
				Denom:    denom,
				Exponent: 0,
			},
			{
				Denom:    "testtoken",
				Exponent: 6,
			},
		},
	}

	// Mock bank keeper expectation
	suite.bank.EXPECT().SetDenomMetaData(suite.ctx, metadata).Times(1)

	msg := &types.MsgSetDenomMetadata{
		Creator:  creator,
		Metadata: metadata,
	}

	res, err := suite.msgServer.SetDenomMetadata(suite.ctx, msg)

	suite.Require().NoError(err)
	suite.Require().NotNil(res)
}

func (suite *IntegrationTestSuite) TestSetDenomMetadata_Unauthorized() {
	creator := sdk.AccAddress("creator").String()
	unauthorized := sdk.AccAddress("unauthorized").String()
	denom := "factory/" + creator + "/testtoken"

	// Set up denom authority
	denomAuth := types.DenomAuthority{
		Admin: creator,
	}
	err := suite.k.SetDenomAuthority(suite.ctx, denom, denomAuth)
	suite.Require().NoError(err)

	// Create valid metadata that passes validation
	metadata := banktypes.Metadata{
		Base:        denom,
		Display:     "testtoken",
		Name:        "Test Token",
		Symbol:      "TEST",
		Description: "A test token",
		DenomUnits: []*banktypes.DenomUnit{
			{
				Denom:    denom,
				Exponent: 0,
			},
			{
				Denom:    "testtoken",
				Exponent: 6,
			},
		},
	}

	msg := &types.MsgSetDenomMetadata{
		Creator:  unauthorized,
		Metadata: metadata,
	}

	res, err := suite.msgServer.SetDenomMetadata(suite.ctx, msg)

	suite.Require().Error(err)
	suite.Require().Nil(res)
	suite.Require().Equal(types.ErrUnauthorized, err)
}
