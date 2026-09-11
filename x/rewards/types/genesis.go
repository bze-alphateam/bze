package types

import (
	"fmt"
	"strconv"

	"cosmossdk.io/math"
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

// validateDenomRewards checks the Denom Rewards genesis state.
//
// Referential integrity: DRs are unique per staking denom, every dependent record points at an
// existing DR, indexes and schedules point at existing prize accumulators, the schedule counter
// has not fallen behind an exported schedule id, and a non-empty distribution queue cursor points
// at an existing schedule.
//
// Values: every record type is free of duplicates (the store keeps one record per key, so a
// duplicate would silently be last-wins on import); staked amounts and accumulators are never
// negative; participant amounts are positive and sum to the DR's staked amount (the accumulator
// math and the all-or-nothing exit rely on it); a participant index never runs ahead of its
// prize accumulator (the pending amount would be negative and the position unclaimable); a
// schedule has a positive daily amount, a non-zero duration and fewer payouts than its duration
// (otherwise it is re-enqueued every day and never finishes).
//
// A chain export always satisfies these rules; only a hand-edited genesis can break them, and it
// is better rejected by validate-genesis than imported into a state that strands escrow or
// wedges the daily distribution pass. Not enforced on purpose: the prize-denom cap
// (Params.MaxPrizeDenomsPerDr only gates the creation of new prizes, and governance may lower it
// below an existing pool's count, so an export can legitimately exceed it) and an index without a
// participant record (a settled position may leave its index markers behind).
func (gs GenesisState) validateDenomRewards() error {
	drStaked := make(map[string]math.Int, len(gs.DenomRewardList))
	for _, dr := range gs.DenomRewardList {
		if _, ok := drStaked[dr.StakingDenom]; ok {
			return fmt.Errorf("duplicate denom reward for staking denom %s", dr.StakingDenom)
		}

		staked := intOrZero(dr.StakedAmount)
		if staked.IsNegative() {
			return fmt.Errorf("denom reward %s has a negative staked amount %s", dr.StakingDenom, staked)
		}
		drStaked[dr.StakingDenom] = staked
	}

	prizeAccumulators := make(map[string]math.LegacyDec, len(gs.DenomRewardPrizeList))
	for _, prize := range gs.DenomRewardPrizeList {
		if _, ok := drStaked[prize.StakingDenom]; !ok {
			return fmt.Errorf("denom reward prize %s/%s references missing denom reward", prize.StakingDenom, prize.PrizeDenom)
		}

		key := string(DenomRewardPrizeKey(prize.StakingDenom, prize.PrizeDenom))
		if _, ok := prizeAccumulators[key]; ok {
			return fmt.Errorf("duplicate denom reward prize %s/%s", prize.StakingDenom, prize.PrizeDenom)
		}

		accumulator := decOrZero(prize.DistributedStake)
		if accumulator.IsNegative() {
			return fmt.Errorf("denom reward prize %s/%s has a negative accumulator %s", prize.StakingDenom, prize.PrizeDenom, accumulator)
		}
		prizeAccumulators[key] = accumulator
	}

	participantKeys := make(map[string]struct{}, len(gs.DenomRewardParticipantList))
	participantSums := make(map[string]math.Int, len(gs.DenomRewardList))
	for _, participant := range gs.DenomRewardParticipantList {
		if _, ok := drStaked[participant.StakingDenom]; !ok {
			return fmt.Errorf("denom reward participant %s/%s references missing denom reward", participant.StakingDenom, participant.Address)
		}

		key := string(DenomRewardParticipantKey(participant.StakingDenom, participant.Address))
		if _, ok := participantKeys[key]; ok {
			return fmt.Errorf("duplicate denom reward participant %s/%s", participant.StakingDenom, participant.Address)
		}
		participantKeys[key] = struct{}{}

		amount := intOrZero(participant.Amount)
		if !amount.IsPositive() {
			return fmt.Errorf("denom reward participant %s/%s has a non-positive amount %s", participant.StakingDenom, participant.Address, amount)
		}

		sum, ok := participantSums[participant.StakingDenom]
		if !ok {
			sum = math.ZeroInt()
		}
		participantSums[participant.StakingDenom] = sum.Add(amount)
	}

	for _, dr := range gs.DenomRewardList {
		sum, ok := participantSums[dr.StakingDenom]
		if !ok {
			sum = math.ZeroInt()
		}
		if staked := drStaked[dr.StakingDenom]; !staked.Equal(sum) {
			return fmt.Errorf("denom reward %s staked amount %s does not match the sum of participant amounts %s", dr.StakingDenom, staked, sum)
		}
	}

	indexKeys := make(map[string]struct{}, len(gs.DenomRewardParticipantIndexList))
	for _, index := range gs.DenomRewardParticipantIndexList {
		if _, ok := drStaked[index.StakingDenom]; !ok {
			return fmt.Errorf("denom reward participant index %s/%s/%s references missing denom reward", index.Address, index.StakingDenom, index.PrizeDenom)
		}
		accumulator, ok := prizeAccumulators[string(DenomRewardPrizeKey(index.StakingDenom, index.PrizeDenom))]
		if !ok {
			return fmt.Errorf("denom reward participant index %s/%s/%s references missing denom reward prize", index.Address, index.StakingDenom, index.PrizeDenom)
		}

		key := string(DenomRewardParticipantIndexKey(index.Address, index.StakingDenom, index.PrizeDenom))
		if _, ok := indexKeys[key]; ok {
			return fmt.Errorf("duplicate denom reward participant index %s/%s/%s", index.Address, index.StakingDenom, index.PrizeDenom)
		}
		indexKeys[key] = struct{}{}

		value := decOrZero(index.Index)
		if value.IsNegative() {
			return fmt.Errorf("denom reward participant index %s/%s/%s has a negative index %s", index.Address, index.StakingDenom, index.PrizeDenom, value)
		}
		if value.GT(accumulator) {
			return fmt.Errorf("denom reward participant index %s/%s/%s is ahead of the prize accumulator (%s > %s)", index.Address, index.StakingDenom, index.PrizeDenom, value, accumulator)
		}
	}

	scheduleKeys := make(map[string]struct{}, len(gs.DenomRewardScheduleList))
	for _, schedule := range gs.DenomRewardScheduleList {
		if _, ok := drStaked[schedule.StakingDenom]; !ok {
			return fmt.Errorf("denom reward schedule %s/%s references missing denom reward", schedule.StakingDenom, schedule.ScheduleId)
		}
		if _, ok := prizeAccumulators[string(DenomRewardPrizeKey(schedule.StakingDenom, schedule.PrizeDenom))]; !ok {
			return fmt.Errorf("denom reward schedule %s/%s references missing denom reward prize %s", schedule.StakingDenom, schedule.ScheduleId, schedule.PrizeDenom)
		}

		id, err := strconv.ParseUint(schedule.ScheduleId, 10, 64)
		if err != nil {
			return fmt.Errorf("denom reward schedule %s/%s has a non-numeric schedule id", schedule.StakingDenom, schedule.ScheduleId)
		}
		if id > gs.DenomRewardScheduleCounter {
			return fmt.Errorf("denom reward schedule counter %d is behind schedule id %s", gs.DenomRewardScheduleCounter, schedule.ScheduleId)
		}

		key := string(DenomRewardScheduleKey(schedule.StakingDenom, schedule.ScheduleId))
		if _, ok := scheduleKeys[key]; ok {
			return fmt.Errorf("duplicate denom reward schedule %s/%s", schedule.StakingDenom, schedule.ScheduleId)
		}
		scheduleKeys[key] = struct{}{}

		if !intOrZero(schedule.DailyAmount).IsPositive() {
			return fmt.Errorf("denom reward schedule %s/%s has a non-positive daily amount %s", schedule.StakingDenom, schedule.ScheduleId, intOrZero(schedule.DailyAmount))
		}
		if schedule.Duration == 0 {
			return fmt.Errorf("denom reward schedule %s/%s has zero duration", schedule.StakingDenom, schedule.ScheduleId)
		}
		if schedule.Payouts >= schedule.Duration {
			return fmt.Errorf("denom reward schedule %s/%s has %d payouts for a duration of %d and would never finish", schedule.StakingDenom, schedule.ScheduleId, schedule.Payouts, schedule.Duration)
		}
	}

	if gs.DenomRewardsDistributionQueue != nil && gs.DenomRewardsDistributionQueue.Cursor != "" {
		if _, ok := scheduleKeys[gs.DenomRewardsDistributionQueue.Cursor]; !ok {
			return fmt.Errorf("denom rewards distribution queue cursor %q references missing schedule", gs.DenomRewardsDistributionQueue.Cursor)
		}
	}

	return nil
}

// intOrZero treats a customtype math.Int left nil by an absent JSON field as zero, which is what
// the keeper reads back after import.
func intOrZero(i math.Int) math.Int {
	if i.IsNil() {
		return math.ZeroInt()
	}

	return i
}

// decOrZero is intOrZero for math.LegacyDec.
func decOrZero(d math.LegacyDec) math.LegacyDec {
	if d.IsNil() {
		return math.LegacyZeroDec()
	}

	return d
}
