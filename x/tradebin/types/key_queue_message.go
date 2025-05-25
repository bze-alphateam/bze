package types

import "encoding/binary"

var _ binary.ByteOrder

const (
	// QueueMessagePrefix is the prefix to retrieve all QueueMessage
	QueueMessagePrefix        = "Qm/value/"
	QueueMessageCounterPrefix = "Qm/counter/"

	counterKey = "cnt"
)

func QueueMessageKey(
	messageId string,
) []byte {
	var key []byte

	mBytes := []byte(messageId)
	key = append(key, mBytes...)
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
