package types

const (
	// StakingRewardParticipantIndexKeyPrefix is the prefix of the reverse index
	// over staking reward participants. The participant store itself is keyed
	// address-first (`srp/v/{address}/{rewardId}/`), so enumerating one
	// reward's participants — required by the boost cleanup sweep — needs this
	// reward-first index.
	StakingRewardParticipantIndexKeyPrefix = "srpi/v/"
)

// StakingRewardParticipantIndexValue is the value stored for every index
// entry: the index has set semantics, presence is all that matters.
var StakingRewardParticipantIndexValue = []byte{1}

// StakingRewardParticipantIndexKey returns the store key of one index entry
func StakingRewardParticipantIndexKey(rewardId, address string) []byte {
	return []byte(rewardId + "/" + address + "/")
}

// StakingRewardParticipantIndexRewardPrefix returns the store prefix holding
// all of one reward's index entries
func StakingRewardParticipantIndexRewardPrefix(rewardId string) []byte {
	return []byte(StakingRewardParticipantIndexKeyPrefix + rewardId + "/")
}
