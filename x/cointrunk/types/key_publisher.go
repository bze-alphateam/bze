package types

import "encoding/binary"

var _ binary.ByteOrder

const (
	// PublisherKeyPrefix is the prefix to retrieve all Publisher
	PublisherKeyPrefix = "Publisher/value/"
)

// PublisherKey returns the store key to retrieve a Publisher from the index fields
func PublisherKey(
	index string,
) []byte {
	var key []byte

	indexBytes := []byte(index)
	key = append(key, indexBytes...)
	key = append(key, []byte("/")...)

	return key
}
