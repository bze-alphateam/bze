package types

import (
	"cosmossdk.io/math"
	"testing"

	"github.com/stretchr/testify/require"
)

// validDenomRewardsGenesis returns a consistent Denom Rewards genesis covering every
// record type: two DRs, prizes on both, a participant with and one without an index,
// two schedules and a mid-drain distribution queue.
func validDenomRewardsGenesis() GenesisState {
	return GenesisState{
		Params: DefaultParams(),
		DenomRewardList: []DenomReward{
			{StakingDenom: "udenom1", Lock: 7, MinStake: 0, StakedAmount: math.NewInt(1000)},
			{StakingDenom: "udenom2", Lock: 7, MinStake: 0, StakedAmount: math.NewInt(0)},
		},
		DenomRewardPrizeList: []DenomRewardPrize{
			{StakingDenom: "udenom1", PrizeDenom: "uprizea", DistributedStake: math.LegacyMustNewDecFromStr("1.5"), LastDistributionEpoch: 10},
			{StakingDenom: "udenom2", PrizeDenom: "uprizeb", DistributedStake: math.LegacyMustNewDecFromStr("0"), LastDistributionEpoch: 0},
		},
		DenomRewardParticipantList: []DenomRewardParticipant{
			{Address: "addr1", StakingDenom: "udenom1", Amount: math.NewInt(400)},
			{Address: "addr2", StakingDenom: "udenom1", Amount: math.NewInt(600)},
		},
		DenomRewardParticipantIndexList: []DenomRewardParticipantIndex{
			{Address: "addr1", StakingDenom: "udenom1", PrizeDenom: "uprizea", Index: math.LegacyMustNewDecFromStr("0.5")},
		},
		DenomRewardScheduleList: []DenomRewardSchedule{
			{ScheduleId: "000000000001", StakingDenom: "udenom1", PrizeDenom: "uprizea", DailyAmount: math.NewInt(100), Duration: 30, Payouts: 10},
			{ScheduleId: "000000000002", StakingDenom: "udenom2", PrizeDenom: "uprizeb", DailyAmount: math.NewInt(50), Duration: 5},
		},
		DenomRewardScheduleCounter: 2,
		DenomRewardsDistributionQueue: &DenomRewardsDistributionQueue{
			Pending: true,
			Cursor:  string(DenomRewardScheduleKey("udenom1", "000000000001")),
		},
	}
}

func TestGenesisState_ValidateDenomRewards(t *testing.T) {
	tests := []struct {
		name     string
		mutate   func(gs *GenesisState)
		expError string
	}{
		{
			name:   "default genesis is valid",
			mutate: func(gs *GenesisState) { *gs = *DefaultGenesis() },
		},
		{
			name:   "full denom rewards genesis is valid",
			mutate: func(gs *GenesisState) {},
		},
		{
			name: "queue with empty cursor is valid",
			mutate: func(gs *GenesisState) {
				gs.DenomRewardsDistributionQueue = &DenomRewardsDistributionQueue{Pending: true, Cursor: ""}
			},
		},
		{
			name: "counter above max schedule id is valid",
			mutate: func(gs *GenesisState) {
				gs.DenomRewardScheduleCounter = 99
			},
		},
		{
			name: "duplicate denom reward",
			mutate: func(gs *GenesisState) {
				gs.DenomRewardList = append(gs.DenomRewardList, DenomReward{StakingDenom: "udenom1", StakedAmount: math.NewInt(5)})
			},
			expError: "duplicate denom reward",
		},
		{
			name: "prize references missing denom reward",
			mutate: func(gs *GenesisState) {
				gs.DenomRewardPrizeList[0].StakingDenom = "missing"
			},
			expError: "prize missing/uprizea references missing denom reward",
		},
		{
			name: "participant references missing denom reward",
			mutate: func(gs *GenesisState) {
				gs.DenomRewardParticipantList[1].StakingDenom = "missing"
			},
			expError: "participant missing/addr2 references missing denom reward",
		},
		{
			name: "index references missing denom reward",
			mutate: func(gs *GenesisState) {
				gs.DenomRewardParticipantIndexList[0].StakingDenom = "missing"
			},
			expError: "index addr1/missing/uprizea references missing denom reward",
		},
		{
			name: "index references missing prize",
			mutate: func(gs *GenesisState) {
				gs.DenomRewardParticipantIndexList[0].PrizeDenom = "missing"
			},
			expError: "index addr1/udenom1/missing references missing denom reward prize",
		},
		{
			name: "schedule references missing denom reward",
			mutate: func(gs *GenesisState) {
				gs.DenomRewardScheduleList[0].StakingDenom = "missing"
			},
			expError: "schedule missing/000000000001 references missing denom reward",
		},
		{
			name: "schedule id is not numeric",
			mutate: func(gs *GenesisState) {
				gs.DenomRewardScheduleList[0].ScheduleId = "not-a-number"
			},
			expError: "non-numeric schedule id",
		},
		{
			name: "counter behind max schedule id",
			mutate: func(gs *GenesisState) {
				gs.DenomRewardScheduleCounter = 1
			},
			expError: "counter 1 is behind schedule id 000000000002",
		},
		{
			name: "queue cursor references missing schedule",
			mutate: func(gs *GenesisState) {
				gs.DenomRewardsDistributionQueue.Cursor = string(DenomRewardScheduleKey("udenom1", "000000000042"))
			},
			expError: "queue cursor",
		},
		// value-level rules (BZE-142)
		{
			name: "nil numeric fields (absent in JSON) count as zero",
			mutate: func(gs *GenesisState) {
				gs.DenomRewardList[1].StakedAmount = math.Int{}
				gs.DenomRewardPrizeList[1].DistributedStake = math.LegacyDec{}
			},
		},
		{
			name: "more prizes than the current cap is valid (cap only gates new prizes, params may have been lowered)",
			mutate: func(gs *GenesisState) {
				gs.Params.MaxPrizeDenomsPerDr = 1
				gs.DenomRewardPrizeList = append(gs.DenomRewardPrizeList, DenomRewardPrize{StakingDenom: "udenom1", PrizeDenom: "uprizec", DistributedStake: math.LegacyZeroDec()})
			},
		},
		{
			name: "index without a participant record is valid (settled positions may leave markers)",
			mutate: func(gs *GenesisState) {
				gs.DenomRewardParticipantIndexList = append(gs.DenomRewardParticipantIndexList, DenomRewardParticipantIndex{Address: "addr9", StakingDenom: "udenom1", PrizeDenom: "uprizea", Index: math.LegacyMustNewDecFromStr("1.5")})
			},
		},
		{
			name: "negative staked amount",
			mutate: func(gs *GenesisState) {
				gs.DenomRewardList[1].StakedAmount = math.NewInt(-1)
			},
			expError: "denom reward udenom2 has a negative staked amount -1",
		},
		{
			name: "zero staked amount while participants exist",
			mutate: func(gs *GenesisState) {
				gs.DenomRewardList[0].StakedAmount = math.ZeroInt()
			},
			expError: "denom reward udenom1 staked amount 0 does not match the sum of participant amounts 1000",
		},
		{
			name: "staked amount without participants",
			mutate: func(gs *GenesisState) {
				gs.DenomRewardList[1].StakedAmount = math.NewInt(10)
			},
			expError: "denom reward udenom2 staked amount 10 does not match the sum of participant amounts 0",
		},
		{
			name: "participant amounts do not sum to the staked amount",
			mutate: func(gs *GenesisState) {
				gs.DenomRewardParticipantList[1].Amount = math.NewInt(500)
			},
			expError: "denom reward udenom1 staked amount 1000 does not match the sum of participant amounts 900",
		},
		{
			name: "negative participant amount",
			mutate: func(gs *GenesisState) {
				gs.DenomRewardParticipantList[0].Amount = math.NewInt(-400)
			},
			expError: "participant udenom1/addr1 has a non-positive amount -400",
		},
		{
			name: "zero participant amount",
			mutate: func(gs *GenesisState) {
				gs.DenomRewardParticipantList[0].Amount = math.ZeroInt()
			},
			expError: "participant udenom1/addr1 has a non-positive amount 0",
		},
		{
			name: "negative prize accumulator",
			mutate: func(gs *GenesisState) {
				gs.DenomRewardPrizeList[1].DistributedStake = math.LegacyMustNewDecFromStr("-0.1")
			},
			expError: "prize udenom2/uprizeb has a negative accumulator",
		},
		{
			name: "negative participant index",
			mutate: func(gs *GenesisState) {
				gs.DenomRewardParticipantIndexList[0].Index = math.LegacyMustNewDecFromStr("-0.5")
			},
			expError: "index addr1/udenom1/uprizea has a negative index",
		},
		{
			name: "participant index ahead of the prize accumulator",
			mutate: func(gs *GenesisState) {
				gs.DenomRewardParticipantIndexList[0].Index = math.LegacyMustNewDecFromStr("1.500000000000000001")
			},
			expError: "index addr1/udenom1/uprizea is ahead of the prize accumulator",
		},
		{
			name: "schedule references missing prize",
			mutate: func(gs *GenesisState) {
				gs.DenomRewardScheduleList[1].PrizeDenom = "missing"
			},
			expError: "schedule udenom2/000000000002 references missing denom reward prize missing",
		},
		{
			name: "schedule with zero daily amount",
			mutate: func(gs *GenesisState) {
				gs.DenomRewardScheduleList[1].DailyAmount = math.ZeroInt()
			},
			expError: "schedule udenom2/000000000002 has a non-positive daily amount 0",
		},
		{
			name: "schedule with negative daily amount",
			mutate: func(gs *GenesisState) {
				gs.DenomRewardScheduleList[1].DailyAmount = math.NewInt(-50)
			},
			expError: "schedule udenom2/000000000002 has a non-positive daily amount -50",
		},
		{
			name: "schedule with zero duration",
			mutate: func(gs *GenesisState) {
				gs.DenomRewardScheduleList[1].Duration = 0
			},
			expError: "schedule udenom2/000000000002 has zero duration",
		},
		{
			name: "schedule with payouts equal to duration never finishes",
			mutate: func(gs *GenesisState) {
				gs.DenomRewardScheduleList[0].Payouts = 30
			},
			expError: "schedule udenom1/000000000001 has 30 payouts for a duration of 30 and would never finish",
		},
		{
			name: "schedule with payouts above duration never finishes",
			mutate: func(gs *GenesisState) {
				gs.DenomRewardScheduleList[0].Payouts = 31
			},
			expError: "schedule udenom1/000000000001 has 31 payouts for a duration of 30 and would never finish",
		},
		{
			name: "duplicate prize",
			mutate: func(gs *GenesisState) {
				gs.DenomRewardPrizeList = append(gs.DenomRewardPrizeList, gs.DenomRewardPrizeList[0])
			},
			expError: "duplicate denom reward prize udenom1/uprizea",
		},
		{
			name: "duplicate participant",
			mutate: func(gs *GenesisState) {
				gs.DenomRewardParticipantList = append(gs.DenomRewardParticipantList, gs.DenomRewardParticipantList[0])
			},
			expError: "duplicate denom reward participant udenom1/addr1",
		},
		{
			name: "duplicate participant index",
			mutate: func(gs *GenesisState) {
				gs.DenomRewardParticipantIndexList = append(gs.DenomRewardParticipantIndexList, gs.DenomRewardParticipantIndexList[0])
			},
			expError: "duplicate denom reward participant index addr1/udenom1/uprizea",
		},
		{
			name: "duplicate schedule",
			mutate: func(gs *GenesisState) {
				gs.DenomRewardScheduleList = append(gs.DenomRewardScheduleList, gs.DenomRewardScheduleList[0])
			},
			expError: "duplicate denom reward schedule udenom1/000000000001",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gs := validDenomRewardsGenesis()
			tt.mutate(&gs)

			err := gs.Validate()
			if tt.expError == "" {
				require.NoError(t, err)
				return
			}

			require.Error(t, err)
			require.Contains(t, err.Error(), tt.expError)
		})
	}
}
