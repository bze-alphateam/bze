package types

import "encoding/binary"

var _ binary.ByteOrder

const (
	// StakingRewardKeyPrefix is the prefix to retrieve all StakingReward
	StakingRewardKeyPrefix = "sr/value/"
	CounterKey             = "sr/c/"
)

// StakingRewardKey returns the store key to retrieve a StakingReward from the index fields
func StakingRewardKey(rewardId string) []byte {
	return []byte(rewardId)
}

func StakingRewardCounterKey() []byte {
	return []byte{1}
}
