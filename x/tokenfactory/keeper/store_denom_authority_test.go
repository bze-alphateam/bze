package keeper_test

import (
	"github.com/bze-alphateam/bze/x/tokenfactory/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (suite *IntegrationTestSuite) TestSetAndGetDenomAuthority() {
	denom := "factory/" + sdk.AccAddress("creator").String() + "/test"

	denomAuthority := types.DenomAuthority{
		Admin: sdk.AccAddress("admin").String(),
	}

	// Test SetDenomAuthority
	err := suite.k.SetDenomAuthority(suite.ctx, denom, denomAuthority)
	suite.Require().NoError(err)

	// Test GetDenomAuthority
	retrievedAuth, err := suite.k.GetDenomAuthority(suite.ctx, denom)
	suite.Require().NoError(err)
	suite.Require().Equal(denomAuthority.Admin, retrievedAuth.Admin)
}

func (suite *IntegrationTestSuite) TestGetDenomAuthority_NotFound() {
	denom := "factory/" + sdk.AccAddress("creator").String() + "/nonexistent"

	// Test GetDenomAuthority for non-existent denom
	_, err := suite.k.GetDenomAuthority(suite.ctx, denom)

	// Should return error when denom doesn't exist
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "denom authority not found")
}

func (suite *IntegrationTestSuite) TestSetDenomAuthority_UpdateExisting() {
	denom := "factory/" + sdk.AccAddress("creator").String() + "/test"

	// Set initial authority
	initialAuth := types.DenomAuthority{
		Admin: sdk.AccAddress("admin1").String(),
	}
	err := suite.k.SetDenomAuthority(suite.ctx, denom, initialAuth)
	suite.Require().NoError(err)

	// Update authority
	updatedAuth := types.DenomAuthority{
		Admin: sdk.AccAddress("admin2").String(),
	}
	err = suite.k.SetDenomAuthority(suite.ctx, denom, updatedAuth)
	suite.Require().NoError(err)

	// Verify updated authority
	retrievedAuth, err := suite.k.GetDenomAuthority(suite.ctx, denom)
	suite.Require().NoError(err)
	suite.Require().Equal(updatedAuth.Admin, retrievedAuth.Admin)
	suite.Require().NotEqual(initialAuth.Admin, retrievedAuth.Admin)
}

func (suite *IntegrationTestSuite) TestDenomAuthority_MultipleDenoms() {
	// Test multiple denoms with different authorities
	denom1 := "factory/" + sdk.AccAddress("creator1").String() + "/token1"
	denom2 := "factory/" + sdk.AccAddress("creator2").String() + "/token2"

	auth1 := types.DenomAuthority{
		Admin: sdk.AccAddress("admin1").String(),
	}
	auth2 := types.DenomAuthority{
		Admin: sdk.AccAddress("admin2").String(),
	}

	// Set authorities for both denoms
	err := suite.k.SetDenomAuthority(suite.ctx, denom1, auth1)
	suite.Require().NoError(err)

	err = suite.k.SetDenomAuthority(suite.ctx, denom2, auth2)
	suite.Require().NoError(err)

	// Verify both authorities are stored correctly
	retrievedAuth1, err := suite.k.GetDenomAuthority(suite.ctx, denom1)
	suite.Require().NoError(err)
	suite.Require().Equal(auth1.Admin, retrievedAuth1.Admin)

	retrievedAuth2, err := suite.k.GetDenomAuthority(suite.ctx, denom2)
	suite.Require().NoError(err)
	suite.Require().Equal(auth2.Admin, retrievedAuth2.Admin)

	// Verify they're different
	suite.Require().NotEqual(retrievedAuth1.Admin, retrievedAuth2.Admin)
}

func (suite *IntegrationTestSuite) TestDenomAuthority_EmptyDenom() {
	denomAuthority := types.DenomAuthority{
		Admin: sdk.AccAddress("admin").String(),
	}

	// Test with empty denom
	err := suite.k.SetDenomAuthority(suite.ctx, "", denomAuthority)
	suite.Require().NoError(err) // Should work but store under empty key

	// Test getting with empty denom
	retrievedAuth, err := suite.k.GetDenomAuthority(suite.ctx, "")
	suite.Require().NoError(err)
	suite.Require().Equal(denomAuthority.Admin, retrievedAuth.Admin)
}

func (suite *IntegrationTestSuite) TestDenomAuthority_SpecialCharacterDenom() {
	// Test with special characters in denom
	specialDenom := "factory/bze1test123/token-with-special_chars.v2"

	denomAuthority := types.DenomAuthority{
		Admin: sdk.AccAddress("admin").String(),
	}

	err := suite.k.SetDenomAuthority(suite.ctx, specialDenom, denomAuthority)
	suite.Require().NoError(err)

	retrievedAuth, err := suite.k.GetDenomAuthority(suite.ctx, specialDenom)
	suite.Require().NoError(err)
	suite.Require().Equal(denomAuthority.Admin, retrievedAuth.Admin)
}
