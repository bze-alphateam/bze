package keeper

import (
	"github.com/bze-alphateam/bze/x/daodao/types"
)

// The pure tally math (bps-denominator helpers, ComputeOutcome,
// ComputePollOutcome) lives in `x/daodao/types/tally_math.go` so genesis
// Validate can recompute terminal-status outcomes without an import
// cycle. Only the keeper-flavoured pieces — the early-close decision
// helper — remain here.

// earlyCloseDecision describes the result of an early-close check after a
// MsgVote in a no-revote proposal.
type earlyCloseDecision int

const (
	earlyCloseNone earlyCloseDecision = iota
	earlyClosePass
	earlyCloseReject
)

// checkEarlyClose returns whether a proposal can be closed before
// voting_end. Only called when governance_snapshot.allow_revote == false:
// with revote enabled, any vote can be undone so no lock-in is possible.
//
// remaining = total - (yes + no + abstain)
//
// WITH_QUORUM:
//   Pass-locked ⇔ remaining cannot break the result. Concretely:
//     - quorum already met (yes+no+abstain >= q*total/10000), AND
//     - yes/(yes+no+remaining) >= threshold (every remaining vote goes
//       NO and threshold still holds — strict pessimistic case).
//
//   Reject-locked ⇔ no allocation of remaining can pass. Concretely:
//     - quorum is unreachable even if all remaining vote, OR
//     - threshold is unreachable even if all remaining vote YES.
//
// WITHOUT_QUORUM:
//   Pass-locked ⇔ yes already satisfies threshold of total.
//   Reject-locked ⇔ yes + remaining cannot satisfy threshold of total.
func checkEarlyClose(g types.GovernanceConfig, t types.Tally) earlyCloseDecision {
	voted := t.YesPower + t.NoPower + t.AbstainPower
	if voted > t.TotalPower {
		// Defensive — should be unreachable. Treat as locked-reject to
		// fail-safe rather than mis-compute.
		return earlyCloseReject
	}
	remaining := t.TotalPower - voted

	switch g.ApprovalRule {
	case types.ApprovalRule_APPROVAL_RULE_WITH_QUORUM:
		quorumMet := types.MulGreaterOrEqual(voted, types.BpsDenominator, uint64(g.QuorumBps), t.TotalPower)
		yesNoIfAllNo := t.YesPower + t.NoPower + remaining
		thresholdHoldsAgainstNo := yesNoIfAllNo > 0 &&
			types.MulGreaterOrEqual(t.YesPower, types.BpsDenominator, uint64(g.ThresholdBps), yesNoIfAllNo)
		if quorumMet && thresholdHoldsAgainstNo {
			return earlyClosePass
		}

		maxVoted := voted + remaining // = t.TotalPower
		quorumUnreachable := types.MulLess(maxVoted, types.BpsDenominator, uint64(g.QuorumBps), t.TotalPower)
		yesIfAllYes := t.YesPower + remaining
		yesNoIfAllYes := t.YesPower + t.NoPower + remaining
		thresholdUnreachable := yesNoIfAllYes == 0 ||
			types.MulLess(yesIfAllYes, types.BpsDenominator, uint64(g.ThresholdBps), yesNoIfAllYes)
		if quorumUnreachable || thresholdUnreachable {
			return earlyCloseReject
		}
		return earlyCloseNone

	case types.ApprovalRule_APPROVAL_RULE_WITHOUT_QUORUM:
		if types.MulGreaterOrEqual(t.YesPower, types.BpsDenominator, uint64(g.ThresholdBps), t.TotalPower) {
			return earlyClosePass
		}
		yesIfAllYes := t.YesPower + remaining
		if types.MulLess(yesIfAllYes, types.BpsDenominator, uint64(g.ThresholdBps), t.TotalPower) {
			return earlyCloseReject
		}
		return earlyCloseNone

	default:
		return earlyCloseNone
	}
}
