package types

import "encoding/binary"

var _ binary.ByteOrder

const (
	OrderKeyPrefix        = "Tb/o/"
	UserOrderKeyPrefix    = "Tb/u/"
	PriceOrderKeyPrefix   = "Tb/p/"
	AggOrderKeyPrefix     = "Tb/agg/"
	HistoryOrderKeyPrefix = "Tb/h/"
)

func OrderKey(marketId, orderType, orderId string) []byte {
	return []byte(marketId + "/" + orderType + "/" + orderId + "/")
}

func UserOrderKey(address, marketId, orderType, orderId string) []byte {
	return []byte(address + "/" + marketId + "/" + orderType + "/" + orderId + "/")
}

func PriceOrderKey(marketId, orderType, price, orderId string) []byte {
	return []byte(marketId + "/" + orderType + "/" + price + "/" + orderId + "/")
}

func AggOrderKey(marketId, orderType, price string) []byte {
	return []byte(marketId + "/" + orderType + "/" + price + "/")
}

func HistoryOrderKey(marketId, executedAt, orderId string) []byte {
	return []byte(marketId + "/" + executedAt + "/" + orderId + "/")
}
