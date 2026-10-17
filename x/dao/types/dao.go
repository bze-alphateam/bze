package types

import (
	"time"

	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	// MaxNameLength is the longest DAO name, in bytes.
	MaxNameLength = 64
	// MaxDescriptionLength is the longest DAO description, in bytes.
	MaxDescriptionLength = 2_000
)

// ValidateMetadata checks the DAO name (1..MaxNameLength bytes) and description
// (0..MaxDescriptionLength bytes).
func ValidateMetadata(name, description string) error {
	if len(name) == 0 || len(name) > MaxNameLength {
		return errorsmod.Wrapf(ErrInvalidMetadata, "name must have 1 to %d bytes, got %d", MaxNameLength, len(name))
	}

	if len(description) > MaxDescriptionLength {
		return errorsmod.Wrapf(ErrInvalidMetadata, "description must have at most %d bytes, got %d", MaxDescriptionLength, len(description))
	}

	return nil
}

// Validate checks the stateless rules of a DAO config. The expedite fields are
// validated even when allow_expedite is false, so enabling expedite later never
// activates an invalid setting. The period bounds come from the params, see
// ValidatePeriods.
func (c DaoConfig) Validate() error {
	ratios := []struct {
		name  string
		value math.LegacyDec
	}{
		{"min_quorum", c.MinQuorum},
		{"threshold", c.Threshold},
		{"expedite_quorum", c.ExpediteQuorum},
		{"expedite_threshold", c.ExpediteThreshold},
	}
	for _, r := range ratios {
		if err := validateRatio(r.name, r.value); err != nil {
			return err
		}
	}

	if c.ExpediteQuorum.LT(c.MinQuorum) {
		return errorsmod.Wrapf(ErrInvalidConfig, "expedite_quorum %s is lower than min_quorum %s", c.ExpediteQuorum, c.MinQuorum)
	}

	if c.ExpediteThreshold.LT(c.Threshold) {
		return errorsmod.Wrapf(ErrInvalidConfig, "expedite_threshold %s is lower than threshold %s", c.ExpediteThreshold, c.Threshold)
	}

	if c.ExpeditedPeriod > c.VotingPeriod {
		return errorsmod.Wrapf(ErrInvalidConfig, "expedited_period %s is longer than voting_period %s", c.ExpeditedPeriod, c.VotingPeriod)
	}

	if err := validateOptionalCoin("proposal_cost", c.ProposalCost); err != nil {
		return err
	}

	if err := validateOptionalCoin("proposal_deposit", c.ProposalDeposit); err != nil {
		return err
	}

	if _, ok := DepositRefundPolicy_name[int32(c.DepositRefundPolicy)]; !ok {
		return errorsmod.Wrapf(ErrInvalidConfig, "unknown deposit_refund_policy %d", c.DepositRefundPolicy)
	}

	if _, ok := FundsDestination_name[int32(c.CostDestination)]; !ok {
		return errorsmod.Wrapf(ErrInvalidConfig, "unknown cost_destination %d", c.CostDestination)
	}

	if _, ok := FundsDestination_name[int32(c.KeptDepositDestination)]; !ok {
		return errorsmod.Wrapf(ErrInvalidConfig, "unknown kept_deposit_destination %d", c.KeptDepositDestination)
	}

	return nil
}

// ValidatePeriods checks that both the voting and the expedited period are within
// [minPeriod, maxPeriod] (the min_voting_period and max_voting_period params).
func (c DaoConfig) ValidatePeriods(minPeriod, maxPeriod time.Duration) error {
	periods := []struct {
		name  string
		value time.Duration
	}{
		{"voting_period", c.VotingPeriod},
		{"expedited_period", c.ExpeditedPeriod},
	}
	for _, p := range periods {
		if p.value < minPeriod || p.value > maxPeriod {
			return errorsmod.Wrapf(ErrInvalidConfig, "%s %s is outside [%s, %s]", p.name, p.value, minPeriod, maxPeriod)
		}
	}

	return nil
}

// ChangedFields returns the proto names of the fields whose value differs
// between c and next, in proto order. A coin set to zero differs from an unset
// one.
func (c DaoConfig) ChangedFields(next DaoConfig) []string {
	fields := []struct {
		name    string
		changed bool
	}{
		{"min_quorum", !decEqual(c.MinQuorum, next.MinQuorum)},
		{"threshold", !decEqual(c.Threshold, next.Threshold)},
		{"voting_period", c.VotingPeriod != next.VotingPeriod},
		{"allow_expedite", c.AllowExpedite != next.AllowExpedite},
		{"expedited_period", c.ExpeditedPeriod != next.ExpeditedPeriod},
		{"expedite_quorum", !decEqual(c.ExpediteQuorum, next.ExpediteQuorum)},
		{"expedite_threshold", !decEqual(c.ExpediteThreshold, next.ExpediteThreshold)},
		{"open_submission", c.OpenSubmission != next.OpenSubmission},
		{"proposal_cost", !coinEqual(c.ProposalCost, next.ProposalCost)},
		{"proposal_deposit", !coinEqual(c.ProposalDeposit, next.ProposalDeposit)},
		{"deposit_refund_policy", c.DepositRefundPolicy != next.DepositRefundPolicy},
		{"cost_destination", c.CostDestination != next.CostDestination},
		{"kept_deposit_destination", c.KeptDepositDestination != next.KeptDepositDestination},
	}

	changed := []string{}
	for _, f := range fields {
		if f.changed {
			changed = append(changed, f.name)
		}
	}

	return changed
}

// IsEmpty reports whether the electorate names no reward program at all.
func (e *StakeElectorate) IsEmpty() bool {
	return e == nil || (len(e.SrIds) == 0 && e.DrDenom == "")
}

// validateRatio requires value in (0, 1].
func validateRatio(name string, value math.LegacyDec) error {
	if value.IsNil() || !value.IsPositive() || value.GT(math.LegacyOneDec()) {
		return errorsmod.Wrapf(ErrInvalidConfig, "%s must be in (0, 1], got %s", name, value)
	}

	return nil
}

// validateOptionalCoin accepts a nil coin; a set coin needs a valid denom and a
// non-negative amount.
func validateOptionalCoin(name string, coin *sdk.Coin) error {
	if coin == nil {
		return nil
	}

	if err := coin.Validate(); err != nil {
		return errorsmod.Wrapf(ErrInvalidConfig, "invalid %s: %s", name, err)
	}

	return nil
}

// decEqual compares two decimals, an unset one being equal only to another unset one.
func decEqual(a, b math.LegacyDec) bool {
	if a.IsNil() || b.IsNil() {
		return a.IsNil() == b.IsNil()
	}

	return a.Equal(b)
}

// coinEqual compares two optional coins by denom and amount.
func coinEqual(a, b *sdk.Coin) bool {
	if a == nil || b == nil {
		return a == b
	}

	if a.Amount.IsNil() || b.Amount.IsNil() {
		return a.Denom == b.Denom && a.Amount.IsNil() == b.Amount.IsNil()
	}

	return a.Denom == b.Denom && a.Amount.Equal(b.Amount)
}
