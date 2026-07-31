package keeper_test

import (
	"strings"

	"cosmossdk.io/math"
	"github.com/bze-alphateam/bze/x/rewards/types"
)

func (suite *IntegrationTestSuite) setBoostParent(rewardId, stakedAmount string, payouts, duration uint32) {
	suite.k.SetStakingReward(suite.ctx, types.StakingReward{
		RewardId:         rewardId,
		PrizeAmount:      "1000",
		PrizeDenom:       "ubze",
		StakingDenom:     "ubze",
		Duration:         duration,
		Payouts:          payouts,
		MinStake:         100,
		Lock:             7,
		StakedAmount:     stakedAmount,
		DistributedStake: "0",
	})
}

// distributeOneDay runs a full distribution pass (one "day") through the
// public queue entry points, exercising the distributeStakingReward insertion.
func (suite *IntegrationTestSuite) distributeOneDay() {
	suite.k.EnqueueStakingRewardsDistribution(suite.ctx)
	suite.k.ProcessStakingRewardsDistributionQueue(suite.ctx)
}

func (suite *IntegrationTestSuite) TestDistributeBoosts_AdvancesAccumulatorAndPayouts() {
	suite.setBoostParent("reward-1", "5000", 0, 10)

	// two same-denom boosts plus a different denom: all advance independently
	boosts := []types.Boost{
		{Id: "000000000001", RewardId: "reward-1", Denom: "ubze", DailyAmount: "1000", Duration: 5, Payouts: 0, DistributedStake: "0"},
		{Id: "000000000002", RewardId: "reward-1", Denom: "ubze", DailyAmount: "500", Duration: 5, Payouts: 0, DistributedStake: "0"},
		{Id: "000000000003", RewardId: "reward-1", Denom: "uvdl", DailyAmount: "300", Duration: 5, Payouts: 0, DistributedStake: "0"},
	}
	for _, b := range boosts {
		suite.k.SetBoost(suite.ctx, b)
	}

	suite.distributeOneDay()

	staked := math.LegacyNewDec(5000)
	for _, seeded := range boosts {
		boost, found := suite.k.GetBoost(suite.ctx, "reward-1", seeded.Id)
		suite.Require().True(found)
		suite.Require().Equal(uint32(1), boost.Payouts)

		daily := math.LegacyMustNewDecFromStr(seeded.DailyAmount)
		suite.Require().Equal(daily.Quo(staked).String(), boost.DistributedStake)

		// I1 spot-check: the day's accumulator delta times T emits exactly daily_amount
		delta := math.LegacyMustNewDecFromStr(boost.DistributedStake)
		suite.Require().Equal(daily.String(), delta.Mul(staked).String())
	}

	// the base reward distributed as before
	sr, found := suite.k.GetStakingReward(suite.ctx, "reward-1")
	suite.Require().True(found)
	suite.Require().Equal(uint32(1), sr.Payouts)
}

func (suite *IntegrationTestSuite) TestDistributeBoosts_MultipleDaysVaryingT() {
	suite.setBoostParent("reward-1", "5000", 0, 10)
	suite.k.SetBoost(suite.ctx, types.Boost{
		Id: "000000000001", RewardId: "reward-1", Denom: "ubze", DailyAmount: "1000", Duration: 5, Payouts: 0, DistributedStake: "0",
	})

	suite.distributeOneDay()

	// stake changes between days (join/exit): day 2 divides by the live T
	sr, found := suite.k.GetStakingReward(suite.ctx, "reward-1")
	suite.Require().True(found)
	sr.StakedAmount = "3000"
	suite.k.SetStakingReward(suite.ctx, sr)

	suite.distributeOneDay()

	daily := math.LegacyNewDec(1000)
	expected := daily.Quo(math.LegacyNewDec(5000)).Add(daily.Quo(math.LegacyNewDec(3000)))

	boost, found := suite.k.GetBoost(suite.ctx, "reward-1", "000000000001")
	suite.Require().True(found)
	suite.Require().Equal(uint32(2), boost.Payouts)
	suite.Require().Equal(expected.String(), boost.DistributedStake)
}

func (suite *IntegrationTestSuite) TestDistributeBoosts_SkipParity_ZeroStakers() {
	// A6 regression: zero-staker epoch mid-boost must not advance the boost or panic
	suite.setBoostParent("reward-1", "0", 1, 10)
	suite.k.SetBoost(suite.ctx, types.Boost{
		Id: "000000000001", RewardId: "reward-1", Denom: "ubze", DailyAmount: "1000", Duration: 5, Payouts: 1, DistributedStake: "0.2",
	})

	suite.Require().NotPanics(func() {
		suite.distributeOneDay()
	})

	boost, found := suite.k.GetBoost(suite.ctx, "reward-1", "000000000001")
	suite.Require().True(found)
	suite.Require().Equal(uint32(1), boost.Payouts)
	suite.Require().Equal("0.2", boost.DistributedStake)

	sr, found := suite.k.GetStakingReward(suite.ctx, "reward-1")
	suite.Require().True(found)
	suite.Require().Equal(uint32(1), sr.Payouts)
}

func (suite *IntegrationTestSuite) TestDistributeBoosts_SkipParity_FinishedParent() {
	// A6 regression: finished parent skips distribution, boosts stay untouched
	suite.setBoostParent("reward-1", "5000", 5, 5)
	suite.k.SetBoost(suite.ctx, types.Boost{
		Id: "000000000001", RewardId: "reward-1", Denom: "ubze", DailyAmount: "1000", Duration: 3, Payouts: 2, DistributedStake: "0.4",
	})

	suite.distributeOneDay()

	boost, found := suite.k.GetBoost(suite.ctx, "reward-1", "000000000001")
	suite.Require().True(found)
	suite.Require().Equal(uint32(2), boost.Payouts)
	suite.Require().Equal("0.4", boost.DistributedStake)

	sr, found := suite.k.GetStakingReward(suite.ctx, "reward-1")
	suite.Require().True(found)
	suite.Require().Equal(uint32(5), sr.Payouts)
}

func (suite *IntegrationTestSuite) TestDistributeBoosts_FinishedBoostSkippedButKept() {
	suite.setBoostParent("reward-1", "5000", 0, 10)

	// boost-1 finishes on the first pass (duration 1); boost-2 keeps accruing
	suite.k.SetBoost(suite.ctx, types.Boost{
		Id: "000000000001", RewardId: "reward-1", Denom: "ubze", DailyAmount: "1000", Duration: 1, Payouts: 0, DistributedStake: "0",
	})
	suite.k.SetBoost(suite.ctx, types.Boost{
		Id: "000000000002", RewardId: "reward-1", Denom: "ubze", DailyAmount: "500", Duration: 5, Payouts: 0, DistributedStake: "0",
	})

	suite.distributeOneDay()

	finishedStake := math.LegacyNewDec(1000).Quo(math.LegacyNewDec(5000)).String()
	boost1, found := suite.k.GetBoost(suite.ctx, "reward-1", "000000000001")
	suite.Require().True(found)
	suite.Require().Equal(uint32(1), boost1.Payouts)
	suite.Require().Equal(finishedStake, boost1.DistributedStake)

	suite.distributeOneDay()

	// finished boost stays in store with its final accumulator (claim-servable), untouched by later passes
	boost1, found = suite.k.GetBoost(suite.ctx, "reward-1", "000000000001")
	suite.Require().True(found)
	suite.Require().Equal(uint32(1), boost1.Payouts)
	suite.Require().Equal(finishedStake, boost1.DistributedStake)

	// the unfinished boost kept advancing both days
	expected2 := math.LegacyNewDec(500).Quo(math.LegacyNewDec(5000)).MulInt64(2).String()
	boost2, found := suite.k.GetBoost(suite.ctx, "reward-1", "000000000002")
	suite.Require().True(found)
	suite.Require().Equal(uint32(2), boost2.Payouts)
	suite.Require().Equal(expected2, boost2.DistributedStake)
}

func (suite *IntegrationTestSuite) TestDistributeBoosts_EmitsDistributionEvent() {
	suite.setBoostParent("reward-1", "5000", 0, 10)
	suite.k.SetBoost(suite.ctx, types.Boost{
		Id: "000000000001", RewardId: "reward-1", Denom: "ubze", DailyAmount: "1000", Duration: 5, Payouts: 0, DistributedStake: "0",
	})
	suite.k.SetBoost(suite.ctx, types.Boost{
		Id: "000000000002", RewardId: "reward-1", Denom: "uvdl", DailyAmount: "300", Duration: 5, Payouts: 0, DistributedStake: "0",
	})

	suite.distributeOneDay()

	distributionEvents := 0
	for _, event := range suite.ctx.EventManager().Events() {
		if !strings.Contains(event.Type, "BoostDistributionEvent") {
			continue
		}
		distributionEvents++

		attrs := map[string]string{}
		for _, attr := range event.Attributes {
			attrs[string(attr.Key)] = string(attr.Value)
		}
		suite.Require().Contains(attrs["reward_id"], "reward-1")
		switch {
		case strings.Contains(attrs["boost_id"], "000000000001"):
			suite.Require().Contains(attrs["amount"], "1000")
			suite.Require().Contains(attrs["denom"], "ubze")
		case strings.Contains(attrs["boost_id"], "000000000002"):
			suite.Require().Contains(attrs["amount"], "300")
			suite.Require().Contains(attrs["denom"], "uvdl")
		default:
			suite.Fail("unexpected boost_id in BoostDistributionEvent", attrs["boost_id"])
		}
	}
	suite.Require().Equal(2, distributionEvents)
}

func (suite *IntegrationTestSuite) TestDistributeBoosts_NoBoosts() {
	// base distribution is unchanged when the reward has no boosts
	suite.setBoostParent("reward-1", "5000", 0, 10)

	suite.Require().NotPanics(func() {
		suite.distributeOneDay()
	})

	sr, found := suite.k.GetStakingReward(suite.ctx, "reward-1")
	suite.Require().True(found)
	suite.Require().Equal(uint32(1), sr.Payouts)
}
