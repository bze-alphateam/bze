package keeper_test

import (
	"cosmossdk.io/math"
	"github.com/bze-alphateam/bze/x/rewards/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (suite *IntegrationTestSuite) setupBoostReward(rewardId, stakedAmount string, payouts, duration uint32) {
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

func (suite *IntegrationTestSuite) runDistribution() {
	suite.k.EnqueueStakingRewardsDistribution(suite.ctx)
	suite.k.ProcessStakingRewardsDistributionQueue(suite.ctx)
}

func (suite *IntegrationTestSuite) boostDistributionEvents() (events []sdk.Event) {
	for _, ev := range suite.ctx.EventManager().Events() {
		if ev.Type == "bze.rewards.BoostDistributionEvent" {
			events = append(events, ev)
		}
	}
	return
}

func (suite *IntegrationTestSuite) TestDistributeBoosts_AccumulatorMath() {
	suite.setupBoostReward("reward-1", "5000", 0, 5)
	suite.k.SetBoost(suite.ctx, types.Boost{
		RewardId:    "reward-1",
		Denom:       "uvdl",
		Uid:         1,
		DailyAmount: "700",
		DaysLeft:    3,
		SBoost:      "0",
	})
	suite.k.SetBoost(suite.ctx, types.Boost{
		RewardId:    "reward-1",
		Denom:       "stone",
		Uid:         2,
		DailyAmount: "250",
		DaysLeft:    2,
		SBoost:      "0",
	})

	suite.runDistribution()

	t := math.LegacyNewDec(5000)

	b1, found := suite.k.GetBoost(suite.ctx, "reward-1", "uvdl")
	suite.Require().True(found)
	suite.Require().Equal(uint32(2), b1.DaysLeft)
	suite.Require().Equal(math.LegacyNewDec(700).Quo(t).String(), b1.SBoost)

	b2, found := suite.k.GetBoost(suite.ctx, "reward-1", "stone")
	suite.Require().True(found)
	suite.Require().Equal(uint32(1), b2.DaysLeft)
	suite.Require().Equal(math.LegacyNewDec(250).Quo(t).String(), b2.SBoost)

	// I1 spot-check: the per-day accumulator delta times T equals daily_amount
	delta1 := math.LegacyMustNewDecFromStr(b1.SBoost)
	suite.Require().Equal(math.LegacyNewDec(700).String(), delta1.Mul(t).String())

	// one event per boost, carrying the daily amount
	events := suite.boostDistributionEvents()
	suite.Require().Len(events, 2)
}

func (suite *IntegrationTestSuite) TestDistributeBoosts_MultipleDaysVaryingT() {
	suite.setupBoostReward("reward-1", "5000", 0, 5)
	suite.k.SetBoost(suite.ctx, types.Boost{
		RewardId:    "reward-1",
		Denom:       "uvdl",
		Uid:         1,
		DailyAmount: "700",
		DaysLeft:    3,
		SBoost:      "0",
	})

	// day 1: T = 5000
	suite.runDistribution()

	// stake changes between days (joins/exits): day 2 runs with T = 10000
	sr, found := suite.k.GetStakingReward(suite.ctx, "reward-1")
	suite.Require().True(found)
	sr.StakedAmount = "10000"
	suite.k.SetStakingReward(suite.ctx, sr)

	suite.runDistribution()

	expected := math.LegacyNewDec(700).Quo(math.LegacyNewDec(5000)).
		Add(math.LegacyNewDec(700).Quo(math.LegacyNewDec(10000)))

	b, found := suite.k.GetBoost(suite.ctx, "reward-1", "uvdl")
	suite.Require().True(found)
	suite.Require().Equal(uint32(1), b.DaysLeft)
	suite.Require().Equal(expected.String(), b.SBoost)
}

func (suite *IntegrationTestSuite) TestDistributeBoosts_SkipParityZeroStaked() {
	// A6 regression: zero-staker epoch mid-boost must not advance nor consume days
	suite.setupBoostReward("reward-1", "0", 1, 5)
	suite.k.SetBoost(suite.ctx, types.Boost{
		RewardId:    "reward-1",
		Denom:       "uvdl",
		Uid:         1,
		DailyAmount: "700",
		DaysLeft:    3,
		SBoost:      "0.5",
	})

	suite.Require().NotPanics(func() {
		suite.runDistribution()
	})

	b, found := suite.k.GetBoost(suite.ctx, "reward-1", "uvdl")
	suite.Require().True(found)
	suite.Require().Equal(uint32(3), b.DaysLeft)
	suite.Require().Equal("0.5", b.SBoost)
	suite.Require().Empty(suite.boostDistributionEvents())

	// base reward untouched too
	sr, found := suite.k.GetStakingReward(suite.ctx, "reward-1")
	suite.Require().True(found)
	suite.Require().Equal(uint32(1), sr.Payouts)
}

func (suite *IntegrationTestSuite) TestDistributeBoosts_SkipParityFinishedReward() {
	// A6 regression: finished reward (payouts == duration) leaves boosts untouched
	suite.setupBoostReward("reward-1", "5000", 5, 5)
	suite.k.SetBoost(suite.ctx, types.Boost{
		RewardId:    "reward-1",
		Denom:       "uvdl",
		Uid:         1,
		DailyAmount: "700",
		DaysLeft:    2,
		SBoost:      "0.25",
	})

	suite.runDistribution()

	b, found := suite.k.GetBoost(suite.ctx, "reward-1", "uvdl")
	suite.Require().True(found)
	suite.Require().Equal(uint32(2), b.DaysLeft)
	suite.Require().Equal("0.25", b.SBoost)
	suite.Require().Empty(suite.boostDistributionEvents())
}

func (suite *IntegrationTestSuite) TestDistributeBoosts_FinishedBoostStaysAndIsSkipped() {
	suite.setupBoostReward("reward-1", "5000", 0, 5)
	suite.k.SetBoost(suite.ctx, types.Boost{
		RewardId:    "reward-1",
		Denom:       "uvdl",
		Uid:         1,
		DailyAmount: "700",
		DaysLeft:    1,
		SBoost:      "0",
	})

	// last emission day: days_left 1 -> 0
	suite.runDistribution()

	b, found := suite.k.GetBoost(suite.ctx, "reward-1", "uvdl")
	suite.Require().True(found)
	suite.Require().Equal(uint32(0), b.DaysLeft)
	expected := math.LegacyNewDec(700).Quo(math.LegacyNewDec(5000))
	suite.Require().Equal(expected.String(), b.SBoost)

	// subsequent pass: boost stays in store (finalizing) and is not advanced
	suite.runDistribution()

	b, found = suite.k.GetBoost(suite.ctx, "reward-1", "uvdl")
	suite.Require().True(found)
	suite.Require().Equal(uint32(0), b.DaysLeft)
	suite.Require().Equal(expected.String(), b.SBoost)
	suite.Require().Len(suite.boostDistributionEvents(), 1)

	// base reward kept distributing both days
	sr, found := suite.k.GetStakingReward(suite.ctx, "reward-1")
	suite.Require().True(found)
	suite.Require().Equal(uint32(2), sr.Payouts)
}

func (suite *IntegrationTestSuite) TestDistributeBoosts_RewardWithoutBoosts() {
	suite.setupBoostReward("reward-1", "5000", 0, 5)

	suite.Require().NotPanics(func() {
		suite.runDistribution()
	})

	sr, found := suite.k.GetStakingReward(suite.ctx, "reward-1")
	suite.Require().True(found)
	suite.Require().Equal(uint32(1), sr.Payouts)
	suite.Require().Empty(suite.boostDistributionEvents())
}
