package keeper

import (
	"math"
	"strconv"
)

func CalculateMinAmount(price string) int64 {
	priceFloat, err := strconv.ParseFloat(price, 64)
	if err != nil {
		return 0
	}

	amtFloat := math.Ceil(1 / priceFloat)

	return int64(amtFloat)
}
