package types

import (
	"fmt"
	"time"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Hardcoded constants (not in Params — these are protocol semantics, not policy).
const (
	// FeeDestinationBurnerModule is the Params.dao_creation_fee_destination
	// value that routes the creation fee to x/burner. Default.
	FeeDestinationBurnerModule = "burner"

	// FeeDestinationCommunityPool is the Params.dao_creation_fee_destination
	// value that routes the creation fee to the chain community pool.
	FeeDestinationCommunityPool = "community_pool"

	// MinVotingPeriod is the hardcoded floor on a DAO's governance.voting_period.
	// Enforced in Epic 3 when GovernanceConfig is introduced.
	MinVotingPeriod = time.Hour

	// MinDepositPeriod is the hardcoded floor on a DAO's deposit.deposit_period.
	// Enforced in Epic 4 when DepositConfig is introduced.
	MinDepositPeriod = 24 * time.Hour

	// MaxThresholdBps caps DAO governance threshold. Enforced in Epic 3.
	MaxThresholdBps = uint32(9_900)

	// MaxQuorumBps caps DAO governance quorum. Enforced in Epic 3.
	MaxQuorumBps = uint32(8_500)
)

// Default values for Params.
var (
	DefaultDaoCreationFee            = sdk.NewInt64Coin("ubze", 0) // off by default
	DefaultDaoCreationFeeDestination = FeeDestinationBurnerModule
	DefaultMaxVotingPeriod           = 30 * 24 * time.Hour // 30 days
	DefaultMaxDepositPeriod          = 30 * 24 * time.Hour // 30 days
	DefaultMaxMsgsPerProposal        = uint32(32)
)

// NewParams creates a new Params instance.
func NewParams(
	daoCreationFee sdk.Coin,
	daoCreationFeeDestination string,
	maxVotingPeriod time.Duration,
	maxDepositPeriod time.Duration,
	maxMsgsPerProposal uint32,
) Params {
	return Params{
		DaoCreationFee:            daoCreationFee,
		DaoCreationFeeDestination: daoCreationFeeDestination,
		MaxVotingPeriod:           maxVotingPeriod,
		MaxDepositPeriod:          maxDepositPeriod,
		MaxMsgsPerProposal:        maxMsgsPerProposal,
	}
}

// DefaultParams returns a default set of parameters.
func DefaultParams() Params {
	return NewParams(
		DefaultDaoCreationFee,
		DefaultDaoCreationFeeDestination,
		DefaultMaxVotingPeriod,
		DefaultMaxDepositPeriod,
		DefaultMaxMsgsPerProposal,
	)
}

// Validate validates the set of params.
func (p Params) Validate() error {
	if err := validateDaoCreationFee(p.DaoCreationFee); err != nil {
		return errorsmod.Wrap(ErrInvalidParams, err.Error())
	}
	if err := validateDaoCreationFeeDestination(p.DaoCreationFeeDestination); err != nil {
		return errorsmod.Wrap(ErrInvalidParams, err.Error())
	}
	if err := validateDuration("max_voting_period", p.MaxVotingPeriod, MinVotingPeriod); err != nil {
		return errorsmod.Wrap(ErrInvalidParams, err.Error())
	}
	if err := validateDuration("max_deposit_period", p.MaxDepositPeriod, MinDepositPeriod); err != nil {
		return errorsmod.Wrap(ErrInvalidParams, err.Error())
	}
	if p.MaxMsgsPerProposal < 1 {
		return errorsmod.Wrap(ErrInvalidParams, "max_msgs_per_proposal must be >= 1")
	}
	return nil
}

func validateDaoCreationFee(c sdk.Coin) error {
	// Zero is valid (disables the fee). Non-zero requires a valid denom and
	// a non-negative amount; sdk.Coin's own validation handles the rest.
	if err := c.Validate(); err != nil {
		return fmt.Errorf("dao_creation_fee: %w", err)
	}
	return nil
}

func validateDaoCreationFeeDestination(dest string) error {
	switch dest {
	case FeeDestinationBurnerModule, FeeDestinationCommunityPool:
		return nil
	default:
		return fmt.Errorf("dao_creation_fee_destination must be %q or %q, got %q",
			FeeDestinationBurnerModule, FeeDestinationCommunityPool, dest)
	}
}

func validateDuration(name string, d, floor time.Duration) error {
	if d < floor {
		return fmt.Errorf("%s must be at least %s, got %s", name, floor, d)
	}
	return nil
}
