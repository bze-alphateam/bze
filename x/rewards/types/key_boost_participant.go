package types

const (
	// BoostParticipantKeyPrefix is the prefix to retrieve all BoostParticipant
	BoostParticipantKeyPrefix = "brp/v/"
)

// BoostParticipantKey returns the store key to retrieve a BoostParticipant from the index fields
func BoostParticipantKey(address, rewardId, boostId string) []byte {
	return []byte(address + "/" + rewardId + "/" + boostId + "/")
}

// BoostParticipantByRewardPrefix returns the store prefix holding all of an
// address' participant entries for one reward (used by exit's settle+delete)
func BoostParticipantByRewardPrefix(address, rewardId string) []byte {
	return []byte(address + "/" + rewardId + "/")
}
