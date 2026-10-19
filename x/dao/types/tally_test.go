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
	require.True(t, passes(one, half(), 1, 1, 0, 0), "1 of 1")
	require.False(t, passes(one, half(), 1, 0, 1, 0))
	require.False(t, passes(one, half(), 2, 1, 0, 0), "1 of 2 misses a full quorum")

	for count := int64(1); count <= 50_000; count++ {
		if !passes(half(), half(), 2*count+1, count+1, count, 0) {
			t.Fatalf("%d yes against %d no at 50%% must pass", count+1, count)
		}
		if passes(half(), half(), 2*count, count, count, 0) {
			t.Fatalf("a %d/%d tie at 50%% must not pass", count, count)
		}
	}
}

// The threshold is strict, as in x/gov: a tie never passes at 0.5, and yes
// exactly at the threshold fails while one unit more passes.
func TestTallyPasses_ThresholdIsStrict(t *testing.T) {
	require.False(t, passes(half(), half(), 2, 1, 1, 0), "1/1 tie")
	require.False(t, passes(half(), half(), 100, 50, 50, 0), "50/50 tie")
	require.True(t, passes(half(), half(), 100, 51, 49, 0), "51/49")

	sixty := math.LegacyMustNewDecFromStr("0.6")
	require.False(t, passes(half(), sixty, 10, 6, 4, 0), "yes exactly at 60%")
	require.True(t, passes(half(), sixty, 100, 61, 39, 0), "yes one unit above 60%")
}

// The quorum counts every vote, abstentions included, against the denominator
// and is met at equality; the threshold counts yes against yes + no and must
// be exceeded.
func TestTallyPasses_QuorumThresholdMatrix(t *testing.T) {
	quorum, threshold := math.LegacyMustNewDecFromStr("0.4"), math.LegacyMustNewDecFromStr("0.6")
	for _, tc := range []struct {
		desc                          string
		denominator, yes, no, abstain int64
		want                          bool
	}{
		{"quorum met exactly, threshold exceeded", 100, 25, 15, 0, true},
		{"threshold met exactly", 100, 24, 16, 0, false},
		{"quorum one short", 100, 24, 15, 0, false},
		{"abstain fills the quorum", 100, 4, 2, 34, true},
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
	require.True(t, types.TallyPasses(half(), half(), big, yes.AddRaw(1), yes.SubRaw(1), math.ZeroInt()))
	require.False(t, types.TallyPasses(half(), half(), big, yes, yes, math.ZeroInt()))
}

func expeditePasses(quorum, threshold math.LegacyDec, denominator, yes, no, abstain int64) bool {
	return types.ExpediteTallyPasses(quorum, threshold, math.NewInt(denominator), math.NewInt(yes), math.NewInt(no), math.NewInt(abstain))
}

// The expedite yes bar is measured against the full denominator, not yes + no,
// and must be exceeded; the quorum is met at equality.
func TestExpediteTallyPasses_Bars(t *testing.T) {
	quorum, threshold := math.LegacyMustNewDecFromStr("0.6"), math.LegacyMustNewDecFromStr("0.66")
	for _, tc := range []struct {
		desc                          string
		denominator, yes, no, abstain int64
		want                          bool
	}{
		{"yes one unit above the bar", 150, 100, 0, 0, true},
		{"yes exactly at the bar", 150, 99, 0, 0, false},
		{"no votes do not lower the yes bar", 150, 99, 51, 0, false},
		{"no votes do not block a met yes bar", 150, 100, 50, 0, true},
		{"empty denominator", 0, 0, 0, 0, false},
		{"nobody voted", 150, 0, 0, 0, false},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			require.Equal(t, tc.want, expeditePasses(quorum, threshold, tc.denominator, tc.yes, tc.no, tc.abstain))
		})
	}

	highQuorum := math.LegacyMustNewDecFromStr("0.9")
	require.False(t, expeditePasses(highQuorum, threshold, 150, 100, 0, 34), "quorum one short")
	require.True(t, expeditePasses(highQuorum, threshold, 150, 100, 0, 35), "abstain fills the quorum")
}

// D7: whenever the expedite bars are met, the regular bars still pass after
// every remaining unit of voting power votes no, for any expedite config at or
// above the regular one.
func TestExpediteTallyPasses_D7_NoUnitCanOverturn(t *testing.T) {
	configs := []struct{ quorum, threshold, expediteQuorum, expediteThreshold string }{
		{"0.4", "0.5", "0.6", "0.66"},
		{"0.5", "0.5", "0.5", "0.5"},
		{"0.1", "0.9", "0.1", "0.9"},
		{"1", "0.99", "1", "0.99"},
	}
	for _, c := range configs {
		quorum, threshold := math.LegacyMustNewDecFromStr(c.quorum), math.LegacyMustNewDecFromStr(c.threshold)
		expediteQuorum, expediteThreshold := math.LegacyMustNewDecFromStr(c.expediteQuorum), math.LegacyMustNewDecFromStr(c.expediteThreshold)
		for denominator := int64(1); denominator <= 40; denominator++ {
			for yes := int64(0); yes <= denominator; yes++ {
				for no := int64(0); yes+no <= denominator; no++ {
					for abstain := int64(0); yes+no+abstain <= denominator; abstain++ {
						if !expeditePasses(expediteQuorum, expediteThreshold, denominator, yes, no, abstain) {
							continue
						}
						rest := denominator - yes - no - abstain
						if !passes(quorum, threshold, denominator, yes, no+rest, abstain) {
							t.Fatalf("%+v: D %d, %d/%d/%d passed early but %d more no votes reject it", c, denominator, yes, no, abstain, rest)
						}
					}
				}
			}
		}
	}
}

// Stake tallies reach 10^30 and beyond; the expedite comparison never overflows.
func TestExpediteTallyPasses_LargeStake(t *testing.T) {
	big, _ := math.NewIntFromString("1000000000000000000000000000000000000")
	yes := big.QuoRaw(2)
	require.True(t, types.ExpediteTallyPasses(half(), half(), big, yes.AddRaw(1), math.ZeroInt(), math.ZeroInt()))
	require.False(t, types.ExpediteTallyPasses(half(), half(), big, yes, math.ZeroInt(), math.ZeroInt()))
}
