package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (lp *LiquidityPool) GetReservesCoinsByDenom(denom string) (denomCoin, counterDenom sdk.Coin) {
	if !lp.HasDenom(denom) {
		return denomCoin, counterDenom
	}

	if denom == lp.GetBase() {
		denomCoin = sdk.NewCoin(lp.GetBase(), lp.ReserveBase)
		counterDenom = sdk.NewCoin(lp.GetQuote(), lp.ReserveQuote)
	} else {
		denomCoin = sdk.NewCoin(lp.GetQuote(), lp.ReserveQuote)
		counterDenom = sdk.NewCoin(lp.GetBase(), lp.ReserveBase)
	}

	return denomCoin, counterDenom
}
func (lp *LiquidityPool) ChangeReserves(add, subtract sdk.Coin) error {
	// Validation checks are good
	if add.Denom == subtract.Denom {
		return fmt.Errorf("can not change reserves with amounts of the same denom %s", add.Denom)
	}

	if !lp.HasDenom(add.Denom) || !lp.HasDenom(subtract.Denom) {
		return fmt.Errorf("can not change reserves of pool %s with denoms: %s and %s", lp.GetId(), add.Denom, subtract.Denom)
	}

	if add.Denom == lp.GetBase() {
		// Check if we have enough quote to subtract
		if lp.ReserveQuote.LT(subtract.Amount) {
			return fmt.Errorf("insufficient quote reserve: have %s, need %s", lp.ReserveQuote, subtract.Amount)
		}

		// Add to base reserve
		lp.ReserveBase = lp.ReserveBase.Add(add.Amount)
		// Subtract from quote reserve
		lp.ReserveQuote = lp.ReserveQuote.Sub(subtract.Amount)
	} else {
		// Check if we have enough base to subtract
		if lp.ReserveBase.LT(subtract.Amount) {
			return fmt.Errorf("insufficient base reserve: have %s, need %s", lp.ReserveBase, subtract.Amount)
		}

		// Add to quote reserve
		lp.ReserveQuote = lp.ReserveQuote.Add(add.Amount)
		// Subtract from base reserve
		lp.ReserveBase = lp.ReserveBase.Sub(subtract.Amount)
	}

	return nil
}

func (lp *LiquidityPool) HasDenom(denom string) bool {
	return denom == lp.GetBase() || denom == lp.GetQuote()
}
