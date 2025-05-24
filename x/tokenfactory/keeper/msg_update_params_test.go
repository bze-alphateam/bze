package keeper_test

import (
	"github.com/bze-alphateam/bze/x/tokenfactory/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (suite *IntegrationTestSuite) TestUpdateParams_ValidRequest() {
	authority := suite.k.GetAuthority()

	params := types.Params{
		CreateDenomFee: sdk.NewInt64Coin("ubze", 2000),
	}

	msg := &types.MsgUpdateParams{
		Authority: authority,
		Params:    params,
	}

	res, err := suite.msgServer.UpdateParams(suite.ctx, msg)

	suite.Require().NoError(err)
	suite.Require().NotNil(res)

	// Verify params were updated
	retrievedParams := suite.k.GetParams(suite.ctx)
	suite.Require().True(params.CreateDenomFee.Equal(&retrievedParams.CreateDenomFee))
}

func (suite *IntegrationTestSuite) TestUpdateParams_InvalidAuthority() {
	invalidAuthority := sdk.AccAddress("invalid").String()

	params := types.Params{
		CreateDenomFee: sdk.NewInt64Coin("ubze", 1000),
	}

	msg := &types.MsgUpdateParams{
		Authority: invalidAuthority,
		Params:    params,
	}

	res, err := suite.msgServer.UpdateParams(suite.ctx, msg)

	suite.Require().Error(err)
	suite.Require().Nil(res)
	suite.Require().Contains(err.Error(), "invalid authority")
}

func (suite *IntegrationTestSuite) TestUpdateParams_EmptyAuthority() {
	params := types.Params{
		CreateDenomFee: sdk.NewInt64Coin("ubze", 1000),
	}

	msg := &types.MsgUpdateParams{
		Authority: "", // Empty authority
		Params:    params,
	}

	res, err := suite.msgServer.UpdateParams(suite.ctx, msg)

	suite.Require().Error(err)
	suite.Require().Nil(res)
	suite.Require().Contains(err.Error(), "invalid authority")
}

func (suite *IntegrationTestSuite) TestUpdateParams_MultipleUpdates() {
	authority := suite.k.GetAuthority()

	// First update
	params1 := types.Params{
		CreateDenomFee: sdk.NewInt64Coin("ubze", 500),
	}

	msg1 := &types.MsgUpdateParams{
		Authority: authority,
		Params:    params1,
	}

	res, err := suite.msgServer.UpdateParams(suite.ctx, msg1)
	suite.Require().NoError(err)
	suite.Require().NotNil(res)

	// Verify first update
	retrievedParams := suite.k.GetParams(suite.ctx)
	suite.Require().True(params1.CreateDenomFee.Equal(&retrievedParams.CreateDenomFee))

	// Second update
	params2 := types.Params{
		CreateDenomFee: sdk.NewInt64Coin("uatom", 3000),
	}

	msg2 := &types.MsgUpdateParams{
		Authority: authority,
		Params:    params2,
	}

	res, err = suite.msgServer.UpdateParams(suite.ctx, msg2)
	suite.Require().NoError(err)
	suite.Require().NotNil(res)

	// Verify second update overwrote first
	retrievedParams = suite.k.GetParams(suite.ctx)
	suite.Require().True(params2.CreateDenomFee.Equal(&retrievedParams.CreateDenomFee))
}

func (suite *IntegrationTestSuite) TestUpdateParams_ZeroFee() {
	authority := suite.k.GetAuthority()

	// Test with zero fee
	params := types.Params{
		CreateDenomFee: sdk.Coin{}, // Zero coin
	}

	msg := &types.MsgUpdateParams{
		Authority: authority,
		Params:    params,
	}

	res, err := suite.msgServer.UpdateParams(suite.ctx, msg)
	suite.Require().NoError(err)
	suite.Require().NotNil(res)

	// Verify zero fee was set
	retrievedParams := suite.k.GetParams(suite.ctx)
	suite.Require().True(retrievedParams.CreateDenomFee.IsZero())
}

func (suite *IntegrationTestSuite) TestUpdateParams_MaxValues() {
	authority := suite.k.GetAuthority()

	// Test with maximum values
	params := types.Params{
		CreateDenomFee: sdk.NewInt64Coin("ubze", 9223372036854775807), // max int64
	}

	msg := &types.MsgUpdateParams{
		Authority: authority,
		Params:    params,
	}

	res, err := suite.msgServer.UpdateParams(suite.ctx, msg)
	suite.Require().NoError(err)
	suite.Require().NotNil(res)

	// Verify large values were stored correctly
	retrievedParams := suite.k.GetParams(suite.ctx)
	suite.Require().True(params.CreateDenomFee.Equal(&retrievedParams.CreateDenomFee))
}

func (suite *IntegrationTestSuite) TestUpdateParams_DifferentDenoms() {
	authority := suite.k.GetAuthority()

	// Test updating with different denominations
	testCases := []types.Params{
		{CreateDenomFee: sdk.NewInt64Coin("ubze", 1000)},
		{CreateDenomFee: sdk.NewInt64Coin("uatom", 500)},
		{CreateDenomFee: sdk.NewInt64Coin("ustake", 2000)},
	}

	for _, testParams := range testCases {
		msg := &types.MsgUpdateParams{
			Authority: authority,
			Params:    testParams,
		}

		res, err := suite.msgServer.UpdateParams(suite.ctx, msg)
		suite.Require().NoError(err)
		suite.Require().NotNil(res)

		// Verify each update was applied correctly
		retrievedParams := suite.k.GetParams(suite.ctx)
		suite.Require().True(testParams.CreateDenomFee.Equal(&retrievedParams.CreateDenomFee))
	}
}
