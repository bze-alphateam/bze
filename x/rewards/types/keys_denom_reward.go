package types

// Denom Rewards store keys. DR is parallel machinery to the Staking Rewards
// (SR) code and never shares its stores. All key funcs that return a suffix are
// meant to be used against a store already prefixed with the corresponding
// KeyPrefix (see keeper/store_denom_reward*.go); the *Prefix helpers return the
// full path and are used for sub-prefix iteration.

// DenomRewardKeySeparator joins and terminates the dynamic segments of composite DR keys.
// It must be a byte that can never appear inside a segment: valid denoms match
// [a-zA-Z][a-zA-Z0-9/:._-]{2,127} — note "/" IS legal in denoms (ibc/HASH,
// factory/creator/sub) — bech32 addresses are lowercase alphanumeric and schedule ids are
// numeric, so 0x00 is safe. Joining with "/" instead would make distinct pairs collide
// (("a","b/c") and ("a/b","c") produce the same key) and would make one denom's prefix scan
// bleed into any denom that extends it ("factory" vs "factory/creator/sub").
const DenomRewardKeySeparator = "\x00"

const (
	// DenomRewardKeyPrefix is the prefix to retrieve a DenomReward by staking denom.
	DenomRewardKeyPrefix = "dr/v/"
	// DenomRewardPrizeKeyPrefix is the prefix to retrieve a DenomRewardPrize (accumulator).
	DenomRewardPrizeKeyPrefix = "dra/v/"
	// DenomRewardParticipantKeyPrefix is the prefix for DenomRewardParticipant records.
	// Keys are denom-first (staking denom, then address) so a DR's participants can be
	// prefix-scanned (and swept in the follow-up story) without a reverse index.
	DenomRewardParticipantKeyPrefix = "drp/v/"
	// DenomRewardParticipantMarkerKeyPrefix is the address-first marker index used to
	// list a user's participations. Values are empty; the denom lives in the key.
	DenomRewardParticipantMarkerKeyPrefix = "drp/a/"
	// DenomRewardParticipantIndexKeyPrefix is the prefix for DenomRewardParticipantIndex
	// records, address-first so a single user's pool is prefix-scanned on settle.
	DenomRewardParticipantIndexKeyPrefix = "dri/v/"
	// DenomRewardScheduleKeyPrefix is the prefix for DenomRewardSchedule records.
	DenomRewardScheduleKeyPrefix = "drs/v/"
	// DenomRewardsDistributionQueueKey is the singleton key for the DR distribution queue.
	DenomRewardsDistributionQueueKey = "drs/dq/"
	// DenomRewardCounterKey is the prefix for the DR schedule counter store. It is
	// separate from the SR/trading counter store (CounterKey) so the counters never
	// collide even though both use the sub-key []byte{1}.
	DenomRewardCounterKey = "dr/c/"
)

// DenomRewardKey returns the store key (suffix) to retrieve a DenomReward from its staking denom.
func DenomRewardKey(stakingDenom string) []byte {
	return []byte(stakingDenom + DenomRewardKeySeparator)
}

// DenomRewardPrizeKey returns the store key (suffix) to retrieve a DenomRewardPrize.
func DenomRewardPrizeKey(stakingDenom, prizeDenom string) []byte {
	return []byte(stakingDenom + DenomRewardKeySeparator + prizeDenom + DenomRewardKeySeparator)
}

// DenomRewardPrizePrefix returns the full prefix to iterate every prize of a staking denom.
func DenomRewardPrizePrefix(stakingDenom string) []byte {
	return []byte(DenomRewardPrizeKeyPrefix + stakingDenom + DenomRewardKeySeparator)
}

// DenomRewardParticipantKey returns the denom-first store key (suffix) of a DenomRewardParticipant.
func DenomRewardParticipantKey(stakingDenom, address string) []byte {
	return []byte(stakingDenom + DenomRewardKeySeparator + address + DenomRewardKeySeparator)
}

// DenomRewardParticipantPrefix returns the full prefix to iterate every participant of a staking denom.
func DenomRewardParticipantPrefix(stakingDenom string) []byte {
	return []byte(DenomRewardParticipantKeyPrefix + stakingDenom + DenomRewardKeySeparator)
}

// DenomRewardParticipantMarkerKey returns the address-first marker store key (suffix).
func DenomRewardParticipantMarkerKey(address, stakingDenom string) []byte {
	return []byte(address + DenomRewardKeySeparator + stakingDenom + DenomRewardKeySeparator)
}

// DenomRewardParticipantMarkerPrefix returns the full prefix to iterate every participation of an address.
func DenomRewardParticipantMarkerPrefix(address string) []byte {
	return []byte(DenomRewardParticipantMarkerKeyPrefix + address + DenomRewardKeySeparator)
}

// DenomRewardParticipantIndexKey returns the store key (suffix) of a DenomRewardParticipantIndex.
func DenomRewardParticipantIndexKey(address, stakingDenom, prizeDenom string) []byte {
	return []byte(address + DenomRewardKeySeparator + stakingDenom + DenomRewardKeySeparator + prizeDenom + DenomRewardKeySeparator)
}

// DenomRewardParticipantIndexPrefix returns the full prefix to iterate every index of a
// (address, staking_denom) pair — i.e. one participant's indexes across all prize denoms.
func DenomRewardParticipantIndexPrefix(address, stakingDenom string) []byte {
	return []byte(DenomRewardParticipantIndexKeyPrefix + address + DenomRewardKeySeparator + stakingDenom + DenomRewardKeySeparator)
}

// DenomRewardScheduleKey returns the store key (suffix) of a DenomRewardSchedule.
func DenomRewardScheduleKey(stakingDenom, scheduleId string) []byte {
	return []byte(stakingDenom + DenomRewardKeySeparator + scheduleId + DenomRewardKeySeparator)
}

// DenomRewardSchedulePrefix returns the full prefix to iterate every schedule of a staking denom.
func DenomRewardSchedulePrefix(stakingDenom string) []byte {
	return []byte(DenomRewardScheduleKeyPrefix + stakingDenom + DenomRewardKeySeparator)
}

// DenomRewardScheduleCounterKey is the sub-key of the schedule counter inside the
// DenomRewardCounterKey store. Mirrors StakingRewardCounterKey()'s []byte{1}, but the
// distinct DenomRewardCounterKey prefix keeps it independent from the SR/trading counters.
func DenomRewardScheduleCounterKey() []byte {
	return []byte{1}
}
