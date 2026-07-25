package keeper_test

import (
	"fmt"

	"github.com/bze-alphateam/bze/x/tokenfactory/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (suite *IntegrationTestSuite) TestDenomBranding_ValidRequest() {
	denom := "factory/" + sdk.AccAddress("creator").String() + "/token"
	branding := testBranding()
	suite.Require().NoError(suite.k.SetDenomBranding(suite.ctx, denom, branding))

	res, err := suite.k.DenomBranding(suite.ctx, &types.QueryDenomBrandingRequest{Denom: denom})

	suite.Require().NoError(err)
	suite.Require().NotNil(res)
	suite.Require().NotNil(res.Branding)
	suite.Require().True(branding.Equal(res.Branding))
}

func (suite *IntegrationTestSuite) TestDenomBranding_NotFound() {
	res, err := suite.k.DenomBranding(suite.ctx, &types.QueryDenomBrandingRequest{Denom: "factory/nobody/nothing"})

	suite.Require().Error(err)
	suite.Require().Nil(res)
	suite.Require().Equal(codes.NotFound, status.Code(err))
}

func (suite *IntegrationTestSuite) TestDenomBranding_InvalidRequest() {
	res, err := suite.k.DenomBranding(suite.ctx, nil)
	suite.Require().Error(err)
	suite.Require().Nil(res)
	suite.Require().Equal(codes.InvalidArgument, status.Code(err))

	res, err = suite.k.DenomBranding(suite.ctx, &types.QueryDenomBrandingRequest{})
	suite.Require().Error(err)
	suite.Require().Nil(res)
	suite.Require().Equal(codes.InvalidArgument, status.Code(err))
}

func (suite *IntegrationTestSuite) TestAllDenomBranding_NilRequest() {
	res, err := suite.k.AllDenomBranding(suite.ctx, nil)

	suite.Require().Error(err)
	suite.Require().Nil(res)
	suite.Require().Equal(codes.InvalidArgument, status.Code(err))
}

func (suite *IntegrationTestSuite) TestAllDenomBranding_Empty() {
	res, err := suite.k.AllDenomBranding(suite.ctx, &types.QueryAllDenomBrandingRequest{})

	suite.Require().NoError(err)
	suite.Require().NotNil(res)
	suite.Require().Empty(res.DenomBrandings)
}

func (suite *IntegrationTestSuite) TestAllDenomBranding_Paginated() {
	creator := sdk.AccAddress("creator").String()
	total := 5
	for i := 0; i < total; i++ {
		denom := fmt.Sprintf("factory/%s/token%d", creator, i)
		branding := testBranding()
		suite.Require().NoError(suite.k.SetDenomBranding(suite.ctx, denom, branding))
	}

	// first page
	res, err := suite.k.AllDenomBranding(suite.ctx, &types.QueryAllDenomBrandingRequest{
		Pagination: &query.PageRequest{Limit: 3, CountTotal: true},
	})
	suite.Require().NoError(err)
	suite.Require().Len(res.DenomBrandings, 3)
	suite.Require().NotNil(res.Pagination)
	suite.Require().NotNil(res.Pagination.NextKey)
	suite.Require().Equal(uint64(total), res.Pagination.Total)

	// second page
	res2, err := suite.k.AllDenomBranding(suite.ctx, &types.QueryAllDenomBrandingRequest{
		Pagination: &query.PageRequest{Limit: 3, Key: res.Pagination.NextKey},
	})
	suite.Require().NoError(err)
	suite.Require().Len(res2.DenomBrandings, 2)

	// no overlap between pages and all denoms covered
	seen := make(map[string]struct{})
	for _, record := range append(res.DenomBrandings, res2.DenomBrandings...) {
		_, dup := seen[record.Denom]
		suite.Require().False(dup)
		seen[record.Denom] = struct{}{}
		suite.Require().NotNil(record.Branding)
	}
	suite.Require().Len(seen, total)
}
