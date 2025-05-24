package keeper_test

import (
	"cosmossdk.io/math"

	"github.com/bze-alphateam/bze/x/cointrunk/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (suite *IntegrationTestSuite) TestSetAndGetParams() {
	// Test data
	tax, err := math.LegacyNewDecFromStr("0.1")
	suite.Require().NoError(err)

	params := types.Params{
		AnonArticleLimit: 100,
		AnonArticleCost:  sdk.NewInt64Coin("ubze", 1000),
		PublisherRespectParams: types.PublisherRespectParams{
			Tax:   tax,
			Denom: "ubze",
		},
	}

	// Test SetParams
	err = suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	// Test GetParams
	retrievedParams := suite.k.GetParams(suite.ctx)
	suite.Require().Equal(params.AnonArticleLimit, retrievedParams.AnonArticleLimit)
	suite.Require().True(params.AnonArticleCost.Equal(&retrievedParams.AnonArticleCost))
	suite.Require().True(params.PublisherRespectParams.Tax.Equal(retrievedParams.PublisherRespectParams.Tax))
	suite.Require().Equal(params.PublisherRespectParams.Denom, retrievedParams.PublisherRespectParams.Denom)
}

func (suite *IntegrationTestSuite) TestAnonArticleLimit() {
	// Set params with specific limit
	params := types.Params{
		AnonArticleLimit: 50,
		AnonArticleCost:  sdk.NewInt64Coin("ubze", 500),
		PublisherRespectParams: types.PublisherRespectParams{
			Tax:   math.LegacyMustNewDecFromStr("0.05"),
			Denom: "ubze",
		},
	}

	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	// Test AnonArticleLimit helper method
	limit := suite.k.AnonArticleLimit(suite.ctx)
	suite.Require().Equal(uint64(50), limit)
}

func (suite *IntegrationTestSuite) TestAnonArticleCost() {
	// Set params with specific cost
	expectedCost := sdk.NewInt64Coin("utoken", 2000)
	params := types.Params{
		AnonArticleLimit: 25,
		AnonArticleCost:  expectedCost,
		PublisherRespectParams: types.PublisherRespectParams{
			Tax:   math.LegacyMustNewDecFromStr("0.15"),
			Denom: "utoken",
		},
	}

	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	// Test AnonArticleCost helper method
	cost := suite.k.AnonArticleCost(suite.ctx)
	suite.Require().True(expectedCost.Equal(&cost))
}

func (suite *IntegrationTestSuite) TestPublisherRespectParams() {
	// Set params with specific respect params
	expectedTax := math.LegacyMustNewDecFromStr("0.25")
	expectedDenom := "ustake"
	params := types.Params{
		AnonArticleLimit: 75,
		AnonArticleCost:  sdk.NewInt64Coin("ustake", 1500),
		PublisherRespectParams: types.PublisherRespectParams{
			Tax:   expectedTax,
			Denom: expectedDenom,
		},
	}

	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	// Test PublisherRespectParams helper method
	respectParams := suite.k.PublisherRespectParams(suite.ctx)
	suite.Require().True(expectedTax.Equal(respectParams.Tax))
	suite.Require().Equal(expectedDenom, respectParams.Denom)
}

func (suite *IntegrationTestSuite) TestSetParams_UpdateExisting() {
	// Set initial params
	initialParams := types.Params{
		AnonArticleLimit: 10,
		AnonArticleCost:  sdk.NewInt64Coin("ubze", 100),
		PublisherRespectParams: types.PublisherRespectParams{
			Tax:   math.LegacyMustNewDecFromStr("0.01"),
			Denom: "ubze",
		},
	}

	err := suite.k.SetParams(suite.ctx, initialParams)
	suite.Require().NoError(err)

	// Verify initial params
	params := suite.k.GetParams(suite.ctx)
	suite.Require().Equal(uint64(10), params.AnonArticleLimit)

	// Update params
	updatedParams := types.Params{
		AnonArticleLimit: 200,
		AnonArticleCost:  sdk.NewInt64Coin("uatom", 5000),
		PublisherRespectParams: types.PublisherRespectParams{
			Tax:   math.LegacyMustNewDecFromStr("0.5"),
			Denom: "uatom",
		},
	}

	err = suite.k.SetParams(suite.ctx, updatedParams)
	suite.Require().NoError(err)

	// Verify updated params
	params = suite.k.GetParams(suite.ctx)
	suite.Require().Equal(uint64(200), params.AnonArticleLimit)
	expectedCoin := sdk.NewInt64Coin("uatom", 5000)
	suite.Require().True(expectedCoin.Equal(&params.AnonArticleCost))
	suite.Require().True(math.LegacyMustNewDecFromStr("0.5").Equal(params.PublisherRespectParams.Tax))
	suite.Require().Equal("uatom", params.PublisherRespectParams.Denom)
}

func (suite *IntegrationTestSuite) TestParams_ZeroValues() {
	// Test params with zero/empty values
	params := types.Params{
		AnonArticleLimit: 0,
		AnonArticleCost:  sdk.Coin{},
		PublisherRespectParams: types.PublisherRespectParams{
			Tax:   math.LegacyZeroDec(),
			Denom: "",
		},
	}

	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	// Verify zero values are stored and retrieved correctly
	retrievedParams := suite.k.GetParams(suite.ctx)
	suite.Require().Equal(uint64(0), retrievedParams.AnonArticleLimit)
	suite.Require().True(retrievedParams.AnonArticleCost.IsZero())
	suite.Require().True(retrievedParams.PublisherRespectParams.Tax.IsZero())
	suite.Require().Equal("", retrievedParams.PublisherRespectParams.Denom)

	// Test helper methods with zero values
	limit := suite.k.AnonArticleLimit(suite.ctx)
	suite.Require().Equal(uint64(0), limit)

	cost := suite.k.AnonArticleCost(suite.ctx)
	suite.Require().True(cost.IsZero())

	respectParams := suite.k.PublisherRespectParams(suite.ctx)
	suite.Require().True(respectParams.Tax.IsZero())
	suite.Require().Equal("", respectParams.Denom)
}

func (suite *IntegrationTestSuite) TestParams_LargeValues() {
	// Test params with large values
	largeTax, err := math.LegacyNewDecFromStr("0.999999999999999999")
	suite.Require().NoError(err)

	params := types.Params{
		AnonArticleLimit: 18446744073709551615,                          // max uint64
		AnonArticleCost:  sdk.NewInt64Coin("ubze", 9223372036854775807), // max int64
		PublisherRespectParams: types.PublisherRespectParams{
			Tax:   largeTax,
			Denom: "verylongdenominationthatcouldcauseissues",
		},
	}

	err = suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	// Verify large values are stored correctly
	retrievedParams := suite.k.GetParams(suite.ctx)
	suite.Require().Equal(uint64(18446744073709551615), retrievedParams.AnonArticleLimit)
	suite.Require().True(params.AnonArticleCost.Equal(&retrievedParams.AnonArticleCost))
	suite.Require().True(largeTax.Equal(retrievedParams.PublisherRespectParams.Tax))
	suite.Require().Equal("verylongdenominationthatcouldcauseissues", retrievedParams.PublisherRespectParams.Denom)
}
