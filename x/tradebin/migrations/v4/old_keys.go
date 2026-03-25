package v4

import (
	"fmt"
	"strconv"
)

// transformPriceOld is the old implementation using float64
// This is kept only for migration purposes to construct old keys
func transformPriceOld(price string) string {
	floatVal, err := strconv.ParseFloat(price, 64)
	if err != nil {
		return price
	}

	// Format the float back into a string with zero padding to ensure it's 24 characters long
	// Adjust the precision as needed
	return fmt.Sprintf("%024.10f", floatVal)
}

// oldPriceOrderKey constructs the old price order key format
// Used only during migration to delete old keys
func oldPriceOrderKey(marketId, orderType, price, orderId string) []byte {
	return []byte(marketId + "/" + orderType + "/" + transformPriceOld(price) + "/" + orderId + "/")
}

// oldAggOrderKey constructs the old aggregated order key format
// Used only during migration to delete old keys
func oldAggOrderKey(marketId, orderType, price string) []byte {
	return []byte(marketId + "/" + orderType + "/" + transformPriceOld(price) + "/")
}
