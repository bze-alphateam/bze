package types_test

import (
	"testing"

	"github.com/bze-alphateam/bze/x/rewards/types"
	"github.com/stretchr/testify/require"
)

func TestGenesisState_Validate(t *testing.T) {
	for _, tc := range []struct {
		desc     string
		genState *types.GenesisState
		valid    bool
	}{
		{
			desc:     "default is valid",
			genState: types.DefaultGenesis(),
			valid:    true,
		},
		{
			desc: "valid genesis state",
			genState: &types.GenesisState{

				StakingRewardList: []types.StakingReward{
					{
						RewardId: "0",
					},
					{
						RewardId: "1",
					},
				},
				TradingRewardList: []types.TradingReward{
					{
						RewardId: "0",
					},
					{
						RewardId: "1",
					},
				},
				StakingRewardParticipantList: []types.StakingRewardParticipant{
					{
						Index: "0",
					},
					{
						Index: "1",
					},
				},
				// this line is used by starport scaffolding # types/genesis/validField
			},
			valid: true,
		},
		{
			desc: "duplicated stakingReward",
			genState: &types.GenesisState{
				StakingRewardList: []types.StakingReward{
					{
						RewardId: "0",
					},
					{
						RewardId: "0",
					},
				},
			},
			valid: false,
		},
		{
			desc: "duplicated tradingReward",
			genState: &types.GenesisState{
				TradingRewardList: []types.TradingReward{
					{
						RewardId: "0",
					},
					{
						RewardId: "0",
					},
				},
			},
			valid: false,
		},
		{
			desc: "duplicated stakingRewardParticipant",
			genState: &types.GenesisState{
				StakingRewardParticipantList: []types.StakingRewardParticipant{
					{
						Index: "0",
					},
					{
						Index: "0",
					},
				},
			},
			valid: false,
		},
		// this line is used by starport scaffolding # types/genesis/testcase
	} {
		t.Run(tc.desc, func(t *testing.T) {
			err := tc.genState.Validate()
			if tc.valid {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}
