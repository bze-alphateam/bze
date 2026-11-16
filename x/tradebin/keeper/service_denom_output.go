package keeper

import (
	"cosmossdk.io/math"
	"github.com/bze-alphateam/bze/x/tradebin/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// expectedSwapOutputAmount returns the amount swapTokens would output for input on the pool's current reserves,
// using exactly the same arithmetic: the pool fee is taken from the input, then
// output = output_reserve x real_input / (input_reserve + real_input), truncated.
// It returns zero whenever swapTokens would refuse or overflow on the product or quotient, so a positive result means the swap
// produces at least one unit of the output denom. It doesn't change the pool.
func (k Keeper) expectedSwapOutputAmount(input sdk.Coin, pool *types.LiquidityPool) math.Int {
	if !pool.HasDenom(input.Denom) {
		return math.ZeroInt()
	}

	realInput, _ := k.calculateSwapInputAndFee(input, pool)
	inputReserve, outputReserve := pool.GetReservesCoinsByDenom(input.Denom)

	// checked arithmetic: this runs in EndBlock fee/burn passes, where an amount too large for the 256-bit product
	// must count as "not swappable" instead of panicking. swapTokens can't do that swap either.
	prodInt, err := outputReserve.Amount.SafeMul(realInput.Amount)
	if err != nil {
		return math.ZeroInt()
	}
	quoInt, err := inputReserve.Amount.SafeAdd(realInput.Amount)
	if err != nil {
		return math.ZeroInt()
	}

	prod := math.LegacyNewDecFromInt(prodInt)
	quo := math.LegacyNewDecFromInt(quoInt)
	if !quo.IsPositive() || !prod.IsPositive() {
		return math.ZeroInt()
	}

	return prod.Quo(quo).TruncateInt()
}
