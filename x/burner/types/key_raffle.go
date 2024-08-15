package types

import "fmt"

const (
	RaffleKeyPrefix         = "rf/v/"
	RaffleDeleteHookPrefix  = "rf/dh/"
	RaffleWinnerKeyPrefix   = "rf/w/"
	RaffleParticipantPrefix = "rf/p/"

	RaffleParticipantCounterPrefix = "rf/c/"
	RaffleParticipantCounterKey    = "cnt"
)

func GetRaffleWinnerKeyPrefix(denom string) []byte {
	return []byte(fmt.Sprintf("%s%s/", RaffleWinnerKeyPrefix, denom))
}

func GetRaffleDeleteHookPrefix(endAt uint64) []byte {
	return []byte(fmt.Sprintf("%s%d/", RaffleDeleteHookPrefix, endAt))
}

func RaffleStoreKey(denom string) []byte {
	return []byte(denom)
}

func GetRaffleParticipantPrefixedKey(prefix int64) []byte {
	return []byte(fmt.Sprintf("%s%d/", RaffleParticipantPrefix, prefix))
}

func GetRaffleParticipantKey(index uint64) []byte {
	return []byte(fmt.Sprintf("%d", index))
}

func GetRaffleParticipantCounterKey() []byte {
	var key []byte

	mBytes := []byte(RaffleParticipantCounterKey)
	key = append(key, mBytes...)
	key = append(key, []byte("/")...)

	return key
}
