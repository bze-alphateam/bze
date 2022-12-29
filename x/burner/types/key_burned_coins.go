package types

import "encoding/binary"

var _ binary.ByteOrder

const (
	// BurnedCoinsKeyPrefix is the prefix to retrieve all BurnedCoins
	BurnedCoinsKeyPrefix = "BurnedCoins/value/"
)

// BurnedCoinsKey returns the store key to retrieve a BurnedCoins from the index fields
func BurnedCoinsKey(
	index string,
) []byte {
	var key []byte

	indexBytes := []byte(index)
	key = append(key, indexBytes...)
	key = append(key, []byte("/")...)

	return key
}
