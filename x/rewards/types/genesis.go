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
		Params:                             DefaultParams(),
		StakingRewardList:                  []StakingReward{},
		StakingRewardsCounter:              0,
		TradingRewardsCounter:              0,
		ActiveTradingRewardList:            []TradingReward{},
		PendingTradingRewardList:           []TradingReward{},
		StakingRewardParticipantList:       []StakingRewardParticipant{},
		PendingUnlockParticipantList:       []PendingUnlockParticipant{},
		TradingRewardLeaderboardList:       []TradingRewardLeaderboard{},
		TradingRewardCandidateList:         []TradingRewardCandidate{},
		MarketIdTradingRewardIdList:        []MarketIdTradingRewardId{},
		PendingTradingRewardExpirationList: []TradingRewardExpiration{},
		ActiveTradingRewardExpirationList:  []TradingRewardExpiration{},
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

	// Check for duplicated index in tradingReward
	tradingRewardIndexMap := make(map[string]struct{})
	for _, elem := range gs.ActiveTradingRewardList {
		index := string(TradingRewardKey(elem.RewardId))
		if _, ok := tradingRewardIndexMap[index]; ok {
			return fmt.Errorf("duplicated index for tradingReward")
		}
		tradingRewardIndexMap[index] = struct{}{}
	}

	pendingTradingRewardListIndexMap := make(map[string]struct{})
	for _, elem := range gs.PendingTradingRewardList {
		index := string(TradingRewardKey(elem.RewardId))
		if _, ok := pendingTradingRewardListIndexMap[index]; ok {
			return fmt.Errorf("duplicated index for PendingTradingRewardList")
		}
		pendingTradingRewardListIndexMap[index] = struct{}{}
	}

	// Check for duplicated index in stakingRewardParticipant
	stakingRewardParticipantIndexMap := make(map[string]struct{})
	for _, elem := range gs.StakingRewardParticipantList {
		index := string(StakingRewardParticipantKey(elem.Address, elem.RewardId))
		if _, ok := stakingRewardParticipantIndexMap[index]; ok {
			return fmt.Errorf("duplicated index for StakingRewardParticipantList")
		}
		stakingRewardParticipantIndexMap[index] = struct{}{}
	}

	pendingUnlockParticipantListIndexMap := make(map[string]struct{})
	for _, elem := range gs.PendingUnlockParticipantList {
		index := string(PendingUnlockParticipantKey(elem.Index))
		if _, ok := pendingUnlockParticipantListIndexMap[index]; ok {
			return fmt.Errorf("duplicated index for PendingUnlockParticipantList")
		}
		pendingUnlockParticipantListIndexMap[index] = struct{}{}
	}

	tradingRewardLeaderboardListIndexMap := make(map[string]struct{})
	for _, elem := range gs.TradingRewardLeaderboardList {
		index := string(TradingRewardKey(elem.RewardId))
		if _, ok := tradingRewardLeaderboardListIndexMap[index]; ok {
			return fmt.Errorf("duplicated index for TradingRewardLeaderboardList")
		}
		tradingRewardLeaderboardListIndexMap[index] = struct{}{}
	}

	tradingRewardCandidateListIndexMap := make(map[string]struct{})
	for _, elem := range gs.TradingRewardCandidateList {
		index := string(TradingRewardCandidateKey(elem.RewardId, elem.Address))
		if _, ok := tradingRewardCandidateListIndexMap[index]; ok {
			return fmt.Errorf("duplicated index for TradingRewardCandidateList")
		}
		tradingRewardCandidateListIndexMap[index] = struct{}{}
	}

	marketIdTradingRewardIdListIndexMap := make(map[string]struct{})
	for _, elem := range gs.MarketIdTradingRewardIdList {
		index := string(MarketIdRewardIdKey(elem.MarketId))
		if _, ok := marketIdTradingRewardIdListIndexMap[index]; ok {
			return fmt.Errorf("duplicated index for MarketIdTradingRewardIdList")
		}
		marketIdTradingRewardIdListIndexMap[index] = struct{}{}
	}

	pendingTradingRewardExpirationListIndexMap := make(map[string]struct{})
	for _, elem := range gs.PendingTradingRewardExpirationList {
		index := string(TradingRewardExpirationKey(elem.ExpireAt, elem.RewardId))
		if _, ok := pendingTradingRewardExpirationListIndexMap[index]; ok {
			return fmt.Errorf("duplicated index for PendingTradingRewardExpirationList")
		}
		pendingTradingRewardExpirationListIndexMap[index] = struct{}{}
	}

	activeTradingRewardExpirationListIndexMap := make(map[string]struct{})
	for _, elem := range gs.ActiveTradingRewardExpirationList {
		index := string(TradingRewardExpirationKey(elem.ExpireAt, elem.RewardId))
		if _, ok := activeTradingRewardExpirationListIndexMap[index]; ok {
			return fmt.Errorf("duplicated index for ActiveTradingRewardExpirationList")
		}
		activeTradingRewardExpirationListIndexMap[index] = struct{}{}
	}

	// this line is used by starport scaffolding # genesis/types/validate

	return gs.Params.Validate()
}
