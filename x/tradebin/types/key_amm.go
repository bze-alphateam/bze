package types

const (
	LpKeyPrefix = "lp/"

	// LpSnapshotKeyPrefix is the prefix for storing LP snapshots
	LpSnapshotKeyPrefix = "lps/"

	// LpModificationQueuePrefix is the prefix for storing modified LP IDs
	LpModificationQueuePrefix = "lpm/"
)

func LpPrefix() []byte {
	return []byte(LpKeyPrefix)
}

func LpSnapshotPrefix() []byte {
	return []byte(LpSnapshotKeyPrefix)
}

func PoolKey(poolId string) []byte {
	return []byte(poolId)
}

// LpModificationQueueKey creates a key for a modified LP ID
func LpModificationQueueKey(poolId string) []byte {
	return []byte(poolId)
}
