package types

import (
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
			{StakingDenom: "udenom1", Lock: 7, MinStake: 0, StakedAmount: "1000"},
			{StakingDenom: "udenom2", Lock: 7, MinStake: 0, StakedAmount: "0"},
		},
		DenomRewardPrizeList: []DenomRewardPrize{
			{StakingDenom: "udenom1", PrizeDenom: "uprizea", DistributedStake: "1.5", LastDistributionEpoch: 10},
			{StakingDenom: "udenom2", PrizeDenom: "uprizeb", DistributedStake: "0", LastDistributionEpoch: 0},
		},
		DenomRewardParticipantList: []DenomRewardParticipant{
			{Address: "addr1", StakingDenom: "udenom1", Amount: "400"},
			{Address: "addr2", StakingDenom: "udenom1", Amount: "600"},
		},
		DenomRewardParticipantIndexList: []DenomRewardParticipantIndex{
			{Address: "addr1", StakingDenom: "udenom1", PrizeDenom: "uprizea", Index: "0.5"},
		},
		DenomRewardScheduleList: []DenomRewardSchedule{
			{ScheduleId: "000000000001", StakingDenom: "udenom1", PrizeDenom: "uprizea", DailyAmount: "100", Duration: 30, Payouts: 10},
			{ScheduleId: "000000000002", StakingDenom: "udenom2", PrizeDenom: "uprizeb", DailyAmount: "50", Duration: 5},
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
				gs.DenomRewardList = append(gs.DenomRewardList, DenomReward{StakingDenom: "udenom1", StakedAmount: "5"})
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
