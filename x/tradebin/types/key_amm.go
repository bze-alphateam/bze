package types

const (
	LpKeyPrefix = "lp/"
)

func LpPrefix() []byte {
	return []byte(LpKeyPrefix)
}

func PoolKey(poolId string) []byte {
	return []byte(poolId)
}
