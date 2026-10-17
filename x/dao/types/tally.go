package types

import (
	"math/big"

	"cosmossdk.io/math"
)

// TallyPasses decides a proposal from its tally under the given bars, both
// compared with ≥:
//
//	quorum:  (yes + no + abstain) / denominator ≥ quorum
//	outcome: yes / (yes + no) ≥ threshold
//
// Each side is cross-multiplied into exact integers (a LegacyDec is an integer
// count of 10^-18), so no rounding happens and one tally can never be both
// passable and rejectable (D17). A proposal nobody voted yes or no on is
// rejected before any division (D16), as is one with a zero denominator.
func TallyPasses(quorum, threshold math.LegacyDec, denominator, yes, no, abstain math.Int) bool {
	if !denominator.IsPositive() {
		return false
	}

	decided := yes.Add(no)
	if !decided.IsPositive() {
		return false
	}

	return atLeast(decided.Add(abstain), denominator, quorum) && atLeast(yes, decided, threshold)
}

// ExpediteTallyPasses decides an expedited proposal before its voting end. Both
// bars are measured against the full denominator, compared with ≥:
//
//	quorum: (yes + no + abstain) / denominator ≥ quorum
//	yes:    yes / denominator ≥ threshold
//
// The expedite threshold is at least the regular one, so a proposal that meets
// these bars would still pass after its voting end if every remaining unit of
// voting power voted no (D7): early settlement only speeds up the outcome.
func ExpediteTallyPasses(quorum, threshold math.LegacyDec, denominator, yes, no, abstain math.Int) bool {
	if !denominator.IsPositive() || !yes.IsPositive() {
		return false
	}

	return atLeast(yes.Add(no).Add(abstain), denominator, quorum) && atLeast(yes, denominator, threshold)
}

// atLeast reports whether part / whole ≥ ratio, as part · 10^18 ≥ ratio · whole
// in big integers.
func atLeast(part, whole math.Int, ratio math.LegacyDec) bool {
	left := new(big.Int).Mul(part.BigInt(), math.LegacyOneDec().BigInt())
	right := new(big.Int).Mul(ratio.BigInt(), whole.BigInt())

	return left.Cmp(right) >= 0
}

// RegularTallyPasses decides a proposal at or after its voting end, under the
// pinned config's regular quorum and threshold.
func (p Proposal) RegularTallyPasses() bool {
	return TallyPasses(p.PinnedConfig.MinQuorum, p.PinnedConfig.Threshold, p.PinnedDenominator, p.Yes, p.No, p.Abstain)
}

// ExpediteTallyPasses decides an expedited proposal before its voting end,
// under the pinned config's expedite quorum and threshold.
func (p Proposal) ExpediteTallyPasses() bool {
	return ExpediteTallyPasses(p.PinnedConfig.ExpediteQuorum, p.PinnedConfig.ExpediteThreshold, p.PinnedDenominator, p.Yes, p.No, p.Abstain)
}
