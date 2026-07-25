package types

const (
	// BoostKeyPrefix is the prefix to retrieve all Boost records.
	// Full key layout: boost/v/{reward_id}/{denom}/
	BoostKeyPrefix = "boost/v/"

	// BoostParticipantIndexKeyPrefix is the prefix of the reverse index that
	// lists the participants of a reward for the boost finalization sweep.
	// Full key layout: bpi/v/{reward_id}/{address}/
	BoostParticipantIndexKeyPrefix = "bpi/v/"
)

// BoostParticipantIndexValue is the (set-semantics) value stored under a reverse
// index key — presence is all that matters.
var BoostParticipantIndexValue = []byte{1}

// BoostKey returns the store key (within the BoostKeyPrefix store) for a boost.
// reward_id is the fixed-width 12-digit zero-filled id and MUST come first so
// per-reward prefix iteration works and denoms containing "/" (IBC, factory
// denoms) cannot ambiguate parsing.
func BoostKey(rewardId, denom string) []byte {
	return []byte(rewardId + "/" + denom + "/")
}

// BoostRewardPrefix returns the prefix (within the BoostKeyPrefix store) that
// scans every boost of a single reward.
func BoostRewardPrefix(rewardId string) []byte {
	return []byte(rewardId + "/")
}

// BoostParticipantIndexKey returns the store key (within the
// BoostParticipantIndexKeyPrefix store) for one participant of one reward.
func BoostParticipantIndexKey(rewardId, address string) []byte {
	return []byte(rewardId + "/" + address + "/")
}

// BoostParticipantIndexRewardPrefix returns the prefix (within the
// BoostParticipantIndexKeyPrefix store) that scans every participant of a reward.
func BoostParticipantIndexRewardPrefix(rewardId string) []byte {
	return []byte(rewardId + "/")
}

// BoostCounterKey is the counter-store key holding the global monotonic boost
// uid counter. Sits alongside the staking ({1}) and trading ({2}) counter keys.
func BoostCounterKey() []byte {
	return []byte{3}
}
