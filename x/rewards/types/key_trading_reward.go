package types

import "encoding/binary"

var _ binary.ByteOrder

const (
	// TradingRewardKeyPrefix is the prefix to retrieve all TradingReward
	TradingRewardKeyPrefix = "tr/value/"
	// MarketIdRewardIdKeyPrefix is the prefix to retrieve a reward id for a market id
	MarketIdRewardIdKeyPrefix = "tr/mr/"
	// LeaderboardKeyPrefix is the prefix of a leaderboard for a trading reward
	LeaderboardKeyPrefix = "tr/lb/"
	// RewardCandidateKeyPrefix prefix that holds entries/participants for trading rewards
	RewardCandidateKeyPrefix = "tr/r/"
)

// TradingRewardCandidateKey returns the store key to retrieve a reward candidate
func TradingRewardCandidateKey(rewardId, address string) []byte {
	return []byte(rewardId + "/" + address)
}

// MarketIdRewardIdKey returns the store key to retrieve a TradingReward.RewardId from the index fields
func MarketIdRewardIdKey(marketId string) []byte {
	return []byte(marketId)
}

// TradingRewardKey returns the store key to retrieve a TradingReward from the index fields
func TradingRewardKey(rewardId string) []byte {
	return []byte(rewardId)
}

func TradingRewardCounterKey() []byte {
	return []byte{2}
}
