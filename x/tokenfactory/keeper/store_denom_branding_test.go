package keeper_test

import (
	"github.com/bze-alphateam/bze/x/tokenfactory/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func testBranding() types.DenomBranding {
	return types.DenomBranding{
		Font: "inter",
		Light: &types.BrandingColors{
			Background: "#FFFFFF",
			Text:       "#111111",
			Primary:    "#1a2b3c",
			Secondary:  "#a1b2c3",
		},
		Dark: &types.BrandingColors{
			Background: "#000000",
			Text:       "#eeeeee",
			Primary:    "#c0ffee",
			Secondary:  "#abcdef",
		},
	}
}

func (suite *IntegrationTestSuite) TestSetAndGetDenomBranding() {
	denom := "factory/" + sdk.AccAddress("creator").String() + "/test"
	branding := testBranding()

	err := suite.k.SetDenomBranding(suite.ctx, denom, branding)
	suite.Require().NoError(err)

	retrieved, found := suite.k.GetDenomBranding(suite.ctx, denom)
	suite.Require().True(found)
	suite.Require().True(branding.Equal(retrieved))
}

func (suite *IntegrationTestSuite) TestGetDenomBranding_NotFound() {
	_, found := suite.k.GetDenomBranding(suite.ctx, "factory/nobody/nothing")
	suite.Require().False(found)
}

func (suite *IntegrationTestSuite) TestStoreDenomBranding_Overwrite() {
	denom := "factory/" + sdk.AccAddress("creator").String() + "/test"

	first := testBranding()
	err := suite.k.SetDenomBranding(suite.ctx, denom, first)
	suite.Require().NoError(err)

	second := testBranding()
	second.Font = "jetbrains-mono"
	second.Light.Background = "#123456"
	err = suite.k.SetDenomBranding(suite.ctx, denom, second)
	suite.Require().NoError(err)

	retrieved, found := suite.k.GetDenomBranding(suite.ctx, denom)
	suite.Require().True(found)
	suite.Require().True(second.Equal(retrieved))
	suite.Require().False(first.Equal(retrieved))
}

func (suite *IntegrationTestSuite) TestStoreDenomBranding_InvalidPackage() {
	denom := "factory/" + sdk.AccAddress("creator").String() + "/test"

	branding := testBranding()
	branding.Dark = nil

	err := suite.k.SetDenomBranding(suite.ctx, denom, branding)
	suite.Require().ErrorIs(err, types.ErrInvalidBranding)

	_, found := suite.k.GetDenomBranding(suite.ctx, denom)
	suite.Require().False(found)
}

func (suite *IntegrationTestSuite) TestRemoveDenomBranding() {
	denom := "factory/" + sdk.AccAddress("creator").String() + "/test"

	err := suite.k.SetDenomBranding(suite.ctx, denom, testBranding())
	suite.Require().NoError(err)

	suite.k.RemoveDenomBranding(suite.ctx, denom)
	_, found := suite.k.GetDenomBranding(suite.ctx, denom)
	suite.Require().False(found)

	// removing again is a no-op
	suite.k.RemoveDenomBranding(suite.ctx, denom)
	_, found = suite.k.GetDenomBranding(suite.ctx, denom)
	suite.Require().False(found)
}

func (suite *IntegrationTestSuite) TestGetAllDenomBrandings() {
	suite.Require().Empty(suite.k.GetAllDenomBrandings(suite.ctx))

	denom1 := "factory/" + sdk.AccAddress("creator1").String() + "/aaa"
	denom2 := "factory/" + sdk.AccAddress("creator2").String() + "/bbb"

	branding1 := testBranding()
	branding2 := testBranding()
	branding2.Font = "roboto"

	suite.Require().NoError(suite.k.SetDenomBranding(suite.ctx, denom1, branding1))
	suite.Require().NoError(suite.k.SetDenomBranding(suite.ctx, denom2, branding2))

	all := suite.k.GetAllDenomBrandings(suite.ctx)
	suite.Require().Len(all, 2)

	byDenom := make(map[string]types.DenomBrandingRecord)
	for _, record := range all {
		byDenom[record.Denom] = record
	}
	suite.Require().True(branding1.Equal(*byDenom[denom1].Branding))
	suite.Require().True(branding2.Equal(*byDenom[denom2].Branding))
}
