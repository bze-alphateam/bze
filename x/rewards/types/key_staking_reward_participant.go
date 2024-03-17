package types

import "encoding/binary"

var _ binary.ByteOrder

const (
	// StakingRewardParticipantKeyPrefix is the prefix to retrieve all StakingRewardParticipant
	StakingRewardParticipantKeyPrefix = "srp/v/"
)

// StakingRewardParticipantKey returns the store key to retrieve a StakingRewardParticipant from the index fields
func StakingRewardParticipantKey(address, rewardId string) []byte {
	return []byte(address + "/" + rewardId)
}

// StakingRewardParticipantPrefix returns the store prefix key to retrieve a StakingRewardParticipant for an address
func StakingRewardParticipantPrefix(address string) []byte {
	return []byte(StakingRewardParticipantKeyPrefix + address + "/")
}
