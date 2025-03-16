package keeper

import "fmt"

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
