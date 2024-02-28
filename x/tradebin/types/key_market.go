package types

import (
	"encoding/binary"
)

var _ binary.ByteOrder

const (
	// MarketKeyPrefix is the prefix to retrieve all Market
	MarketKeyPrefix      = "Market/value/"
	MarketAliasKeyPrefix = "Market/alias/"

	marketKeyAssetSeparator = "/"
)

// MarketKey returns the store key to retrieve a Market from the index fields
func MarketKey(base string, quote string) []byte {
	return MarketIdKey(CreateMarketId(base, quote))
}

func MarketAssetKey(asset string) []byte {
	return []byte(asset + marketKeyAssetSeparator)
}

func MarketIdKey(marketId string) []byte {
	return []byte(marketId)
}

func CreateMarketId(base, quote string) string {
	return base + marketKeyAssetSeparator + quote
}
