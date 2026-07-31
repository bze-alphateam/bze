package keeper_test

import (
	"github.com/bze-alphateam/bze/x/rewards/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (suite *IntegrationTestSuite) TestQueryBoost_Boost() {
	boost := newBoost("000000000001", "000000000002", "uboost")
	suite.k.SetBoost(suite.ctx, boost)

	req := &types.QueryBoostRequest{
		RewardId: "000000000001",
		BoostId:  "000000000002",
	}

	response, err := suite.k.Boost(suite.ctx, req)
	suite.Require().NoError(err)
	suite.Require().NotNil(response)
	suite.Require().Equal(boost, response.Boost)
}

func (suite *IntegrationTestSuite) TestQueryBoost_BoostNilRequest() {
	response, err := suite.k.Boost(suite.ctx, nil)

	suite.Require().Error(err)
	suite.Require().Nil(response)
	suite.Require().Equal(codes.InvalidArgument, status.Code(err))
}

func (suite *IntegrationTestSuite) TestQueryBoost_BoostNotFound() {
	req := &types.QueryBoostRequest{
		RewardId: "000000000001",
		BoostId:  "000000000009",
	}

	response, err := suite.k.Boost(suite.ctx, req)
	suite.Require().Error(err)
	suite.Require().Nil(response)
	suite.Require().Equal(codes.NotFound, status.Code(err))
}

// TestQueryBoost_RewardBoosts asserts per-reward isolation, including denoms
// containing "/" (IBC and factory denoms).
func (suite *IntegrationTestSuite) TestQueryBoost_RewardBoosts() {
	ibcDenom := "ibc/27394FB092D2ECCD56123C74F36E4C1F926001CEADA9CA97EA622B25F41E5EB2"
	factoryDenom := "factory/bze1abcdefg/sub"

	suite.k.SetBoost(suite.ctx, newBoost("000000000001", "000000000001", "ubze"))
	suite.k.SetBoost(suite.ctx, newBoost("000000000001", "000000000002", ibcDenom))
	suite.k.SetBoost(suite.ctx, newBoost("000000000001", "000000000003", factoryDenom))
	suite.k.SetBoost(suite.ctx, newBoost("000000000002", "000000000004", "ubze"))

	req := &types.QueryRewardBoostsRequest{RewardId: "000000000001"}

	response, err := suite.k.RewardBoosts(suite.ctx, req)
	suite.Require().NoError(err)
	suite.Require().NotNil(response)
	suite.Require().Len(response.List, 3)

	denoms := map[string]bool{}
	for _, b := range response.List {
		suite.Require().Equal("000000000001", b.RewardId)
		denoms[b.Denom] = true
	}
	suite.Require().True(denoms[ibcDenom])
	suite.Require().True(denoms[factoryDenom])

	//a reward without boosts returns an empty list
	response, err = suite.k.RewardBoosts(suite.ctx, &types.QueryRewardBoostsRequest{RewardId: "000000000003"})
	suite.Require().NoError(err)
	suite.Require().NotNil(response)
	suite.Require().Empty(response.List)
}

func (suite *IntegrationTestSuite) TestQueryBoost_RewardBoostsNilRequest() {
	response, err := suite.k.RewardBoosts(suite.ctx, nil)

	suite.Require().Error(err)
	suite.Require().Nil(response)
	suite.Require().Equal(codes.InvalidArgument, status.Code(err))
}

func (suite *IntegrationTestSuite) TestQueryBoost_AllBoosts() {
	suite.k.SetBoost(suite.ctx, newBoost("000000000001", "000000000001", "ubze"))
	suite.k.SetBoost(suite.ctx, newBoost("000000000001", "000000000002", "uboost"))
	suite.k.SetBoost(suite.ctx, newBoost("000000000002", "000000000003", "ubze"))

	req := &types.QueryAllBoostsRequest{}

	response, err := suite.k.AllBoosts(suite.ctx, req)
	suite.Require().NoError(err)
	suite.Require().NotNil(response)
	suite.Require().Len(response.List, 3)
}

// TestQueryBoost_AllBoostsPagination walks the list in pages of 2 and asserts
// the pages join up to the full set without overlap.
func (suite *IntegrationTestSuite) TestQueryBoost_AllBoostsPagination() {
	suite.k.SetBoost(suite.ctx, newBoost("000000000001", "000000000001", "ubze"))
	suite.k.SetBoost(suite.ctx, newBoost("000000000001", "000000000002", "uboost"))
	suite.k.SetBoost(suite.ctx, newBoost("000000000002", "000000000003", "ubze"))

	req := &types.QueryAllBoostsRequest{
		Pagination: &query.PageRequest{Limit: 2},
	}

	first, err := suite.k.AllBoosts(suite.ctx, req)
	suite.Require().NoError(err)
	suite.Require().NotNil(first)
	suite.Require().Len(first.List, 2)
	suite.Require().NotNil(first.Pagination)
	suite.Require().NotEmpty(first.Pagination.NextKey)

	req.Pagination = &query.PageRequest{Key: first.Pagination.NextKey, Limit: 2}
	second, err := suite.k.AllBoosts(suite.ctx, req)
	suite.Require().NoError(err)
	suite.Require().NotNil(second)
	suite.Require().Len(second.List, 1)

	seen := map[string]bool{}
	for _, b := range append(first.List, second.List...) {
		seen[b.Id] = true
	}
	suite.Require().Len(seen, 3)
}

func (suite *IntegrationTestSuite) TestQueryBoost_AllBoostsNilRequest() {
	response, err := suite.k.AllBoosts(suite.ctx, nil)

	suite.Require().Error(err)
	suite.Require().Nil(response)
	suite.Require().Equal(codes.InvalidArgument, status.Code(err))
}
