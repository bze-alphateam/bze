package keeper

import (
	"cosmossdk.io/math"
	"fmt"
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

func (k Keeper) BalanceProvidedAmounts(base, quote, reserveBase, reserveQuote math.Int) (math.Int, math.Int, error) {
	if base.IsNil() || quote.IsNil() {
		return math.ZeroInt(), math.ZeroInt(), fmt.Errorf("can not balance with non positive base or quote")
	}

	if reserveBase.IsZero() || reserveQuote.IsZero() {
		//pools should not be empty, they are created with a desired price
		return math.ZeroInt(), math.ZeroInt(), fmt.Errorf("pool is empty")
	}

	// Calculate how much would be needed for the provided amounts
	possibleQuote := base.Mul(reserveQuote).Quo(reserveBase)
	possibleBase := quote.Mul(reserveBase).Quo(reserveQuote)

	var optimalBase, optimalQuote math.Int
	// Use the lesser amounts to maintain the ratio
	if possibleQuote.LTE(quote) {
		optimalBase = base
		optimalQuote = possibleQuote
	} else {
		optimalBase = possibleBase
		optimalQuote = quote
	}

	return optimalBase, optimalQuote, nil
}
