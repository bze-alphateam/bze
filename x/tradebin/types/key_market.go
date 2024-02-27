package types

import "encoding/binary"

var _ binary.ByteOrder

const (
	// MarketKeyPrefix is the prefix to retrieve all Market
	MarketKeyPrefix      = "Market/value/"
	MarketAliasKeyPrefix = "Market/alias/"
)

// MarketKey returns the store key to retrieve a Market from the index fields
func MarketKey(
	asset1 string,
	asset2 string,
) []byte {
	key := MarketAssetKey(asset1)

	asset2Bytes := MarketAssetKey(asset2)
	key = append(key, asset2Bytes...)

	return key
}

func MarketAssetKey(
	asset string,
) []byte {
	var key []byte

	asset1Bytes := []byte(asset)
	key = append(key, asset1Bytes...)
	key = append(key, []byte("/")...)

	return key
}
