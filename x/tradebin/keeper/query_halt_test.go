package keeper_test

import (
	"github.com/bze-alphateam/bze/x/tradebin/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (suite *IntegrationTestSuite) TestQueryHaltedDenoms_EmptyByDefault() {
	resp, err := suite.k.HaltedDenoms(suite.ctx, &types.QueryHaltedDenomsRequest{Pagination: &query.PageRequest{CountTotal: true}})
	suite.Require().NoError(err)
	suite.Require().Empty(resp.Denoms)
	suite.Require().Zero(resp.Pagination.Total)
}

func (suite *IntegrationTestSuite) TestQueryHaltedDenoms_PaginatedInStoreOrder() {
	suite.setHaltedDenoms(denomHalted, haltedIbcDenom, haltedFactoryDenom)

	first, err := suite.k.HaltedDenoms(suite.ctx, &types.QueryHaltedDenomsRequest{Pagination: &query.PageRequest{Limit: 2, CountTotal: true}})
	suite.Require().NoError(err)
	suite.Require().Equal([]string{haltedFactoryDenom, haltedIbcDenom}, first.Denoms)
	suite.Require().EqualValues(3, first.Pagination.Total)
	suite.Require().NotEmpty(first.Pagination.NextKey)

	second, err := suite.k.HaltedDenoms(suite.ctx, &types.QueryHaltedDenomsRequest{Pagination: &query.PageRequest{Key: first.Pagination.NextKey, Limit: 2}})
	suite.Require().NoError(err)
	suite.Require().Equal([]string{denomHalted}, second.Denoms)
	suite.Require().Empty(second.Pagination.NextKey)

	all, err := suite.k.HaltedDenoms(suite.ctx, &types.QueryHaltedDenomsRequest{})
	suite.Require().NoError(err)
	suite.Require().Equal([]string{haltedFactoryDenom, haltedIbcDenom, denomHalted}, all.Denoms)
}

func (suite *IntegrationTestSuite) TestQueryHaltedDenoms_NilRequest() {
	resp, err := suite.k.HaltedDenoms(suite.ctx, nil)
	suite.Require().Nil(resp)
	suite.Require().Equal(codes.InvalidArgument, status.Code(err))
}

func (suite *IntegrationTestSuite) TestQueryDenomHalted_Flag() {
	resp, err := suite.k.DenomHalted(suite.ctx, &types.QueryDenomHaltedRequest{Denom: denomHalted})
	suite.Require().NoError(err)
	suite.Require().False(resp.Halted)

	suite.setHaltedDenoms(denomHalted)

	resp, err = suite.k.DenomHalted(suite.ctx, &types.QueryDenomHaltedRequest{Denom: denomHalted})
	suite.Require().NoError(err)
	suite.Require().True(resp.Halted)

	// exact match only: ids that contain the denom are not halted, nor is any other denom
	for _, d := range []string{denomBze, denomStake, marketIdOf(haltedQuoteMarket()), denomStake + "_" + denomHalted} {
		resp, err = suite.k.DenomHalted(suite.ctx, &types.QueryDenomHaltedRequest{Denom: d})
		suite.Require().NoError(err, d)
		suite.Require().False(resp.Halted, d)
	}
}

func (suite *IntegrationTestSuite) TestQueryDenomHalted_InvalidRequests() {
	resp, err := suite.k.DenomHalted(suite.ctx, nil)
	suite.Require().Nil(resp)
	suite.Require().Equal(codes.InvalidArgument, status.Code(err))

	resp, err = suite.k.DenomHalted(suite.ctx, &types.QueryDenomHaltedRequest{})
	suite.Require().Nil(resp)
	suite.Require().Equal(codes.InvalidArgument, status.Code(err))
	suite.Require().Contains(err.Error(), "denom is required")
}
