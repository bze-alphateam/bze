package types

import (
	"fmt"
	"strconv"
	// this line is used by starport scaffolding # genesis/types/import
)

// DefaultIndex is the default global index
const DefaultIndex uint64 = 1

// DefaultGenesis returns the default genesis state
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		// this line is used by starport scaffolding # genesis/types/default
		Params: DefaultParams(),
	}
}

// Validate performs basic genesis state validation returning an error upon any
// failure.
func (gs GenesisState) Validate() error {
	// this line is used by starport scaffolding # genesis/types/validate
	if err := gs.validateDenomRewards(); err != nil {
		return err
	}

	return gs.Params.Validate()
}

// validateDenomRewards checks the referential integrity of the Denom Rewards genesis
// state: DRs are unique per staking denom, every dependent record points at an existing
// DR, indexes point at existing prize accumulators, the schedule counter has not fallen
// behind an exported schedule id, and a non-empty distribution queue cursor points at an
// existing schedule.
func (gs GenesisState) validateDenomRewards() error {
	drDenoms := make(map[string]struct{}, len(gs.DenomRewardList))
	for _, dr := range gs.DenomRewardList {
		if _, ok := drDenoms[dr.StakingDenom]; ok {
			return fmt.Errorf("duplicate denom reward for staking denom %s", dr.StakingDenom)
		}
		drDenoms[dr.StakingDenom] = struct{}{}
	}

	prizeKeys := make(map[string]struct{}, len(gs.DenomRewardPrizeList))
	for _, prize := range gs.DenomRewardPrizeList {
		if _, ok := drDenoms[prize.StakingDenom]; !ok {
			return fmt.Errorf("denom reward prize %s/%s references missing denom reward", prize.StakingDenom, prize.PrizeDenom)
		}
		prizeKeys[string(DenomRewardPrizeKey(prize.StakingDenom, prize.PrizeDenom))] = struct{}{}
	}

	for _, participant := range gs.DenomRewardParticipantList {
		if _, ok := drDenoms[participant.StakingDenom]; !ok {
			return fmt.Errorf("denom reward participant %s/%s references missing denom reward", participant.StakingDenom, participant.Address)
		}
	}

	for _, index := range gs.DenomRewardParticipantIndexList {
		if _, ok := drDenoms[index.StakingDenom]; !ok {
			return fmt.Errorf("denom reward participant index %s/%s/%s references missing denom reward", index.Address, index.StakingDenom, index.PrizeDenom)
		}
		if _, ok := prizeKeys[string(DenomRewardPrizeKey(index.StakingDenom, index.PrizeDenom))]; !ok {
			return fmt.Errorf("denom reward participant index %s/%s/%s references missing denom reward prize", index.Address, index.StakingDenom, index.PrizeDenom)
		}
	}

	scheduleKeys := make(map[string]struct{}, len(gs.DenomRewardScheduleList))
	for _, schedule := range gs.DenomRewardScheduleList {
		if _, ok := drDenoms[schedule.StakingDenom]; !ok {
			return fmt.Errorf("denom reward schedule %s/%s references missing denom reward", schedule.StakingDenom, schedule.ScheduleId)
		}

		id, err := strconv.ParseUint(schedule.ScheduleId, 10, 64)
		if err != nil {
			return fmt.Errorf("denom reward schedule %s/%s has a non-numeric schedule id", schedule.StakingDenom, schedule.ScheduleId)
		}
		if id > gs.DenomRewardScheduleCounter {
			return fmt.Errorf("denom reward schedule counter %d is behind schedule id %s", gs.DenomRewardScheduleCounter, schedule.ScheduleId)
		}

		scheduleKeys[string(DenomRewardScheduleKey(schedule.StakingDenom, schedule.ScheduleId))] = struct{}{}
	}

	if gs.DenomRewardsDistributionQueue != nil && gs.DenomRewardsDistributionQueue.Cursor != "" {
		if _, ok := scheduleKeys[gs.DenomRewardsDistributionQueue.Cursor]; !ok {
			return fmt.Errorf("denom rewards distribution queue cursor %q references missing schedule", gs.DenomRewardsDistributionQueue.Cursor)
		}
	}

	return nil
}
