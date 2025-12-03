package types

import "encoding/binary"

var _ binary.ByteOrder

const (
	// QueueMessagePrefix is the prefix to retrieve all QueueMessage
	QueueMessagePrefix        = "Qm/value/"
	QueueMessageCounterPrefix = "Qm/counter/"

	counterKey = "cnt"
)

// QueueMessageKey creates a composite key with market ID and message ID
// Format: {market-id}/{zero-filled-id}/
// This allows efficient lookups by market while maintaining temporal order within each market
func QueueMessageKey(marketId, messageId string) []byte {
	var key []byte

	key = append(key, []byte(marketId)...)
	key = append(key, []byte("/")...)
	key = append(key, []byte(messageId)...)
	key = append(key, []byte("/")...)

	return key
}

// QueueMessageMarketPrefix creates a prefix for querying all messages in a market
// Format: {market-id}/
func QueueMessageMarketPrefix(marketId string) []byte {
	var key []byte

	key = append(key, []byte(marketId)...)
	key = append(key, []byte("/")...)

	return key
}

func QueueMessageCounterKey() []byte {
	var key []byte

	mBytes := []byte(counterKey)
	key = append(key, mBytes...)
	key = append(key, []byte("/")...)

	return key
}
