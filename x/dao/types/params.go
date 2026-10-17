package types

import (
	"fmt"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	FeeDestinationBurner        = "burner"
	FeeDestinationCommunityPool = "community_pool"

	DefaultMaxStaticMembers          = uint64(128)
	DefaultMinVotingPeriod           = time.Hour
	DefaultMaxVotingPeriod           = 90 * 24 * time.Hour
	DefaultDaoCreationFeeDestination = FeeDestinationBurner
)

// DefaultDaoCreationFee is 50,000 BZE.
var DefaultDaoCreationFee = sdk.NewInt64Coin("ubze", 50_000_000_000)

// NewParams creates a new Params instance
func NewParams(
	maxStaticMembers uint64,
	minVotingPeriod time.Duration,
	maxVotingPeriod time.Duration,
	daoCreationFee sdk.Coin,
	daoCreationFeeDestination string,
) Params {
	return Params{
		MaxStaticMembers:          maxStaticMembers,
		MinVotingPeriod:           minVotingPeriod,
		MaxVotingPeriod:           maxVotingPeriod,
		DaoCreationFee:            daoCreationFee,
		DaoCreationFeeDestination: daoCreationFeeDestination,
	}
}

// DefaultParams returns a default set of parameters
func DefaultParams() Params {
	return NewParams(
		DefaultMaxStaticMembers,
		DefaultMinVotingPeriod,
		DefaultMaxVotingPeriod,
		DefaultDaoCreationFee,
		DefaultDaoCreationFeeDestination,
	)
}

// Validate validates the set of params
func (p Params) Validate() error {
	if p.MaxStaticMembers == 0 {
		return fmt.Errorf("max_static_members must be positive")
	}

	if p.MinVotingPeriod <= 0 {
		return fmt.Errorf("min_voting_period must be positive: %s", p.MinVotingPeriod)
	}

	if p.MaxVotingPeriod < p.MinVotingPeriod {
		return fmt.Errorf("max_voting_period %s is lower than min_voting_period %s", p.MaxVotingPeriod, p.MinVotingPeriod)
	}

	// a zero amount is valid and makes DAO creation free
	if err := p.DaoCreationFee.Validate(); err != nil {
		return fmt.Errorf("invalid dao_creation_fee: %w", err)
	}

	if p.DaoCreationFeeDestination != FeeDestinationBurner && p.DaoCreationFeeDestination != FeeDestinationCommunityPool {
		return fmt.Errorf("invalid dao_creation_fee_destination %q: expected %q or %q", p.DaoCreationFeeDestination, FeeDestinationBurner, FeeDestinationCommunityPool)
	}

	return nil
}
