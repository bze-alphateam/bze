package types

import (
	"time"

	errorsmod "cosmossdk.io/errors"
)

// ValidateDepositConfigStateless enforces the brick-prevention caps on a
// DepositConfig that don't require Params or chain state. Called from
// MsgCreateDao.ValidateBasic and MsgUpdateDepositConfig.ValidateBasic.
//
// Bounds enforced here:
//   - min_deposit.Validate() (denom is valid, amount is non-negative).
//   - min_deposit.amount > 0. A zero-amount min_deposit defeats the
//     mechanism — every proposal would auto-pass the deposit phase.
//   - deposit_period >= MinDepositPeriod (=24h). Upper bound depends on
//     Params.max_deposit_period and is enforced by
//     ValidateDepositConfigAgainstParams (keeper-level).
//   - forfeit_destination and voting_refund_policy are known variants
//     (UNSPECIFIED rejected).
func ValidateDepositConfigStateless(d DepositConfig) error {
	if err := d.MinDeposit.Validate(); err != nil {
		return errorsmod.Wrapf(ErrInvalidDepositConfig, "min_deposit: %s", err.Error())
	}
	if d.MinDeposit.Amount.IsNil() || !d.MinDeposit.Amount.IsPositive() {
		return errorsmod.Wrap(ErrInvalidDepositConfig, "min_deposit.amount must be > 0")
	}
	if d.DepositPeriod < MinDepositPeriod {
		return errorsmod.Wrapf(ErrInvalidDepositConfig,
			"deposit_period %s is below floor %s", d.DepositPeriod, MinDepositPeriod)
	}
	switch d.ForfeitDestination {
	case ForfeitDestination_FORFEIT_DESTINATION_BURNER,
		ForfeitDestination_FORFEIT_DESTINATION_TREASURY:
		// ok
	default:
		return errorsmod.Wrapf(ErrInvalidDepositConfig,
			"forfeit_destination must be BURNER or TREASURY, got %v", d.ForfeitDestination)
	}
	switch d.VotingRefundPolicy {
	case RefundPolicy_REFUND_POLICY_ALWAYS,
		RefundPolicy_REFUND_POLICY_ON_PASS,
		RefundPolicy_REFUND_POLICY_NEVER:
		// ok
	default:
		return errorsmod.Wrapf(ErrInvalidDepositConfig,
			"voting_refund_policy must be ALWAYS / ON_PASS / NEVER, got %v", d.VotingRefundPolicy)
	}
	return nil
}

// ValidateDepositConfigAgainstParams enforces the Param-dependent upper
// bound on deposit_period. Called from the keeper after Params are
// loaded; kept separate from ValidateDepositConfigStateless so
// ValidateBasic can run with no keeper state.
func ValidateDepositConfigAgainstParams(d DepositConfig, maxDepositPeriod time.Duration) error {
	if d.DepositPeriod > maxDepositPeriod {
		return errorsmod.Wrapf(ErrInvalidDepositConfig,
			"deposit_period %s exceeds chain cap %s", d.DepositPeriod, maxDepositPeriod)
	}
	return nil
}
