package types

import "encoding/binary"

var _ binary.ByteOrder

const (
	// AcceptedDomainKeyPrefix is the prefix to retrieve all AcceptedDomain
	AcceptedDomainKeyPrefix = "AcceptedDomain/value/"
)

// AcceptedDomainKey returns the store key to retrieve an AcceptedDomain from the index fields
func AcceptedDomainKey(
	index string,
) []byte {
	var key []byte

	indexBytes := []byte(index)
	key = append(key, indexBytes...)
	key = append(key, []byte("/")...)

	return key
}
