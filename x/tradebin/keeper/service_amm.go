package keeper

import (
	"cosmossdk.io/math"
	"fmt"
	burnermoduletypes "github.com/bze-alphateam/bze/x/burner/types"
	"github.com/bze-alphateam/bze/x/tradebin/types"
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

func (k Keeper) swapTokens(ctx sdk.Context, input sdk.Coin, pool *types.LiquidityPool) (output sdk.Coin, err error) {
	if !pool.HasDenom(input.Denom) {
		return output, fmt.Errorf("denom %s does not exist in pool %s", input.Denom, pool.GetId())
	}

	realInput, fee := k.calculateSwapInputAndFee(input, pool)
	if pool.Fee.IsPositive() && !fee.IsPositive() {
		return output, fmt.Errorf("amount is too low to be traded")
	}

	feeToPool, err := k.collectSwapFee(ctx, fee, pool)
	if err != nil {
		return output, err
	}

	inputReserve, outputReserve := pool.GetReservesCoinsByDenom(input.Denom)

	//output_reserve x real_input (the input - fee)
	prod := math.LegacyNewDecFromInt(outputReserve.Amount.Mul(realInput.Amount))

	//input_reserve + real_input (the input - fee)
	quo := math.LegacyNewDecFromInt(inputReserve.Amount.Add(realInput.Amount))
	if !quo.IsPositive() || !prod.IsPositive() {
		return output, fmt.Errorf("non positive product or quotient on swap tokens")
	}

	outputAmount := prod.Quo(quo).TruncateInt()
	output = sdk.NewCoin(outputReserve.Denom, outputAmount)
	if !output.IsPositive() {
		return output, fmt.Errorf("non positive output on swap tokens")
	}

	//add the part of the fee that should remain in the LP (as LP Reward to LP providers)
	err = pool.ChangeReserves(realInput.Add(feeToPool), output)
	if err != nil {
		return output, err
	}

	k.SetLiquidityPool(ctx, *pool)

	return output, nil
}

// collectSwapFee - calculates the distribution of the fee, and it returns the part of the fee that should be added to
// LP (for LP Providers rewards)
func (k Keeper) collectSwapFee(ctx sdk.Context, fee sdk.Coin, pool *types.LiquidityPool) (sdk.Coin, error) {
	if !fee.IsPositive() {
		//return 0 coin
		return sdk.NewCoin(fee.Denom, math.ZeroInt()), nil
	}

	feeDec := math.LegacyNewDecFromInt(fee.Amount)
	treasury := sdk.NewCoin(fee.Denom, feeDec.Mul(pool.FeeDest.Treasury).TruncateInt())
	if treasury.IsPositive() {
		fee = fee.Sub(treasury)
		moduleAcc := k.accountKeeper.GetModuleAccount(ctx, types.ModuleName)
		err := k.distrKeeper.FundCommunityPool(ctx, sdk.NewCoins(treasury), moduleAcc.GetAddress())
		if err != nil {
			return sdk.Coin{}, err
		}

		if !fee.IsPositive() {

			return sdk.NewCoin(fee.Denom, math.ZeroInt()), nil
		}
	}

	burner := sdk.NewCoin(fee.Denom, feeDec.Mul(pool.FeeDest.Burner).TruncateInt())
	if burner.IsPositive() {
		fee = fee.Sub(burner)
		err := k.bankKeeper.SendCoinsFromModuleToModule(ctx, types.ModuleName, burnermoduletypes.ModuleName, sdk.NewCoins(burner))
		if err != nil {
			return sdk.Coin{}, err
		}
	}

	//just to make sure it's never negative (thinking that truncating the Dec might result in 0 or even negative value)
	if !fee.IsPositive() {
		//return 0 coin
		return sdk.NewCoin(fee.Denom, math.ZeroInt()), nil
	}

	return fee, nil
}

func (k Keeper) calculateSwapInputAndFee(input sdk.Coin, pool *types.LiquidityPool) (remainingInput, fee sdk.Coin) {
	feeAmount := math.LegacyNewDecFromInt(input.Amount).Mul(pool.Fee).TruncateInt()
	rAmount := input.Amount.Sub(feeAmount)

	return sdk.NewCoin(input.GetDenom(), rAmount), sdk.NewCoin(input.GetDenom(), feeAmount)
}
