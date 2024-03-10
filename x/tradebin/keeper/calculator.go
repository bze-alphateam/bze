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

	//multiply with 1k it to make sure we avoid dust
	//users might suffer loss when trading coins sold for very low prices
	//that's why we make sure we multiply by 1k to lower that loss
	amtFloat := math.Ceil(1/priceFloat) * 1000

	return int64(amtFloat)
}
