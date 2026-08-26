package keeper_test

import (
	"cosmossdk.io/math"

	"github.com/bze-alphateam/bze/x/rewards/types"
)

// Regression for BZE-104: the staking-reward accumulator must truncate S += r/T (round down), so it
// never credits more than the funded reward. Participants claim floor(deposited·(S-joinedAt)); if
// the accumulator rounded r/T up instead, a large staker's floor could exceed the escrowed r and
// pull the difference out of the shared module account. These tests pin the truncating behavior at
// the distribution boundary (no bank effects needed — the accumulator value is the invariant).

// processOneStakingReward seeds a single reward and runs one distribution pass over it, returning the
// stored DistributedStake (S) afterwards as a LegacyDec.
func (suite *IntegrationTestSuite) processOneStakingReward(sr types.StakingReward) math.LegacyDec {
	suite.k.SetStakingReward(suite.ctx, sr)
	suite.k.EnqueueStakingRewardsDistribution(suite.ctx)
	suite.k.ProcessStakingRewardsDistributionQueue(suite.ctx)

	updated, found := suite.k.GetStakingReward(suite.ctx, sr.RewardId)
	suite.Require().True(found)

	s, err := math.LegacyNewDecFromStr(updated.DistributedStake)
	suite.Require().NoError(err)

	return s
}

// TestDistributeStakingReward_TruncatesAccumulator: a repeating quotient (2/3) must land on the
// truncated ...666, not the round-to-nearest ...667. The extra ulp is exactly what would let summed
// claims drift above the funded reward.
func (suite *IntegrationTestSuite) TestDistributeStakingReward_TruncatesAccumulator() {
	s := suite.processOneStakingReward(types.StakingReward{
		RewardId:         "rounding-2-over-3",
		PrizeAmount:      "2",
		PrizeDenom:       "ubze",
		StakingDenom:     "ubze",
		Duration:         100,
		Payouts:          0,
		MinStake:         1,
		Lock:             0,
		StakedAmount:     "3",
		DistributedStake: "0",
	})

	suite.Require().Equal("0.666666666666666666", s.String())
}

// TestDistributeStakingReward_SubUlpFractionDropped: with a staked total on the order of 1e18 a tiny
// reward yields r/T below one ulp (1e-18). Truncation drops it to zero, so a staker holding the whole
// stake claims floor(deposited·S) = 0 ≤ r and nothing leaves the pool. Round-to-nearest would have
// bumped S to a full 1e-18 ulp, letting that staker claim more units than were funded.
func (suite *IntegrationTestSuite) TestDistributeStakingReward_SubUlpFractionDropped() {
	const stakedTotal = "3000000000000000000" // 3e18
	const prizeAmount = "2"

	s := suite.processOneStakingReward(types.StakingReward{
		RewardId:         "sub-ulp-fraction",
		PrizeAmount:      prizeAmount,
		PrizeDenom:       "ubze",
		StakingDenom:     "ubze",
		Duration:         100,
		Payouts:          0,
		MinStake:         1,
		Lock:             0,
		StakedAmount:     stakedTotal,
		DistributedStake: "0",
	})

	// r/T = 2/3e18 ≈ 6.6e-19 < 1e-18, so the accumulator gains nothing.
	suite.Require().True(s.IsZero(), "accumulator must stay zero, got %s", s)

	// Funds-conservation: the largest possible claimant (holding the entire stake) can claim no more
	// than the funded reward. floor(deposited·S) with S=0 is 0.
	deposited := math.LegacyNewDecFromInt(math.NewIntFromUint64(3000000000000000000))
	claimed := deposited.Mul(s).TruncateInt()
	funded, ok := math.NewIntFromString(prizeAmount)
	suite.Require().True(ok)
	suite.Require().True(claimed.LTE(funded), "claim %s must not exceed funded %s", claimed, funded)
}
