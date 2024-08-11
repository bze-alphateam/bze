package types

import "fmt"

const (
	RaffleKeyPrefix        = "rf/v/"
	RaffleDeleteHookPrefix = "rf/dh/"
)

func GetRaffleDeleteHookPrefix(endAt uint64) []byte {
	return []byte(fmt.Sprintf("%s%d/", RaffleDeleteHookPrefix, endAt))
}

func RaffleStoreKey(denom string) []byte {
	return []byte(denom)
}
