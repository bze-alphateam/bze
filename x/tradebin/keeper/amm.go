package keeper

import (
	"fmt"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	lpDenomPrefix = "lp"

	sharesScaleExponent = 6
)

// CreatePoolId - orders assets alphabetically and returns them in current order and their respective pool id
func (k Keeper) CreatePoolId(base, quote string) (newBase, newQuote, id string) {
	newBase = base
	newQuote = quote
	if base > quote {
		newBase = quote
		newQuote = base
	}

	return newBase, newQuote, k.getPoolId(newBase, newQuote)
}

// getPoolId - creates Pool ID from given assets
func (k Keeper) getPoolId(base, quote string) string {
	return fmt.Sprintf("%s_%s", base, quote)
}

func (k Keeper) getPoolDenom(poolId string) string {
	return fmt.Sprintf("u%s", k.getPoolScaledDenom(poolId))
}

func (k Keeper) getPoolScaledDenom(poolId string) string {
	return fmt.Sprintf("%s_%s", lpDenomPrefix, poolId)
}

func (k Keeper) balanceProvidedAmounts(base, quote, reserveBase, reserveQuote uint64) (sdk.Int, sdk.Int, error) {
	poolBaseReserve := sdk.NewIntFromUint64(reserveBase)
	poolQuoteReserve := sdk.NewIntFromUint64(reserveQuote)
	if poolBaseReserve.IsZero() || poolQuoteReserve.IsZero() {
		//pools should not be empty, they are created with a desired price
		return sdk.ZeroInt(), sdk.ZeroInt(), fmt.Errorf("pool is empty")
	}

	desiredBase := sdk.NewIntFromUint64(base)
	desiredQuote := sdk.NewIntFromUint64(quote)

	// Calculate how much would be needed for the provided amounts
	possibleQuote := desiredBase.Mul(poolQuoteReserve).Quo(poolBaseReserve)
	possibleBase := desiredQuote.Mul(poolBaseReserve).Quo(poolQuoteReserve)

	var optimalBase, optimalQuote sdk.Int
	// Use the lesser amounts to maintain the ratio
	if possibleQuote.LTE(desiredQuote) {
		optimalBase = desiredBase
		optimalQuote = possibleQuote
	} else {
		optimalBase = possibleBase
		optimalQuote = desiredQuote
	}

	return optimalBase, optimalQuote, nil
}
