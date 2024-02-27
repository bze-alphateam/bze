package types

import "encoding/binary"

var _ binary.ByteOrder

const (
	// MarketKeyPrefix is the prefix to retrieve all Market
	MarketKeyPrefix      = "value/market/"
	MarketAliasKeyPrefix = "alias/market/"
)

// MarketKey returns the store key to retrieve a Market from the index fields
func MarketKey(
	base string,
	quote string,
) []byte {
	key := MarketAssetKey(base)

	asset2Bytes := MarketAssetKey(quote)
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
