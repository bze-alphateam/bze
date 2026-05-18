package types

import (
	"time"

	errorsmod "cosmossdk.io/errors"
)

// Title / description caps for MsgCreateProposal. Hardcoded UI/UX bounds —
// not protocol policy, so no Params slot.
//
// Description cap chosen to fit a typical block-size budget when multiplied
// by the per-proposal `msgs` cap (Params.max_msgs_per_proposal, default 32).
const (
	MaxProposalTitleLen       = 200
	MaxProposalDescriptionLen = 16_384
)

// ValidateGovernanceConfigStateless enforces the brick-prevention caps on a
// GovernanceConfig that don't require Params or chain state. Called from
// MsgCreateDao.ValidateBasic and MsgUpdateGovernanceConfig.ValidateBasic.
//
// Bounds enforced here:
//   - approval_rule is WITH_QUORUM or WITHOUT_QUORUM (UNSPECIFIED rejected).
//   - threshold_bps in [1, MaxThresholdBps] (=9_900 = 99%).
//   - quorum_bps:
//       WITH_QUORUM:    [1, MaxQuorumBps] (=8_500 = 85%).
//       WITHOUT_QUORUM: must be exactly 0 (quorum is meaningless here).
//   - voting_period >= MinVotingPeriod (=1h). The upper bound depends on
//     Params.max_voting_period and is enforced separately by
//     ValidateGovernanceConfigAgainstParams (keeper-level).
//
// allow_revote is a bool — no validation needed.
func ValidateGovernanceConfigStateless(g GovernanceConfig) error {
	switch g.ApprovalRule {
	case ApprovalRule_APPROVAL_RULE_WITH_QUORUM:
		if g.QuorumBps == 0 {
			return errorsmod.Wrap(ErrInvalidGovernanceConfig,
				"WITH_QUORUM requires quorum_bps >= 1")
		}
		if g.QuorumBps > MaxQuorumBps {
			return errorsmod.Wrapf(ErrInvalidGovernanceConfig,
				"WITH_QUORUM quorum_bps %d exceeds cap %d", g.QuorumBps, MaxQuorumBps)
		}
	case ApprovalRule_APPROVAL_RULE_WITHOUT_QUORUM:
		if g.QuorumBps != 0 {
			return errorsmod.Wrapf(ErrInvalidGovernanceConfig,
				"WITHOUT_QUORUM requires quorum_bps == 0, got %d", g.QuorumBps)
		}
	default:
		return errorsmod.Wrapf(ErrInvalidGovernanceConfig,
			"approval_rule must be WITH_QUORUM or WITHOUT_QUORUM, got %v", g.ApprovalRule)
	}

	if g.ThresholdBps == 0 {
		return errorsmod.Wrap(ErrInvalidGovernanceConfig, "threshold_bps must be >= 1")
	}
	if g.ThresholdBps > MaxThresholdBps {
		return errorsmod.Wrapf(ErrInvalidGovernanceConfig,
			"threshold_bps %d exceeds cap %d", g.ThresholdBps, MaxThresholdBps)
	}

	if g.VotingPeriod < MinVotingPeriod {
		return errorsmod.Wrapf(ErrInvalidGovernanceConfig,
			"voting_period %s is below floor %s", g.VotingPeriod, MinVotingPeriod)
	}

	return nil
}

// ValidateGovernanceConfigAgainstParams enforces the Param-dependent upper
// bound on voting_period. Called from the keeper after Params are loaded.
//
// Kept separate from ValidateGovernanceConfigStateless so ValidateBasic can
// still reject obviously-bad configs without keeper state.
func ValidateGovernanceConfigAgainstParams(g GovernanceConfig, maxVotingPeriod time.Duration) error {
	if g.VotingPeriod > maxVotingPeriod {
		return errorsmod.Wrapf(ErrInvalidGovernanceConfig,
			"voting_period %s exceeds chain cap %s", g.VotingPeriod, maxVotingPeriod)
	}
	return nil
}
