package chain

import (
	"github.com/cosmos/interchaintest/v10/chain/cosmos"
)

// Fast epoch durations set in genesis. The epochs genesis is replaced whole,
// keeping the day, hour and week entries in that (alphabetical) order.
const (
	DayEpochDuration  = "30s"
	HourEpochDuration = "5s"
	WeekEpochDuration = "210s"

	// VotingPeriod and ExpeditedVotingPeriod are the gov voting periods.
	VotingPeriod          = "20s"
	ExpeditedVotingPeriod = "10s"
	// MinDeposit and ExpeditedMinDeposit are the gov deposits, in ubze.
	MinDeposit          = "1000000"
	ExpeditedMinDeposit = "2000000"

	// DenomRewardLock is the rewards denom_reward_lock param, in days
	// (24 hour epochs, about 2 minutes with the fast epochs).
	DenomRewardLock = 1
)

// epoch returns a not-yet-started epoch, in the shape `bzed init` writes it.
func epoch(identifier, duration string) map[string]any {
	return map[string]any{
		"identifier":                 identifier,
		"start_time":                 "0001-01-01T00:00:00Z",
		"duration":                   duration,
		"current_epoch":              "0",
		"current_epoch_start_time":   "0001-01-01T00:00:00Z",
		"epoch_counting_started":     false,
		"current_epoch_start_height": "0",
	}
}

// DefaultGenesis returns the genesis overrides of every e2e network. Suites
// append their own with WithGenesis.
func DefaultGenesis() []cosmos.GenesisKV {
	return []cosmos.GenesisKV{
		cosmos.NewGenesisKV("app_state.epochs.epochs", []map[string]any{
			epoch("day", DayEpochDuration),
			epoch("hour", HourEpochDuration),
			epoch("week", WeekEpochDuration),
		}),

		cosmos.NewGenesisKV("app_state.gov.params.voting_period", VotingPeriod),
		cosmos.NewGenesisKV("app_state.gov.params.expedited_voting_period", ExpeditedVotingPeriod),
		cosmos.NewGenesisKV("app_state.gov.params.min_deposit", []map[string]string{{"denom": Denom, "amount": MinDeposit}}),
		cosmos.NewGenesisKV("app_state.gov.params.expedited_min_deposit", []map[string]string{{"denom": Denom, "amount": ExpeditedMinDeposit}}),

		// The rewards genesis uses camelCase JSON names (as `bzed init` writes them).
		cosmos.NewGenesisKV("app_state.rewards.params.denomRewardLock", DenomRewardLock),

		cosmos.NewGenesisKV("app_state.staking.params.bond_denom", Denom),
		cosmos.NewGenesisKV("app_state.mint.params.mint_denom", Denom),
	}
}
