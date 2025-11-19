package keeper_test

import (
	"github.com/bze-alphateam/bze/x/tokenfactory/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (suite *IntegrationTestSuite) TestSetAndGetParams() {
	// Test data
	params := types.Params{
		CreateDenomFee: sdk.NewInt64Coin("ubze", 1000),
	}

	// Test SetParams
	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	// Test GetParams
	retrievedParams := suite.k.GetParams(suite.ctx)
	suite.Require().True(params.CreateDenomFee.Equal(&retrievedParams.CreateDenomFee))
}

func (suite *IntegrationTestSuite) TestSetParams_UpdateExisting() {
	// Set initial params
	initialParams := types.Params{
		CreateDenomFee: sdk.NewInt64Coin("ubze", 500),
	}

	err := suite.k.SetParams(suite.ctx, initialParams)
	suite.Require().NoError(err)

	// Verify initial params
	params := suite.k.GetParams(suite.ctx)
	suite.Require().True(initialParams.CreateDenomFee.Equal(&params.CreateDenomFee))

	// Update params
	updatedParams := types.Params{
		CreateDenomFee: sdk.NewInt64Coin("uatom", 2000),
	}

	err = suite.k.SetParams(suite.ctx, updatedParams)
	suite.Require().NoError(err)

	// Verify updated params
	params = suite.k.GetParams(suite.ctx)
	suite.Require().True(updatedParams.CreateDenomFee.Equal(&params.CreateDenomFee))
}

func (suite *IntegrationTestSuite) TestParams_ZeroValues() {
	// Test params with zero fee
	params := types.Params{
		CreateDenomFee: sdk.Coin{}, // Zero coin
	}

	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	// Verify zero values are stored and retrieved correctly
	retrievedParams := suite.k.GetParams(suite.ctx)
	suite.Require().True(retrievedParams.CreateDenomFee.IsZero())
}

func (suite *IntegrationTestSuite) TestParams_LargeValues() {
	// Test params with large fee values
	params := types.Params{
		CreateDenomFee: sdk.NewInt64Coin("ubze", 9223372036854775807), // max int64
	}

	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	// Verify large values are stored correctly
	retrievedParams := suite.k.GetParams(suite.ctx)
	suite.Require().True(params.CreateDenomFee.Equal(&retrievedParams.CreateDenomFee))
}

func (suite *IntegrationTestSuite) TestParams_DifferentDenoms() {
	// Test params with different denominations
	testCases := []types.Params{
		{CreateDenomFee: sdk.NewInt64Coin("ubze", 1000)},
		{CreateDenomFee: sdk.NewInt64Coin("uatom", 500)},
		{CreateDenomFee: sdk.NewInt64Coin("ustake", 2000)},
	}

	for _, testParams := range testCases {
		err := suite.k.SetParams(suite.ctx, testParams)
		suite.Require().NoError(err)

		retrievedParams := suite.k.GetParams(suite.ctx)
		suite.Require().True(testParams.CreateDenomFee.Equal(&retrievedParams.CreateDenomFee))
	}
}
