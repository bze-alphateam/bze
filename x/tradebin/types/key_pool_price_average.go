package types

const (
	// PoolPriceAverageKeyPrefix is the prefix under which the moving average of every pool's price is stored,
	// one record per pool id. It is not a prefix of any other module key, and no other module key is a prefix of it.
	PoolPriceAverageKeyPrefix = "pool_price_avg/"
)

func PoolPriceAveragePrefix() []byte {
	return []byte(PoolPriceAverageKeyPrefix)
}

func PoolPriceAverageKey(poolId string) []byte {
	return []byte(poolId)
}
