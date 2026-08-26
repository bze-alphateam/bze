package types

import (
	"testing"

	"cosmossdk.io/math"
	"github.com/stretchr/testify/require"
)

// The guards reject a zero/negative stake total (invariant I6) and a non-positive amount.
func TestValidateDenomDistribution_Guards(t *testing.T) {
	cases := []struct {
		name        string
		amount      int64
		stakedTotal int64
		wantErr     string
	}{
		{"zero T", 1000, 0, "no stakers found in denom reward ubze"},
		{"negative T", 1000, -5, "no stakers found in denom reward ubze"},
		{"zero amount", 0, 100, "distribution amount should be positive"},
		{"negative amount", -10, 100, "distribution amount should be positive"},
		{"zero T wins over zero amount", 0, 0, "no stakers found in denom reward ubze"},
	}

	for _, tc := range cases {
		err := ValidateDenomDistribution("ubze", math.NewInt(tc.amount), math.NewInt(tc.stakedTotal))
		require.EqualError(t, err, tc.wantErr, tc.name)
	}

	require.NoError(t, ValidateDenomDistribution("ubze", math.NewInt(1), math.NewInt(1)))
}

// The accumulator bump matches hand-computed S += amount/T across exact, fractional and repeating
// divisions, accumulates onto an existing S, stamps the epoch, and leaves the receiver untouched.
//
// The bump must TRUNCATE the quotient, never round to nearest, so the accumulator never credits
// more than the true amount/T and summed claims stay within the escrowed amount (BZE-104). The
// "truncates, never rounds up" case pins this: 2/3 must land on ...666, not the round-to-nearest
// ...667.
func TestDenomRewardPrize_WithDistribution_Math(t *testing.T) {
	cases := []struct {
		name        string
		startS      string
		amount      int64
		stakedTotal int64
		expectedS   string
	}{
		{"exact integer division", "0", 1000, 4, "250"},
		{"exact fractional division", "0", 1, 8, "0.125"},
		{"repeating decimal, 18dp", "0", 1000, 3, "333.333333333333333333"},
		{"truncates, never rounds up", "0", 2, 3, "0.666666666666666666"},
		{"accumulates onto existing S", "125", 500, 4, "250"},
	}

	for _, tc := range cases {
		prize := DenomRewardPrize{StakingDenom: "ubze", PrizeDenom: "uprize", DistributedStake: math.LegacyMustNewDecFromStr(tc.startS)}

		got := prize.WithDistribution(math.NewInt(tc.amount), math.NewInt(tc.stakedTotal), 42)

		require.True(t, math.LegacyMustNewDecFromStr(tc.expectedS).Equal(got.DistributedStake), "%s: expected S=%s got S=%s", tc.name, tc.expectedS, got.DistributedStake)
		require.Equal(t, int64(42), got.LastDistributionEpoch, tc.name)
		require.Equal(t, "ubze", got.StakingDenom, tc.name)
		require.Equal(t, "uprize", got.PrizeDenom, tc.name)

		// value receiver: the original is untouched
		require.True(t, math.LegacyMustNewDecFromStr(tc.startS).Equal(prize.DistributedStake), tc.name)
		require.Equal(t, int64(0), prize.LastDistributionEpoch, tc.name)
	}
}
