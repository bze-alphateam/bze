package types

import "encoding/binary"

var _ binary.ByteOrder

const (
	// TradingRewardKeyPrefix is the prefix to retrieve all TradingReward
	TradingRewardKeyPrefix = "tr/value/"
)

// TradingRewardKey returns the store key to retrieve a TradingReward from the index fields
func TradingRewardKey(rewardId string) []byte {
	return []byte(rewardId)
}

func TradingRewardCounterKey() []byte {
	return []byte{2}
}
