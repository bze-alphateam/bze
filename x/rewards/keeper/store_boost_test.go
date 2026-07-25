package keeper_test

import (
	"github.com/bze-alphateam/bze/x/rewards/types"
)

func newBoost(rewardId, denom string, uid uint64) types.Boost {
	return types.Boost{
		RewardId:       rewardId,
		Denom:          denom,
		Uid:            uid,
		DailyAmount:    "1000",
		DaysLeft:       30,
		SBoost:         "0",
		FinalizeCursor: "",
		Creator:        "bze1creator",
	}
}

func (suite *IntegrationTestSuite) TestBoost_SetGetRemove() {
	boost := newBoost("000000000001", "ubze", 1)

	suite.k.SetBoost(suite.ctx, boost)

	got, found := suite.k.GetBoost(suite.ctx, "000000000001", "ubze")
	suite.Require().True(found)
	suite.Require().Equal(boost.RewardId, got.RewardId)
	suite.Require().Equal(boost.Denom, got.Denom)
	suite.Require().Equal(boost.Uid, got.Uid)
	suite.Require().Equal(boost.DailyAmount, got.DailyAmount)
	suite.Require().Equal(boost.DaysLeft, got.DaysLeft)

	suite.k.RemoveBoost(suite.ctx, "000000000001", "ubze")

	_, found = suite.k.GetBoost(suite.ctx, "000000000001", "ubze")
	suite.Require().False(found)
}

func (suite *IntegrationTestSuite) TestBoost_GetNonExistent() {
	_, found := suite.k.GetBoost(suite.ctx, "000000000009", "ubze")
	suite.Require().False(found)
}

// TestBoost_GetRewardBoosts_PerRewardIteration verifies the per-reward prefix
// scan isolates a single reward's boosts even when denoms contain "/" (IBC and
// factory denoms), which is why reward_id (fixed-width) must come first in the key.
func (suite *IntegrationTestSuite) TestBoost_GetRewardBoosts_PerRewardIteration() {
	reward1 := "000000000001"
	reward2 := "000000000002"

	ibcDenom := "ibc/27394FB092D2ECCD56123C74F36E4C1F926001CEADA9CA97EA622B25F41E5EB2"
	factoryDenom := "factory/bze1abcdefg/sub"

	// reward1: three boosts with plain, IBC and factory denoms
	suite.k.SetBoost(suite.ctx, newBoost(reward1, "ubze", 1))
	suite.k.SetBoost(suite.ctx, newBoost(reward1, ibcDenom, 2))
	suite.k.SetBoost(suite.ctx, newBoost(reward1, factoryDenom, 3))

	// reward2: one boost that must never leak into reward1's scan
	suite.k.SetBoost(suite.ctx, newBoost(reward2, "ubze", 4))

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
	suite.Require().Equal("ubze", got2[0].Denom)

	// a reward with no boosts scans empty
	suite.Require().Empty(suite.k.GetRewardBoosts(suite.ctx, "000000000003"))

	// the IBC and factory denom boosts are individually retrievable by exact key
	gotIbc, found := suite.k.GetBoost(suite.ctx, reward1, ibcDenom)
	suite.Require().True(found)
	suite.Require().Equal(ibcDenom, gotIbc.Denom)

	gotFactory, found := suite.k.GetBoost(suite.ctx, reward1, factoryDenom)
	suite.Require().True(found)
	suite.Require().Equal(factoryDenom, gotFactory.Denom)
}

func (suite *IntegrationTestSuite) TestBoost_GetAllBoosts() {
	suite.Require().Empty(suite.k.GetAllBoosts(suite.ctx))

	suite.k.SetBoost(suite.ctx, newBoost("000000000001", "ubze", 1))
	suite.k.SetBoost(suite.ctx, newBoost("000000000001", "uother", 2))
	suite.k.SetBoost(suite.ctx, newBoost("000000000002", "ubze", 3))

	all := suite.k.GetAllBoosts(suite.ctx)
	suite.Require().Len(all, 3)
}

func (suite *IntegrationTestSuite) TestBoost_UidCounter() {
	// starts at zero
	suite.Require().Equal(uint64(0), suite.k.GetBoostCounter(suite.ctx))

	// monotonic reservation
	suite.Require().Equal(uint64(1), suite.k.ReserveBoostUid(suite.ctx))
	suite.Require().Equal(uint64(2), suite.k.ReserveBoostUid(suite.ctx))
	suite.Require().Equal(uint64(3), suite.k.ReserveBoostUid(suite.ctx))
	suite.Require().Equal(uint64(3), suite.k.GetBoostCounter(suite.ctx))

	// explicit set is honored
	suite.k.SetBoostCounter(suite.ctx, 100)
	suite.Require().Equal(uint64(100), suite.k.GetBoostCounter(suite.ctx))
	suite.Require().Equal(uint64(101), suite.k.ReserveBoostUid(suite.ctx))
}

// TestBoost_UidCounter_IndependentFromOthers guards the {3} counter key against
// collision with the staking ({1}) and trading ({2}) counters.
func (suite *IntegrationTestSuite) TestBoost_UidCounter_IndependentFromOthers() {
	suite.k.SetStakingRewardsCounter(suite.ctx, 5)
	suite.k.SetTradingRewardsCounter(suite.ctx, 9)

	suite.Require().Equal(uint64(1), suite.k.ReserveBoostUid(suite.ctx))

	suite.Require().Equal(uint64(5), suite.k.GetStakingRewardsCounter(suite.ctx))
	suite.Require().Equal(uint64(9), suite.k.GetTradingRewardsCounter(suite.ctx))
}
