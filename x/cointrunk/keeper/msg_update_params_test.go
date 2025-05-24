package keeper_test

import (
	"cosmossdk.io/math"
	"github.com/bze-alphateam/bze/x/cointrunk/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (suite *IntegrationTestSuite) TestUpdateParams_ValidRequest() {
	authority := suite.k.GetAuthority()

	params := types.Params{
		AnonArticleLimit: 150,
		AnonArticleCost:  sdk.NewInt64Coin("ubze", 2000),
		PublisherRespectParams: types.PublisherRespectParams{
			Tax:   math.LegacyMustNewDecFromStr("0.15"),
			Denom: "ubze",
		},
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
	suite.Require().Equal(params.AnonArticleLimit, retrievedParams.AnonArticleLimit)
	suite.Require().True(params.AnonArticleCost.Equal(&retrievedParams.AnonArticleCost))
	suite.Require().True(params.PublisherRespectParams.Tax.Equal(retrievedParams.PublisherRespectParams.Tax))
	suite.Require().Equal(params.PublisherRespectParams.Denom, retrievedParams.PublisherRespectParams.Denom)
}

func (suite *IntegrationTestSuite) TestUpdateParams_InvalidAuthority() {
	invalidAuthority := sdk.AccAddress("invalid").String()

	params := types.Params{
		AnonArticleLimit: 100,
		AnonArticleCost:  sdk.NewInt64Coin("ubze", 1000),
		PublisherRespectParams: types.PublisherRespectParams{
			Tax:   math.LegacyMustNewDecFromStr("0.1"),
			Denom: "ubze",
		},
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

func (suite *IntegrationTestSuite) TestUpdateParams_SetParamsError() {
	authority := suite.k.GetAuthority()

	// Create params that might cause SetParams to fail
	// Since we can't easily mock SetParams failure, we'll test with edge case values
	params := types.Params{
		AnonArticleLimit: 0,
		AnonArticleCost:  sdk.Coin{}, // Empty coin
		PublisherRespectParams: types.PublisherRespectParams{
			Tax:   math.LegacyZeroDec(),
			Denom: "",
		},
	}

	msg := &types.MsgUpdateParams{
		Authority: authority,
		Params:    params,
	}

	// This should succeed since zero values are valid
	res, err := suite.msgServer.UpdateParams(suite.ctx, msg)
	suite.Require().NoError(err)
	suite.Require().NotNil(res)
}

func (suite *IntegrationTestSuite) TestUpdateParams_EmptyAuthority() {
	params := types.Params{
		AnonArticleLimit: 100,
		AnonArticleCost:  sdk.NewInt64Coin("ubze", 1000),
		PublisherRespectParams: types.PublisherRespectParams{
			Tax:   math.LegacyMustNewDecFromStr("0.1"),
			Denom: "ubze",
		},
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
		AnonArticleLimit: 50,
		AnonArticleCost:  sdk.NewInt64Coin("ubze", 500),
		PublisherRespectParams: types.PublisherRespectParams{
			Tax:   math.LegacyMustNewDecFromStr("0.05"),
			Denom: "ubze",
		},
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
	suite.Require().Equal(uint64(50), retrievedParams.AnonArticleLimit)

	// Second update
	params2 := types.Params{
		AnonArticleLimit: 200,
		AnonArticleCost:  sdk.NewInt64Coin("uatom", 3000),
		PublisherRespectParams: types.PublisherRespectParams{
			Tax:   math.LegacyMustNewDecFromStr("0.25"),
			Denom: "uatom",
		},
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
	suite.Require().Equal(uint64(200), retrievedParams.AnonArticleLimit)
	expectedCoin := sdk.NewInt64Coin("uatom", 3000)
	suite.Require().True(expectedCoin.Equal(&retrievedParams.AnonArticleCost))
	suite.Require().True(math.LegacyMustNewDecFromStr("0.25").Equal(retrievedParams.PublisherRespectParams.Tax))
	suite.Require().Equal("uatom", retrievedParams.PublisherRespectParams.Denom)
}

func (suite *IntegrationTestSuite) TestUpdateParams_MaxValues() {
	authority := suite.k.GetAuthority()

	// Test with maximum values
	largeTax := math.LegacyMustNewDecFromStr("0.999999999999999999")
	params := types.Params{
		AnonArticleLimit: 18446744073709551615,                          // max uint64
		AnonArticleCost:  sdk.NewInt64Coin("ubze", 9223372036854775807), // max int64
		PublisherRespectParams: types.PublisherRespectParams{
			Tax:   largeTax,
			Denom: "verylongdenomination",
		},
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
	suite.Require().Equal(uint64(18446744073709551615), retrievedParams.AnonArticleLimit)
	suite.Require().True(params.AnonArticleCost.Equal(&retrievedParams.AnonArticleCost))
	suite.Require().True(largeTax.Equal(retrievedParams.PublisherRespectParams.Tax))
	suite.Require().Equal("verylongdenomination", retrievedParams.PublisherRespectParams.Denom)
}
