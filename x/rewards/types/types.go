package types

const (
	//TODO: move them to module params
	MaxUnlocksPerBlock                  = 100
	MaxStakingDistributionsPerBlock     = 100
	MaxTradingRewardRemovalsPerBlock    = 100
	MaxDenomRewardDistributionsPerBlock = 100

	// MaxTradingRewardCandidateRemovalsPerBlock caps the candidate rows of expired trading rewards
	// removed in one block, across all rewards waiting for removal. A row removal is a single
	// store delete, much cheaper than the operations capped above.
	MaxTradingRewardCandidateRemovalsPerBlock = 500
)
