package types

import (
	"fmt"
	_ "github.com/cosmos/cosmos-sdk/x/auth/types"
)

// DefaultIndex is the default capability global index
const DefaultIndex uint64 = 1

// DefaultGenesis returns the default Capability genesis state
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		StakingRewardList:     []StakingReward{},
		StakingRewardsCounter: 0,
		TradingRewardsCounter: 0,
		// this line is used by starport scaffolding # genesis/types/default
		Params: DefaultParams(),
	}
}

// Validate performs basic genesis state validation returning an error upon any
// failure.
func (gs GenesisState) Validate() error {
	// Check for duplicated index in stakingReward
	stakingRewardIndexMap := make(map[string]struct{})

	for _, elem := range gs.StakingRewardList {
		index := string(StakingRewardKey(elem.RewardId))
		if _, ok := stakingRewardIndexMap[index]; ok {
			return fmt.Errorf("duplicated index for stakingReward")
		}
		stakingRewardIndexMap[index] = struct{}{}
	}
	// this line is used by starport scaffolding # genesis/types/validate

	return gs.Params.Validate()
}
