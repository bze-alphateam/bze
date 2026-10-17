package types

import (
	"fmt"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	FeeDestinationBurner        = "burner"
	FeeDestinationCommunityPool = "community_pool"

	DefaultMaxSrPerDao               = uint64(5)
	DefaultMaxStaticMembers          = uint64(128)
	DefaultMaxOpenProposals          = uint64(10)
	DefaultMaxMsgsPerProposal        = uint64(16)
	DefaultMaxProposalBytes          = uint64(20_000)
	DefaultMaxTitleLen               = uint64(140)
	DefaultMaxSummaryLen             = uint64(5_000)
	DefaultMinVotingPeriod           = time.Hour
	DefaultMaxVotingPeriod           = 90 * 24 * time.Hour
	DefaultExecutionWindow           = 14 * 24 * time.Hour
	DefaultDaoCreationFeeDestination = FeeDestinationBurner
	DefaultMinElectorateLockDays     = uint32(5)
)

// DefaultDaoCreationFee is 50,000 BZE.
var DefaultDaoCreationFee = sdk.NewInt64Coin("ubze", 50_000_000_000)

// NewParams creates a new Params instance
func NewParams(
	maxSrPerDao uint64,
	maxStaticMembers uint64,
	maxOpenProposals uint64,
	maxMsgsPerProposal uint64,
	maxProposalBytes uint64,
	maxTitleLen uint64,
	maxSummaryLen uint64,
	minVotingPeriod time.Duration,
	maxVotingPeriod time.Duration,
	executionWindow time.Duration,
	daoCreationFee sdk.Coin,
	daoCreationFeeDestination string,
	minElectorateLockDays uint32,
) Params {
	return Params{
		MaxSrPerDao:               maxSrPerDao,
		MaxStaticMembers:          maxStaticMembers,
		MaxOpenProposals:          maxOpenProposals,
		MaxMsgsPerProposal:        maxMsgsPerProposal,
		MaxProposalBytes:          maxProposalBytes,
		MaxTitleLen:               maxTitleLen,
		MaxSummaryLen:             maxSummaryLen,
		MinVotingPeriod:           minVotingPeriod,
		MaxVotingPeriod:           maxVotingPeriod,
		ExecutionWindow:           executionWindow,
		DaoCreationFee:            daoCreationFee,
		DaoCreationFeeDestination: daoCreationFeeDestination,
		MinElectorateLockDays:     minElectorateLockDays,
	}
}

// DefaultParams returns a default set of parameters
func DefaultParams() Params {
	return NewParams(
		DefaultMaxSrPerDao,
		DefaultMaxStaticMembers,
		DefaultMaxOpenProposals,
		DefaultMaxMsgsPerProposal,
		DefaultMaxProposalBytes,
		DefaultMaxTitleLen,
		DefaultMaxSummaryLen,
		DefaultMinVotingPeriod,
		DefaultMaxVotingPeriod,
		DefaultExecutionWindow,
		DefaultDaoCreationFee,
		DefaultDaoCreationFeeDestination,
		DefaultMinElectorateLockDays,
	)
}

// Validate validates the set of params
func (p Params) Validate() error {
	if p.MaxSrPerDao == 0 {
		return fmt.Errorf("max_sr_per_dao must be positive")
	}

	if p.MaxStaticMembers == 0 {
		return fmt.Errorf("max_static_members must be positive")
	}

	if p.MaxOpenProposals == 0 {
		return fmt.Errorf("max_open_proposals must be positive")
	}

	if p.MaxMsgsPerProposal == 0 {
		return fmt.Errorf("max_msgs_per_proposal must be positive")
	}

	if p.MaxProposalBytes == 0 {
		return fmt.Errorf("max_proposal_bytes must be positive")
	}

	// a title is required, so a zero cap would refuse every proposal
	if p.MaxTitleLen == 0 {
		return fmt.Errorf("max_title_len must be positive")
	}

	// a zero summary cap is valid: proposals then carry a title only

	if p.MinVotingPeriod <= 0 {
		return fmt.Errorf("min_voting_period must be positive: %s", p.MinVotingPeriod)
	}

	if p.MaxVotingPeriod < p.MinVotingPeriod {
		return fmt.Errorf("max_voting_period %s is lower than min_voting_period %s", p.MaxVotingPeriod, p.MinVotingPeriod)
	}

	// a zero window is valid: a failed execution can then not be retried and
	// expires at the next settle
	if p.ExecutionWindow < 0 {
		return fmt.Errorf("execution_window must not be negative: %s", p.ExecutionWindow)
	}

	// a zero amount is valid and makes DAO creation free
	if err := p.DaoCreationFee.Validate(); err != nil {
		return fmt.Errorf("invalid dao_creation_fee: %w", err)
	}

	if p.DaoCreationFeeDestination != FeeDestinationBurner && p.DaoCreationFeeDestination != FeeDestinationCommunityPool {
		return fmt.Errorf("invalid dao_creation_fee_destination %q: expected %q or %q", p.DaoCreationFeeDestination, FeeDestinationBurner, FeeDestinationCommunityPool)
	}

	// a zero lock would let stake move between wallets mid-vote (D2)
	if p.MinElectorateLockDays == 0 {
		return fmt.Errorf("min_electorate_lock_days must be positive")
	}

	return nil
}
