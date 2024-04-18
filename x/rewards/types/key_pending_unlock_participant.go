package types

import (
	"encoding/binary"
	"fmt"
)

var _ binary.ByteOrder

const (
	// PendingUnlockParticipantKeyPrefix is the prefix to retrieve all PendingUnlockParticipant
	PendingUnlockParticipantKeyPrefix = "pup/v/"
)

// PendingUnlockParticipantKey returns the store key to retrieve a PendingUnlockParticipant from the index fields
func PendingUnlockParticipantKey(key string) []byte {
	return []byte(key + "/")
}

func CreatePendingUnlockParticipantKey(epoch int64, key string) string {
	return fmt.Sprintf("%d/%s", epoch, key)
}

// PendingUnlockParticipantPrefix returns the store key to retrieve all PendingUnlockParticipant for an epoch
func PendingUnlockParticipantPrefix(epoch int64) string {
	return fmt.Sprintf("%s%d/", PendingUnlockParticipantKeyPrefix, epoch)
}
