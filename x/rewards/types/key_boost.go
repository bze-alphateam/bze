package types

const (
	// BoostKeyPrefix is the prefix to retrieve all Boost
	BoostKeyPrefix = "boost/v/"
)

// BoostKey returns the store key to retrieve a Boost from the index fields.
// Both ids are fixed-width zero-filled, so keys parse unambiguously.
func BoostKey(rewardId, boostId string) []byte {
	return []byte(rewardId + "/" + boostId + "/")
}

// BoostByRewardPrefix returns the store prefix holding all boosts of a reward
func BoostByRewardPrefix(rewardId string) []byte {
	return []byte(rewardId + "/")
}

func BoostCounterKey() []byte {
	return []byte{3}
}
