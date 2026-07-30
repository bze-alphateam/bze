package keeper_test

import (
	"github.com/bze-alphateam/bze/x/rewards/types"
)

func newBoost(rewardId, boostId, denom string) types.Boost {
	return types.Boost{
		Id:               boostId,
		RewardId:         rewardId,
		Denom:            denom,
		DailyAmount:      "1000",
		Duration:         30,
		Payouts:          0,
		DistributedStake: "0",
		Creator:          "bze1creator",
	}
}

func (suite *IntegrationTestSuite) TestBoost_SetGetRemove() {
	boost := newBoost("000000000001", "000000000001", "ubze")

	suite.k.SetBoost(suite.ctx, boost)

	got, found := suite.k.GetBoost(suite.ctx, "000000000001", "000000000001")
	suite.Require().True(found)
	suite.Require().Equal(boost, got)

	suite.k.RemoveBoost(suite.ctx, "000000000001", "000000000001")

	_, found = suite.k.GetBoost(suite.ctx, "000000000001", "000000000001")
	suite.Require().False(found)
}

func (suite *IntegrationTestSuite) TestBoost_GetNonExistent() {
	_, found := suite.k.GetBoost(suite.ctx, "000000000009", "000000000001")
	suite.Require().False(found)
}

// TestBoost_SameDenomMultiplicity verifies that two boosts of the same denom
// on one reward coexist — boosts are keyed by id, not denom.
func (suite *IntegrationTestSuite) TestBoost_SameDenomMultiplicity() {
	rewardId := "000000000001"

	suite.k.SetBoost(suite.ctx, newBoost(rewardId, "000000000001", "ubze"))
	suite.k.SetBoost(suite.ctx, newBoost(rewardId, "000000000002", "ubze"))

	first, found := suite.k.GetBoost(suite.ctx, rewardId, "000000000001")
	suite.Require().True(found)
	suite.Require().Equal("000000000001", first.Id)

	second, found := suite.k.GetBoost(suite.ctx, rewardId, "000000000002")
	suite.Require().True(found)
	suite.Require().Equal("000000000002", second.Id)

	suite.Require().Len(suite.k.GetRewardBoosts(suite.ctx, rewardId), 2)

	// removing one leaves the other untouched
	suite.k.RemoveBoost(suite.ctx, rewardId, "000000000001")
	suite.Require().Len(suite.k.GetRewardBoosts(suite.ctx, rewardId), 1)

	_, found = suite.k.GetBoost(suite.ctx, rewardId, "000000000002")
	suite.Require().True(found)
}

// TestBoost_GetRewardBoosts_PerRewardIteration verifies the per-reward prefix
// scan isolates a single reward's boosts, including boosts whose denom
// contains "/" (IBC and factory denoms).
func (suite *IntegrationTestSuite) TestBoost_GetRewardBoosts_PerRewardIteration() {
	reward1 := "000000000001"
	reward2 := "000000000002"

	ibcDenom := "ibc/27394FB092D2ECCD56123C74F36E4C1F926001CEADA9CA97EA622B25F41E5EB2"
	factoryDenom := "factory/bze1abcdefg/sub"

	// reward1: three boosts with plain, IBC and factory denoms
	suite.k.SetBoost(suite.ctx, newBoost(reward1, "000000000001", "ubze"))
	suite.k.SetBoost(suite.ctx, newBoost(reward1, "000000000002", ibcDenom))
	suite.k.SetBoost(suite.ctx, newBoost(reward1, "000000000003", factoryDenom))

	// reward2: one boost that must never leak into reward1's scan
	suite.k.SetBoost(suite.ctx, newBoost(reward2, "000000000004", "ubze"))

	got := suite.k.GetRewardBoosts(suite.ctx, reward1)
	suite.Require().Len(got, 3)

	denoms := map[string]bool{}
	for _, b := range got {
		suite.Require().Equal(reward1, b.RewardId)
		denoms[b.Denom] = true
	}
	suite.Require().True(denoms["ubze"])
	suite.Require().True(denoms[ibcDenom])
	suite.Require().True(denoms[factoryDenom])

	// reward2 scan returns only its own boost
	got2 := suite.k.GetRewardBoosts(suite.ctx, reward2)
	suite.Require().Len(got2, 1)
	suite.Require().Equal(reward2, got2[0].RewardId)
	suite.Require().Equal("000000000004", got2[0].Id)

	// a reward with no boosts scans empty
	suite.Require().Empty(suite.k.GetRewardBoosts(suite.ctx, "000000000003"))
}

func (suite *IntegrationTestSuite) TestBoost_GetAllBoosts() {
	suite.Require().Empty(suite.k.GetAllBoosts(suite.ctx))

	suite.k.SetBoost(suite.ctx, newBoost("000000000001", "000000000001", "ubze"))
	suite.k.SetBoost(suite.ctx, newBoost("000000000001", "000000000002", "uother"))
	suite.k.SetBoost(suite.ctx, newBoost("000000000002", "000000000003", "ubze"))

	all := suite.k.GetAllBoosts(suite.ctx)
	suite.Require().Len(all, 3)
}

func (suite *IntegrationTestSuite) TestBoost_Counter() {
	// starts at zero
	suite.Require().Equal(uint64(0), suite.k.GetBoostsCounter(suite.ctx))

	// monotonic reservation, returned in fixed-width store form
	suite.Require().Equal("000000000001", suite.k.ReserveBoostId(suite.ctx))
	suite.Require().Equal("000000000002", suite.k.ReserveBoostId(suite.ctx))
	suite.Require().Equal("000000000003", suite.k.ReserveBoostId(suite.ctx))
	suite.Require().Equal(uint64(3), suite.k.GetBoostsCounter(suite.ctx))

	// explicit set is honored (genesis import)
	suite.k.SetBoostsCounter(suite.ctx, 100)
	suite.Require().Equal(uint64(100), suite.k.GetBoostsCounter(suite.ctx))
	suite.Require().Equal("000000000101", suite.k.ReserveBoostId(suite.ctx))
}

// TestBoost_Counter_IndependentFromOthers guards the {3} counter key against
// collision with the staking ({1}) and trading ({2}) counters.
func (suite *IntegrationTestSuite) TestBoost_Counter_IndependentFromOthers() {
	suite.k.SetStakingRewardsCounter(suite.ctx, 5)
	suite.k.SetTradingRewardsCounter(suite.ctx, 9)

	suite.Require().Equal("000000000001", suite.k.ReserveBoostId(suite.ctx))

	suite.Require().Equal(uint64(5), suite.k.GetStakingRewardsCounter(suite.ctx))
	suite.Require().Equal(uint64(9), suite.k.GetTradingRewardsCounter(suite.ctx))
	suite.Require().Equal(uint64(1), suite.k.GetBoostsCounter(suite.ctx))
}
