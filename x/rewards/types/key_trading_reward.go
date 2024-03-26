package types

import (
	"encoding/binary"
	"fmt"
)

var _ binary.ByteOrder

const (
	// PendingTradingRewardKeyPrefix is the prefix to retrieve all pending TradingReward
	PendingTradingRewardKeyPrefix = "tr/v/p/"
	// ActiveTradingRewardKeyPrefix is the prefix to retrieve all active TradingReward
	ActiveTradingRewardKeyPrefix = "tr/v/a/"
	// MarketIdRewardIdKeyPrefix is the prefix to retrieve a reward id for a market id
	MarketIdRewardIdKeyPrefix = "tr/mr/"
	// LeaderboardKeyPrefix is the prefix of a leaderboard for a trading reward
	LeaderboardKeyPrefix = "tr/lb/"
	// RewardCandidateKeyPrefix prefix that holds entries/participants for trading rewards
	RewardCandidateKeyPrefix = "tr/r/"
	// PendingTradingRewardExpirationKeyPrefix - the prefix used to save trading reward expiration
	PendingTradingRewardExpirationKeyPrefix = "tr/exp/p/"
	ActiveTradingRewardExpirationKeyPrefix  = "tr/exp/a/"
)

// TradingRewardCandidateKey returns the store key to retrieve a reward candidate
func TradingRewardCandidateKey(rewardId, address string) []byte {
	return []byte(rewardId + "/" + address + "/")
}

// MarketIdRewardIdKey returns the store key to retrieve a TradingReward.RewardId from the index fields
func MarketIdRewardIdKey(marketId string) []byte {
	return []byte(marketId + "/")
}

// TradingRewardKey returns the store key to retrieve a TradingReward from the index fields
func TradingRewardKey(rewardId string) []byte {
	return []byte(rewardId + "/")
}

func TradingRewardExpirationKey(expireAt uint32, rewardId string) []byte {
	return []byte(TradingRewardExpirationByExpireAtPrefix(expireAt) + rewardId + "/")
}

func TradingRewardExpirationByExpireAtPrefix(expireAt uint32) string {
	return fmt.Sprintf("%d/", expireAt)
}

func TradingRewardCounterKey() []byte {
	return []byte{2}
}
