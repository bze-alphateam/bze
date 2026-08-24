package types

import (
	"fmt"

	"cosmossdk.io/math"
)

// ValidateDenomDistribution holds the guards every distribution into a prize accumulator must pass
// (security invariant I6): the live staked total must be positive — with T = 0 the payout would
// credit nobody and strand escrow — and the amount must be positive. Pure: it reads no state.
func ValidateDenomDistribution(stakingDenom string, amount, stakedTotal math.Int) error {
	if !stakedTotal.IsPositive() {
		return fmt.Errorf("no stakers found in denom reward %s", stakingDenom)
	}

	if !amount.IsPositive() {
		return fmt.Errorf("distribution amount should be positive")
	}

	return nil
}

// WithDistribution returns the accumulator bumped by amount/T with the distribution epoch stamped:
// S = S + amount / T. The accumulator only ever grows; T is the live staked total at the moment of
// distribution. Pure — callers are expected to have run ValidateDenomDistribution first.
func (p DenomRewardPrize) WithDistribution(amount, stakedTotal math.Int, epoch int64) DenomRewardPrize {
	p.DistributedStake = p.DistributedStake.Add(math.LegacyNewDecFromInt(amount).Quo(math.LegacyNewDecFromInt(stakedTotal)))
	p.LastDistributionEpoch = epoch

	return p
}
