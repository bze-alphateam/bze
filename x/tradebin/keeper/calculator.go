package keeper

import (
	"fmt"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func CalculateMinAmount(price string) sdk.Int {
	// Convert the price string to a Dec
	priceDec, err := sdk.NewDecFromStr(price)
	if err != nil {
		fmt.Println("Error converting price to Dec:", err)
		return sdk.NewInt(0)
	}
	if priceDec.IsZero() {
		return sdk.NewInt(0)
	}

	// The denominator for our operation, represented as a Dec
	oneDec := sdk.NewDec(1)

	// Perform the division (1 / price), ensuring high precision
	amtDec := oneDec.Quo(priceDec)
	// Ceil the result to ensure we avoid dust effectively
	amtDec = amtDec.Ceil()

	// Multiply by 1000 to adjust for potential dust and lower loss,
	// as described in your comment.

	//Note: gave up on this due to uncontrollable dust that can be lost during a trade
	//amtDec = amtDec.MulInt64(1000)

	return amtDec.TruncateInt()
}
