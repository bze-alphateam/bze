package keeper_test

import (
	"errors"

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
		DenomUnits: []*banktypes.DenomUnit{
			{Denom: expectedDenom, Exponent: 0},
			{Denom: "TESTTOKEN", Exponent: 6},
		},
		Base:    expectedDenom,
		Display: "TESTTOKEN",
		Name:    subdenom,
		Symbol:  "TESTTOKEN",
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
	suite.Require().True(authority.Mintable, "a new denom is mintable")
}

// A new denom gets bank metadata that passes bank's own validation (BZE-215):
// name = subdenom, symbol = UPPER(subdenom) and a 6-decimal display unit
// named like the symbol.
func (suite *IntegrationTestSuite) TestCreateDenom_DefaultMetadataIsValid() {
	creator := sdk.AccAddress("creator").String()
	suite.Require().NoError(suite.k.SetParams(suite.ctx, types.Params{CreateDenomFee: sdk.NewInt64Coin("ubze", 0)}))

	tests := []struct {
		name     string
		subdenom string
		symbol   string
	}{
		{"lowercase", "uvdl", "UVDL"},
		{"already uppercase", "GGE", "GGE"},
		{"letter then digits", "c2m", "C2M"},
		{"dots and dashes", "u.token-1", "U.TOKEN-1"},
	}
	for _, tc := range tests {
		suite.Run(tc.name, func() {
			denom := "factory/" + creator + "/" + tc.subdenom
			suite.bank.EXPECT().HasSupply(suite.ctx, tc.subdenom).Return(false).Times(1)
			suite.bank.EXPECT().GetDenomMetaData(suite.ctx, denom).Return(banktypes.Metadata{}, false).Times(2)

			var stored banktypes.Metadata
			suite.bank.EXPECT().SetDenomMetaData(suite.ctx, gomock.Any()).Do(func(_ any, md banktypes.Metadata) {
				stored = md
			}).Times(1)

			res, err := suite.msgServer.CreateDenom(suite.ctx, &types.MsgCreateDenom{Creator: creator, Subdenom: tc.subdenom})
			suite.Require().NoError(err)
			suite.Require().Equal(denom, res.NewDenom)

			suite.Require().NoError(stored.Validate())
			suite.Require().Equal(banktypes.Metadata{
				DenomUnits: []*banktypes.DenomUnit{
					{Denom: denom, Exponent: 0},
					{Denom: tc.symbol, Exponent: 6},
				},
				Base:    denom,
				Display: tc.symbol,
				Name:    tc.subdenom,
				Symbol:  tc.symbol,
			}, stored)
		})
	}
}

// A subdenom that is not a valid coin denom itself (a digit first, shorter
// than 3 characters, a character a denom cannot hold) is refused by the msg
// server too, not only by ValidateBasic, and nothing is charged or stored.
func (suite *IntegrationTestSuite) TestCreateDenom_InvalidSubdenomRefused() {
	creator := sdk.AccAddress("creator").String()
	suite.Require().NoError(suite.k.SetParams(suite.ctx, types.Params{CreateDenomFee: sdk.NewInt64Coin("ubze", 1000)}))

	for _, subdenom := range []string{"2mars", "1000000000", "ab", "tok!"} {
		suite.Run(subdenom, func() {
			suite.bank.EXPECT().HasSupply(suite.ctx, subdenom).Return(false).Times(1)

			_, err := suite.msgServer.CreateDenom(suite.ctx, &types.MsgCreateDenom{Creator: creator, Subdenom: subdenom})
			suite.Require().ErrorIs(err, types.ErrInvalidSubdenom)
			suite.Require().ErrorContains(err, "invalid denom: "+subdenom)

			_, err = suite.k.GetDenomAuthority(suite.ctx, "factory/"+creator+"/"+subdenom)
			suite.Require().Error(err)
		})
	}
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
	amount := "1000"
	coins := amount + denom

	// Set up denom authority
	denomAuth := types.DenomAuthority{
		Admin:    creator,
		Mintable: true,
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
		Coins:   coins,
	}

	res, err := suite.msgServer.Mint(suite.ctx, msg)

	suite.Require().NoError(err)
	suite.Require().NotNil(res)
}

func (suite *IntegrationTestSuite) TestMint_InvalidAmount() {
	creator := sdk.AccAddress("creator").String()

	msg := &types.MsgMint{
		Creator: creator,
		Coins:   "invalid-amount",
	}

	res, err := suite.msgServer.Mint(suite.ctx, msg)

	suite.Require().Error(err)
	suite.Require().Nil(res)
	suite.Require().Contains(err.Error(), "invalid amount")
}

func (suite *IntegrationTestSuite) TestMint_DenomDoesNotExist() {
	creator := sdk.AccAddress("creator").String()
	denom := "factory/" + creator + "/nonexistent"
	coins := "1000" + denom

	// Mock denom doesn't exist
	suite.bank.EXPECT().GetDenomMetaData(suite.ctx, denom).Return(banktypes.Metadata{}, false).Times(1)

	msg := &types.MsgMint{
		Creator: creator,
		Coins:   coins,
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
	coins := "1000" + denom

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
		Coins:   coins,
	}

	res, err := suite.msgServer.Mint(suite.ctx, msg)

	suite.Require().Error(err)
	suite.Require().Nil(res)
	suite.Require().Equal(types.ErrUnauthorized, err)
}

func (suite *IntegrationTestSuite) TestBurn_ValidRequest() {
	creator := sdk.AccAddress("creator").String()
	denom := "factory/" + creator + "/testtoken"
	amount := "1000"
	coins := amount + denom

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
		Coins:   coins,
	}

	res, err := suite.msgServer.Burn(suite.ctx, msg)

	suite.Require().NoError(err)
	suite.Require().NotNil(res)
}

func (suite *IntegrationTestSuite) TestBurn_InvalidAmount() {
	creator := sdk.AccAddress("creator").String()

	msg := &types.MsgBurn{
		Creator: creator,
		Coins:   "invalid-amount",
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
	coins := "1000" + denom

	// Set up denom authority with different admin
	denomAuth := types.DenomAuthority{
		Admin: creator,
	}
	err := suite.k.SetDenomAuthority(suite.ctx, denom, denomAuth)
	suite.Require().NoError(err)

	msg := &types.MsgBurn{
		Creator: unauthorized,
		Coins:   coins,
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

// renounceAdmin sets up a denom with `creator` as admin, then calls
// ChangeAdmin with NewAdmin="" so the denom enters the locked state.
// Used by the post-renounce invariant tests.
func (suite *IntegrationTestSuite) renounceAdmin(creator, denom string) {
	denomAuth := types.DenomAuthority{Admin: creator, Mintable: true}
	err := suite.k.SetDenomAuthority(suite.ctx, denom, denomAuth)
	suite.Require().NoError(err)

	res, err := suite.msgServer.ChangeAdmin(suite.ctx, &types.MsgChangeAdmin{
		Creator:  creator,
		Denom:    denom,
		NewAdmin: "",
	})
	suite.Require().NoError(err)
	suite.Require().NotNil(res)

	updated, err := suite.k.GetDenomAuthority(suite.ctx, denom)
	suite.Require().NoError(err)
	suite.Require().Equal("", updated.Admin)
	suite.Require().False(updated.Mintable)
}

// After admin is renounced (set to ""), no caller — including the former
// admin — can mint. The denom is permanently mint-locked.
func (suite *IntegrationTestSuite) TestChangeAdmin_AfterRenounce_MintBlocked() {
	creator := sdk.AccAddress("creator").String()
	denom := "factory/" + creator + "/testtoken"
	suite.renounceAdmin(creator, denom)

	suite.bank.EXPECT().GetDenomMetaData(suite.ctx, denom).Return(banktypes.Metadata{Base: denom}, true).Times(1)

	res, err := suite.msgServer.Mint(suite.ctx, &types.MsgMint{
		Creator: creator,
		Coins:   "1000" + denom,
	})

	suite.Require().Error(err)
	suite.Require().Nil(res)
	suite.Require().Equal(types.ErrUnauthorized, err)
}

// After renounce, the former admin can no longer burn either.
func (suite *IntegrationTestSuite) TestChangeAdmin_AfterRenounce_BurnBlocked() {
	creator := sdk.AccAddress("creator").String()
	denom := "factory/" + creator + "/testtoken"
	suite.renounceAdmin(creator, denom)

	res, err := suite.msgServer.Burn(suite.ctx, &types.MsgBurn{
		Creator: creator,
		Coins:   "1000" + denom,
	})

	suite.Require().Error(err)
	suite.Require().Nil(res)
	suite.Require().Equal(types.ErrUnauthorized, err)
}

// After renounce, no one — not even the former admin — can claim back the
// admin role. Renounce is one-way.
func (suite *IntegrationTestSuite) TestChangeAdmin_AfterRenounce_ChangeAdminBlocked() {
	creator := sdk.AccAddress("creator").String()
	newAdmin := sdk.AccAddress("newadmin").String()
	denom := "factory/" + creator + "/testtoken"
	suite.renounceAdmin(creator, denom)

	res, err := suite.msgServer.ChangeAdmin(suite.ctx, &types.MsgChangeAdmin{
		Creator:  creator,
		Denom:    denom,
		NewAdmin: newAdmin,
	})

	suite.Require().Error(err)
	suite.Require().Nil(res)
	suite.Require().Equal(types.ErrUnauthorized, err)
}

// After renounce, denom metadata is also frozen.
func (suite *IntegrationTestSuite) TestChangeAdmin_AfterRenounce_SetDenomMetadataBlocked() {
	creator := sdk.AccAddress("creator").String()
	denom := "factory/" + creator + "/testtoken"
	suite.renounceAdmin(creator, denom)

	metadata := banktypes.Metadata{
		Base:    denom,
		Display: "testtoken",
		Name:    "Test Token",
		Symbol:  "TEST",
		DenomUnits: []*banktypes.DenomUnit{
			{Denom: denom, Exponent: 0},
			{Denom: "testtoken", Exponent: 6},
		},
	}

	res, err := suite.msgServer.SetDenomMetadata(suite.ctx, &types.MsgSetDenomMetadata{
		Creator:  creator,
		Metadata: metadata,
	})

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
