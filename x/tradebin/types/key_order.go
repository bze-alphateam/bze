package types

import "encoding/binary"

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
	return []byte(marketId + "/" + orderType + "/" + price + "/" + orderId + "/")
}

func PriceOrderPrefixKey(marketId, orderType, price string) []byte {
	return []byte(marketId + "/" + orderType + "/" + price + "/")
}

func AggOrderKey(marketId, orderType, price string) []byte {
	return []byte(marketId + "/" + orderType + "/" + price + "/")
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
