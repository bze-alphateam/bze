package types

const (
	// ModuleName defines the module name
	ModuleName = "rewards"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName

	// MemStoreKey defines the in-memory store key
	MemStoreKey = "mem_rewards"

	// StakingRewardKeyPrefix is the prefix to retrieve all StakingReward
	StakingRewardKeyPrefix = "sr/value/"
	CounterKey             = "sr/c/"
)

var (
	ParamsKey = []byte("p_rewards")
)

func KeyPrefix(p string) []byte {
	return []byte(p)
}

// StakingRewardKey returns the store key to retrieve a StakingReward from the index fields
func StakingRewardKey(rewardId string) []byte {
	return []byte(rewardId + "/")
}

func StakingRewardCounterKey() []byte {
	return []byte{1}
}
