package keeper_test

import (
	"github.com/bze-alphateam/bze/x/tokenfactory/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (suite *IntegrationTestSuite) TestParams_ValidRequest() {
	// Set some params first
	params := types.Params{
		CreateDenomFee: sdk.NewInt64Coin("ubze", 1000),
	}

	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	req := &types.QueryParamsRequest{}
	res, err := suite.k.Params(suite.ctx, req)

	suite.Require().NoError(err)
	suite.Require().NotNil(res)
	suite.Require().NotNil(res.Params)
	suite.Require().True(params.CreateDenomFee.Equal(&res.Params.CreateDenomFee))
}

func (suite *IntegrationTestSuite) TestParams_NilRequest() {
	res, err := suite.k.Params(suite.ctx, nil)

	suite.Require().Error(err)
	suite.Require().Nil(res)
	suite.Require().Equal(codes.InvalidArgument, status.Code(err))
	suite.Require().Contains(err.Error(), "invalid request")
}

func (suite *IntegrationTestSuite) TestParams_ContextUnwrapping() {
	// Set some params
	params := types.Params{
		CreateDenomFee: sdk.NewInt64Coin("utoken", 500),
	}

	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	req := &types.QueryParamsRequest{}

	// Test with SDK context
	res, err := suite.k.Params(suite.ctx, req)
	suite.Require().NoError(err)
	suite.Require().NotNil(res)
	suite.Require().True(params.CreateDenomFee.Equal(&res.Params.CreateDenomFee))
}

func (suite *IntegrationTestSuite) TestDenomAuthority_ValidRequest() {
	creator := sdk.AccAddress("creator").String()
	denom := "factory/" + creator + "/testtoken"

	// Set up denom authority
	denomAuth := types.DenomAuthority{
		Admin: creator,
	}
	err := suite.k.SetDenomAuthority(suite.ctx, denom, denomAuth)
	suite.Require().NoError(err)

	req := &types.QueryDenomAuthorityRequest{
		Denom: denom,
	}
	res, err := suite.k.DenomAuthority(suite.ctx, req)

	suite.Require().NoError(err)
	suite.Require().NotNil(res)
	suite.Require().NotNil(res.DenomAuthority)
	suite.Require().Equal(creator, res.DenomAuthority.Admin)
}

func (suite *IntegrationTestSuite) TestDenomAuthority_NilRequest() {
	res, err := suite.k.DenomAuthority(suite.ctx, nil)

	suite.Require().Error(err)
	suite.Require().Nil(res)
	suite.Require().Equal(codes.InvalidArgument, status.Code(err))
	suite.Require().Contains(err.Error(), "invalid request")
}

func (suite *IntegrationTestSuite) TestDenomAuthority_EmptyDenomQuery() {
	req := &types.QueryDenomAuthorityRequest{
		Denom: "",
	}
	res, err := suite.k.DenomAuthority(suite.ctx, req)

	suite.Require().Error(err)
	suite.Require().Nil(res)
	suite.Require().Equal(codes.InvalidArgument, status.Code(err))
	suite.Require().Contains(err.Error(), "invalid request")
}

func (suite *IntegrationTestSuite) TestDenomAuthority_DenomNotFound() {
	nonExistentDenom := "factory/" + sdk.AccAddress("creator").String() + "/nonexistent"

	req := &types.QueryDenomAuthorityRequest{
		Denom: nonExistentDenom,
	}
	res, err := suite.k.DenomAuthority(suite.ctx, req)

	suite.Require().Error(err)
	suite.Require().Nil(res)
	suite.Require().Contains(err.Error(), "denom authority not found")
}

func (suite *IntegrationTestSuite) TestDenomAuthority_ContextUnwrapping() {
	creator := sdk.AccAddress("creator").String()
	denom := "factory/" + creator + "/testtoken"

	// Set up denom authority
	denomAuth := types.DenomAuthority{
		Admin: creator,
	}
	err := suite.k.SetDenomAuthority(suite.ctx, denom, denomAuth)
	suite.Require().NoError(err)

	req := &types.QueryDenomAuthorityRequest{
		Denom: denom,
	}

	// Test with SDK context
	res, err := suite.k.DenomAuthority(suite.ctx, req)
	suite.Require().NoError(err)
	suite.Require().NotNil(res)
	suite.Require().Equal(creator, res.DenomAuthority.Admin)
}

func (suite *IntegrationTestSuite) TestDenomAuthority_SpecialCharacterDenomQuery() {
	creator := sdk.AccAddress("creator").String()
	specialDenom := "factory/" + creator + "/token-with-special_chars.v2"

	// Set up denom authority
	denomAuth := types.DenomAuthority{
		Admin: creator,
	}
	err := suite.k.SetDenomAuthority(suite.ctx, specialDenom, denomAuth)
	suite.Require().NoError(err)

	req := &types.QueryDenomAuthorityRequest{
		Denom: specialDenom,
	}
	res, err := suite.k.DenomAuthority(suite.ctx, req)

	suite.Require().NoError(err)
	suite.Require().NotNil(res)
	suite.Require().Equal(creator, res.DenomAuthority.Admin)
}
