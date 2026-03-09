package types

import (
	"encoding/binary"
	"strings"

	"cosmossdk.io/math"
)

var _ binary.ByteOrder

const (
	OrderKeyPrefix        = "Tb/o/"
	UserOrderKeyPrefix    = "Tb/u/"
	PriceOrderKeyPrefix   = "Tb/p/"
	AggOrderKeyPrefix     = "Tb/agg/"
	HistoryOrderKeyPrefix = "Tb/h/"
	OrderCounterPrefix    = "Tb/cnt/"

	orderCounterKey = "ocnt"
)

func OrderKey(marketId, orderType, orderId string) []byte {
	return []byte(marketId + "/" + orderType + "/" + orderId + "/")
}

func UserOrderKey(address, marketId, orderType, orderId string) []byte {
	return []byte(address + "/" + marketId + "/" + orderType + "/" + orderId + "/")
}

func UserOrderByUserPrefix(address string) []byte {
	return []byte(UserOrderKeyPrefix + address + "/")
}

func UserOrderByUserAndMarketPrefix(address, marketId string) []byte {
	return []byte(UserOrderKeyPrefix + address + "/" + marketId + "/")
}

func PriceOrderKey(marketId, orderType, price, orderId string) []byte {
	return []byte(marketId + "/" + orderType + "/" + transformPrice(price) + "/" + orderId + "/")
}

func PriceOrderPrefixKey(marketId, orderType, price string) []byte {
	return []byte(marketId + "/" + orderType + "/" + transformPrice(price) + "/")
}

func AggOrderKey(marketId, orderType, price string) []byte {
	return []byte(marketId + "/" + orderType + "/" + transformPrice(price) + "/")
}

func AggOrderByMarketAndTypePrefix(marketId, orderType string) []byte {
	return []byte(AggOrderKeyPrefix + marketId + "/" + orderType + "/")
}

func HistoryOrderKey(marketId, executedAt, orderId string) []byte {
	return []byte(marketId + "/" + executedAt + "/" + orderId + "/")
}

func HistoryOrderByMarketPrefix(marketId string) []byte {
	return []byte(HistoryOrderKeyPrefix + marketId + "/")
}

func OrderCounterKey() []byte {
	var key []byte

	mBytes := []byte(orderCounterKey)
	key = append(key, mBytes...)
	key = append(key, []byte("/")...)

	return key
}

func transformPrice(price string) string {
	// Parse the price string using Cosmos SDK math to avoid float64 precision issues
	dec, err := math.LegacyNewDecFromStr(price)
	if err != nil {
		return price
	}

	// LegacyDec.String() always returns exactly 18 decimal places
	decStr := dec.String()

	// Split on decimal point - guaranteed to exist with 18 decimal places
	dotIndex := strings.Index(decStr, ".")
	intPart := decStr[:dotIndex]
	fracPart := decStr[dotIndex+1:] // Exactly 18 digits

	// Pad integer part to 13 digits (supports up to 9,999,999,999,999)
	// Total width: 13 + 1 + 18 = 32 characters
	if len(intPart) < 13 {
		intPart = strings.Repeat("0", 13-len(intPart)) + intPart
	}

	return intPart + "." + fracPart
}
