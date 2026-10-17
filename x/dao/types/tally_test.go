package types_test

import (
	"testing"

	"cosmossdk.io/math"
	"github.com/stretchr/testify/require"

	"github.com/bze-alphateam/bze/x/dao/types"
)

func passes(quorum, threshold math.LegacyDec, denominator, yes, no, abstain int64) bool {
	return types.TallyPasses(quorum, threshold, math.NewInt(denominator), math.NewInt(yes), math.NewInt(no), math.NewInt(abstain))
}

func half() math.LegacyDec { return math.LegacyMustNewDecFromStr("0.5") }

// D16: a tally with no yes and no no votes is rejected before any division,
// whatever the abstentions; so is an empty denominator.
func TestTallyPasses_D16_NoDecidedVotes(t *testing.T) {
	require.False(t, passes(half(), half(), 10, 0, 0, 10))
	require.False(t, passes(half(), half(), 10, 0, 0, 0))
	require.False(t, passes(half(), half(), 0, 0, 0, 0))
	require.False(t, passes(math.LegacyZeroDec(), math.LegacyZeroDec(), 0, 0, 0, 0))
	require.False(t, passes(half(), half(), 0, 3, 0, 0))
}

// D17: the DAO DAO boundary suite. A ratio that is not a finite decimal (7/13)
// is truncated to 18 decimals, as DAO DAO's Decimal::from_ratio does, and the
// exact cross-multiplication then lets exactly 7 of 13 pass and 6 fail.
func TestTallyPasses_D17_Boundaries(t *testing.T) {
	sevenThirteenths := math.LegacyNewDec(7).QuoTruncate(math.LegacyNewDec(13))
	one := math.LegacyOneDec()

	require.True(t, passes(sevenThirteenths, sevenThirteenths, 13, 7, 6, 0))
	require.False(t, passes(sevenThirteenths, sevenThirteenths, 13, 6, 7, 0), "one below the threshold")
	require.False(t, passes(sevenThirteenths, half(), 13, 6, 0, 0), "one below the quorum")
	require.True(t, passes(one, one, 1, 1, 0, 0), "1 of 1")
	require.False(t, passes(one, one, 1, 0, 1, 0))
	require.False(t, passes(one, one, 2, 1, 0, 0), "1 of 2 misses a full quorum")

	for count := int64(1); count <= 50_000; count++ {
		if !passes(half(), half(), 2*count, count, count, 0) {
			t.Fatalf("%d yes of %d against 50%% must pass", count, 2*count)
		}
	}
}

// The quorum counts every vote, abstentions included, against the denominator;
// the threshold counts yes against yes + no. Both are met at equality.
func TestTallyPasses_QuorumThresholdMatrix(t *testing.T) {
	quorum, threshold := math.LegacyMustNewDecFromStr("0.4"), math.LegacyMustNewDecFromStr("0.6")
	for _, tc := range []struct {
		desc                          string
		denominator, yes, no, abstain int64
		want                          bool
	}{
		{"both bars met exactly", 100, 24, 16, 0, true},
		{"quorum one short", 100, 23, 16, 0, false},
		{"threshold one short", 100, 23, 17, 0, false},
		{"abstain fills the quorum", 100, 3, 2, 35, true},
		{"abstain does not move the threshold", 100, 2, 3, 95, false},
		{"unanimous over quorum", 100, 40, 0, 0, true},
		{"unanimous under quorum", 100, 39, 0, 0, false},
		{"full turnout, all no", 100, 0, 100, 0, false},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			require.Equal(t, tc.want, passes(quorum, threshold, tc.denominator, tc.yes, tc.no, tc.abstain))
		})
	}
}

// Stake tallies reach 10^30 and beyond; the cross-multiplication never overflows.
func TestTallyPasses_LargeStake(t *testing.T) {
	big, _ := math.NewIntFromString("1000000000000000000000000000000000000")
	yes := big.QuoRaw(2)
	require.True(t, types.TallyPasses(half(), half(), big, yes, yes, math.ZeroInt()))
	require.False(t, types.TallyPasses(half(), half(), big, yes.SubRaw(1), yes, math.ZeroInt()))
}
