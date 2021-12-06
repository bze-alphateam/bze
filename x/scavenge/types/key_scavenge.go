package types

import "encoding/binary"

var _ binary.ByteOrder

const (
	// ScavengeKeyPrefix is the prefix to retrieve all Scavenge
	ScavengeKeyPrefix = "Scavenge/value/"
)

// ScavengeKey returns the store key to retrieve a Scavenge from the index fields
func ScavengeKey(
	index string,
) []byte {
	var key []byte

	indexBytes := []byte(index)
	key = append(key, indexBytes...)
	key = append(key, []byte("/")...)

	return key
}
