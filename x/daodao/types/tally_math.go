package types

import (
	"math/bits"
)

// BpsDenominator is the basis-points scale: thresholds / quorums are
// expressed against 10_000 (so 5_000 = 50%).
const BpsDenominator = uint64(10_000)

// MulGreaterOrEqual returns whether a*b >= c*d, computed in 128 bits to
// avoid uint64 overflow at near-MaxUint64 powers. STATIC member weights,
// REWARD_STAKED participant amounts, and their sums are all uint64-bounded
// — but a*BpsDenominator overflows once power > MaxUint64 / 10_000.
// An admin with a 2^60 voting weight could otherwise construct an input
// that flips pass/reject under naive uint64 math.
//
// math/bits.Mul64 compiles to a single MUL on amd64 / arm64 and exposes
// both halves of the 128-bit product. Cost is one MUL plus a comparison.
func MulGreaterOrEqual(a, b, c, d uint64) bool {
	aHi, aLo := bits.Mul64(a, b)
	cHi, cLo := bits.Mul64(c, d)
	if aHi != cHi {
		return aHi > cHi
	}
	return aLo >= cLo
}

// MulLess is the negation of MulGreaterOrEqual: `a*b < c*d` in 128 bits.
func MulLess(a, b, c, d uint64) bool {
	return !MulGreaterOrEqual(a, b, c, d)
}

// TallyOutcome is the final disposition of a closed proposal — either
// pass (PASSED / EXECUTED) or reject (REJECTED). Pure-function output;
// the keeper translates this into a ProposalStatus when persisting.
type TallyOutcome int

const (
	OutcomeReject TallyOutcome = iota
	OutcomePass
)

// ComputeOutcome returns the final disposition of a closed proposal under
// the proposal's frozen GovernanceConfig. Pure function — no chain state.
// Used by:
//   - keeper end-blocker / early-close to decide PASSED vs REJECTED.
//   - genesis Validate to verify terminal-status proposals' status is
//     actually justified by their stored tally (Finding 1 fix).
//
// All basis-point comparisons go through MulGreaterOrEqual / MulLess so
// 128-bit arithmetic prevents uint64 overflow at adversarial powers.
func ComputeOutcome(g GovernanceConfig, t Tally) TallyOutcome {
	switch g.ApprovalRule {
	case ApprovalRule_APPROVAL_RULE_WITH_QUORUM:
		return outcomeWithQuorum(g, t)
	case ApprovalRule_APPROVAL_RULE_WITHOUT_QUORUM:
		return outcomeWithoutQuorum(g, t)
	default:
		// Unknown approval rule — fail safe. Validation should prevent us
		// from ever getting here for stored proposals.
		return OutcomeReject
	}
}

// outcomeWithQuorum implements the WITH_QUORUM rule:
//
//	voted   = yes + no + abstain
//	quorum  = voted * 10000 / total       >= quorum_bps
//	support = yes   * 10000 / (yes + no)  >= threshold_bps  (abstain excluded)
//	pass    = quorum AND support
//
// Edge: if yes+no == 0, support is undefined → REJECT.
func outcomeWithQuorum(g GovernanceConfig, t Tally) TallyOutcome {
	if t.TotalPower == 0 {
		return OutcomeReject
	}
	voted := t.YesPower + t.NoPower + t.AbstainPower
	if MulLess(voted, BpsDenominator, uint64(g.QuorumBps), t.TotalPower) {
		return OutcomeReject
	}
	yesNo := t.YesPower + t.NoPower
	if yesNo == 0 {
		return OutcomeReject
	}
	if MulLess(t.YesPower, BpsDenominator, uint64(g.ThresholdBps), yesNo) {
		return OutcomeReject
	}
	return OutcomePass
}

// outcomeWithoutQuorum implements the WITHOUT_QUORUM rule:
//
//	pass = yes / total >= threshold_bps / 10000
//
// No / abstain effectively block, as do non-voters.
func outcomeWithoutQuorum(g GovernanceConfig, t Tally) TallyOutcome {
	if t.TotalPower == 0 {
		return OutcomeReject
	}
	if MulLess(t.YesPower, BpsDenominator, uint64(g.ThresholdBps), t.TotalPower) {
		return OutcomeReject
	}
	return OutcomePass
}

// PollOutcome captures finalize results for a poll. Mirrors TallyOutcome
// but carries WinningChoiceIndex when status is CONCLUDED.
type PollOutcome struct {
	Status             PollStatus
	WinningChoiceIndex uint32
}

// ComputePollOutcome decides a poll's terminal state given its frozen
// tally / quorum / NOTA settings. Pure function. Algorithm (per Epic 6
// plan):
//
//  1. If quorum_bps > 0 AND voted*10000 < quorum_bps*total → REJECTED.
//  2. Find max(choice_power).
//  3. If max == 0 → REJECTED (degenerate; no votes at all).
//  4. Collect all indices with power == max. If len > 1 → REJECTED (tie).
//  5. If include_nota AND winner == nota_index → REJECTED (NOTA wins).
//  6. Otherwise → CONCLUDED with winning_choice_index = winner.
func ComputePollOutcome(p Poll) PollOutcome {
	t := p.Tally

	if p.QuorumBps > 0 {
		if MulLess(t.TotalVotedPower, BpsDenominator, uint64(p.QuorumBps), t.TotalPower) {
			return PollOutcome{Status: PollStatus_POLL_STATUS_REJECTED}
		}
	}

	var maxPower uint64
	winners := make([]uint32, 0, 1)
	for i, cp := range t.ChoicePower {
		if cp > maxPower {
			maxPower = cp
			winners = winners[:0]
			winners = append(winners, uint32(i))
		} else if cp == maxPower && cp > 0 {
			winners = append(winners, uint32(i))
		}
	}

	if maxPower == 0 {
		return PollOutcome{Status: PollStatus_POLL_STATUS_REJECTED}
	}
	if len(winners) > 1 {
		return PollOutcome{Status: PollStatus_POLL_STATUS_REJECTED}
	}

	winner := winners[0]
	if p.IncludeNota && int(winner) == len(p.Choices)-1 {
		return PollOutcome{Status: PollStatus_POLL_STATUS_REJECTED}
	}

	return PollOutcome{
		Status:             PollStatus_POLL_STATUS_CONCLUDED,
		WinningChoiceIndex: winner,
	}
}
