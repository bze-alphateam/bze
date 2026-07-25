package keeper_test

import (
	"github.com/bze-alphateam/bze/x/rewards/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (suite *IntegrationTestSuite) TestQueryBoost_FoundAndNotFound() {
	suite.k.SetBoost(suite.ctx, newBoost("000000000001", "ubze", 1))

	resp, err := suite.k.Boost(suite.ctx, &types.QueryBoostRequest{RewardId: "000000000001", Denom: "ubze"})
	suite.Require().NoError(err)
	suite.Require().Equal("ubze", resp.Boost.Denom)
	suite.Require().Equal(uint64(1), resp.Boost.Uid)

	_, err = suite.k.Boost(suite.ctx, &types.QueryBoostRequest{RewardId: "000000000001", Denom: "missing"})
	suite.Require().Error(err)
	suite.Require().Equal(codes.NotFound, status.Code(err))
}

func (suite *IntegrationTestSuite) TestQueryBoost_NilRequest() {
	_, err := suite.k.Boost(suite.ctx, nil)
	suite.Require().Error(err)
	suite.Require().Equal(codes.InvalidArgument, status.Code(err))
}

// TestQueryRewardBoosts_WeirdDenoms verifies per-reward scanning works for denoms
// that contain "/" (IBC, factory) and isolates rewards from each other.
func (suite *IntegrationTestSuite) TestQueryRewardBoosts_WeirdDenoms() {
	ibcDenom := "ibc/27394FB092D2ECCD56123C74F36E4C1F926001CEADA9CA97EA622B25F41E5EB2"
	factoryDenom := "factory/bze1abcdefg/sub"

	suite.k.SetBoost(suite.ctx, newBoost("000000000001", "ubze", 1))
	suite.k.SetBoost(suite.ctx, newBoost("000000000001", ibcDenom, 2))
	suite.k.SetBoost(suite.ctx, newBoost("000000000001", factoryDenom, 3))
	suite.k.SetBoost(suite.ctx, newBoost("000000000002", "ubze", 4))

	resp, err := suite.k.RewardBoosts(suite.ctx, &types.QueryRewardBoostsRequest{RewardId: "000000000001"})
	suite.Require().NoError(err)
	suite.Require().Len(resp.List, 3)
	denoms := map[string]bool{}
	for _, b := range resp.List {
		suite.Require().Equal("000000000001", b.RewardId)
		denoms[b.Denom] = true
	}
	suite.Require().True(denoms["ubze"])
	suite.Require().True(denoms[ibcDenom])
	suite.Require().True(denoms[factoryDenom])

	// exact retrieval by weird denom
	single, err := suite.k.Boost(suite.ctx, &types.QueryBoostRequest{RewardId: "000000000001", Denom: ibcDenom})
	suite.Require().NoError(err)
	suite.Require().Equal(ibcDenom, single.Boost.Denom)

	// reward isolation
	resp2, err := suite.k.RewardBoosts(suite.ctx, &types.QueryRewardBoostsRequest{RewardId: "000000000002"})
	suite.Require().NoError(err)
	suite.Require().Len(resp2.List, 1)

	// reward with no boosts
	resp3, err := suite.k.RewardBoosts(suite.ctx, &types.QueryRewardBoostsRequest{RewardId: "000000000003"})
	suite.Require().NoError(err)
	suite.Require().Empty(resp3.List)
}

func (suite *IntegrationTestSuite) TestQueryAllBoosts_Pagination() {
	suite.k.SetBoost(suite.ctx, newBoost("000000000001", "ubze", 1))
	suite.k.SetBoost(suite.ctx, newBoost("000000000001", "uother", 2))
	suite.k.SetBoost(suite.ctx, newBoost("000000000002", "ubze", 3))

	// first page: limit 2
	page1, err := suite.k.AllBoosts(suite.ctx, &types.QueryAllBoostsRequest{
		Pagination: &query.PageRequest{Limit: 2},
	})
	suite.Require().NoError(err)
	suite.Require().Len(page1.List, 2)
	suite.Require().NotNil(page1.Pagination)
	suite.Require().NotNil(page1.Pagination.NextKey)

	// second page continues from NextKey
	page2, err := suite.k.AllBoosts(suite.ctx, &types.QueryAllBoostsRequest{
		Pagination: &query.PageRequest{Key: page1.Pagination.NextKey, Limit: 2},
	})
	suite.Require().NoError(err)
	suite.Require().Len(page2.List, 1)

	// unpaginated returns all three
	all, err := suite.k.AllBoosts(suite.ctx, &types.QueryAllBoostsRequest{})
	suite.Require().NoError(err)
	suite.Require().Len(all.List, 3)
}
